// Package tui is the r2d2 slack screen.
package tui

import (
	"fmt"
	"time"

	"github.com/ryangerardwilson/c3po/internal/tuijob"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;97m"
	styleMuted  = "\x1b[90m"
	styleError  = "\x1b[7m"
	styleReset  = "\x1b[0m"
)

// syncEvery is how often an open screen asks the desk to refresh.
var syncEvery = 10 * time.Minute

// helpHold is the key wait while the shortcut list is open.
// A refresh that comes due during that list waits until the list closes.
var helpHold = 24 * time.Hour

var now = time.Now

// jobPoll is how often the loop looks for a finished job while one runs.
var jobPoll = 50 * time.Millisecond

// leaveWait bounds the wait for pending marks when the screen leaves. The
// desk commits locally, so this is milliseconds unless the desk is stuck.
var leaveWait = 10 * time.Second

type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

type keyWaiter interface {
	readUntil(deadline time.Time) (key string, ok bool, err error)
}

// model is the screen's UI state. board is a cache of the desk's last
// answer and is only ever replaced whole. marking holds the reads the desk
// has not confirmed yet; the table is the cache minus those rows.
type model struct {
	board   Board
	marking map[string]bool
	syncing bool
	opening bool
	leave   bool
	ws      int
	cursor  int
	status  string
	err     error
	help    bool
	width   int
	height  int
	session *tuisession.Session
	jobs    *tuijob.Queue
}

func newModel(board Board, sess *tuisession.Session) *model {
	return &model{board: board, session: sess, marking: map[string]bool{}}
}

// loop paints, then waits for a key or a finished job. A key only changes
// UI state and may queue a job; no key waits on the desk before the next
// paint.
func (m *model) loop(term screen) error {
	m.restore()
	m.jobs = tuijob.New()
	defer m.jobs.Close()
	next := now().Add(syncEvery)
	for {
		m.width, m.height = term.size()
		term.paint(m.render())
		key, ok, err := m.waitKey(term, next)
		if err != nil {
			return err
		}
		m.jobs.Apply()
		if m.leave {
			m.noteSession()
			return nil
		}
		if !ok {
			if m.help || m.jobs.Busy() || now().Before(next) {
				continue
			}
			m.sync()
			next = now().Add(syncEvery)
			continue
		}
		manual := key == "s" && !m.help
		if m.handleKey(key) {
			m.drain(term)
			m.noteSession()
			return nil
		}
		if manual {
			next = now().Add(syncEvery)
		}
		m.noteSession()
	}
}

func (m *model) readDeadline(next time.Time) time.Time {
	if m.help {
		return now().Add(helpHold)
	}
	return next
}

// waitKey waits for a key until the refresh is due. While a job runs it
// wakes every jobPoll so the result is painted when it lands. A screen that
// cannot wait with a deadline lets the job finish before it reads a key.
func (m *model) waitKey(term screen, next time.Time) (string, bool, error) {
	waiter, ok := term.(keyWaiter)
	if !ok {
		if m.jobs.Busy() {
			m.jobs.Wait(leaveWait)
			return "", false, nil
		}
		key, err := term.readKey()
		if err != nil {
			return "", false, err
		}
		return key, true, nil
	}
	if m.jobs.Busy() {
		return waiter.readUntil(now().Add(jobPoll))
	}
	return waiter.readUntil(m.readDeadline(next))
}

// drain lets pending marks reach the desk before the screen leaves. It
// waits for local commits only; the screen never waits on a backup.
func (m *model) drain(term screen) {
	if len(m.marking) == 0 {
		return
	}
	m.status = fmt.Sprintf("saving %d", len(m.marking))
	m.err = nil
	term.paint(m.render())
	m.jobs.Wait(leaveWait)
	m.jobs.Apply()
}

func (m *model) handleKey(key string) bool {
	if m.help {
		return m.helpKey(key)
	}
	switch key {
	case "?":
		m.help = true
		m.status = ""
		m.err = nil
	case "ctrl+c", "q", "esc":
		return true
	case "j", "down":
		m.move(1)
	case "k", "up":
		m.move(-1)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = m.count() - 1
		m.clamp()
	case "h", "left":
		m.switchPreset(-1)
	case "l", "right":
		m.switchPreset(1)
	case "enter":
		return m.open()
	case "r":
		m.mark()
	case "s":
		m.sync()
	}
	return false
}

func (m *model) helpKey(key string) bool {
	switch key {
	case "?", "esc":
		m.help = false
	case "ctrl+c", "q":
		return true
	}
	return false
}

func (m *model) move(delta int) {
	n := m.count()
	if n == 0 {
		return
	}
	m.cursor = cycled(m.clamp(), delta, n)
}

func cycled(cursor, delta, n int) int {
	if n <= 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	} else if cursor >= n {
		cursor = n - 1
	}
	return (cursor + delta%n + n) % n
}

func (m *model) switchPreset(delta int) {
	n := len(m.board.Workspaces)
	if n == 0 {
		return
	}
	m.ws = (m.ws + delta + n) % n
	m.cursor = 0
	m.status = ""
	m.err = nil
}

// open asks the desk to open the row. The screen leaves once the desk says
// it opened.
func (m *model) open() bool {
	item, ok := m.current()
	if !ok || m.opening {
		return false
	}
	preset := m.presetID()
	m.opening = true
	m.status = "opening"
	m.err = nil
	m.jobs.Go(func() func() {
		err := openDeskItem(preset, item.MessageID)
		return func() {
			m.opening = false
			if err != nil {
				m.fail(err)
				return
			}
			m.leave = true
		}
	})
	return false
}

// mark hides the row at once and asks the desk to record the read. The
// desk's answer replaces the cache; a failure brings the row back.
func (m *model) mark() {
	item, ok := m.current()
	if !ok {
		return
	}
	preset := m.presetID()
	key := markKey(preset, item.MessageID)
	m.marking[key] = true
	m.status = "marking"
	m.err = nil
	m.clamp()
	m.jobs.Go(func() func() {
		board, err := readDeskItem(preset, item.MessageID)
		return func() {
			delete(m.marking, key)
			if err != nil {
				m.fail(err)
				m.clamp()
				return
			}
			m.replace(board)
			m.status = "marked"
			m.err = nil
		}
	})
}

func markKey(preset, id string) string {
	return preset + "/" + id
}

// sync asks the desk to refresh from Slack and then reads the desk. Keys keep
// working while it runs. A failure leaves the rows in place.
func (m *model) sync() {
	if m.syncing {
		return
	}
	m.syncing = true
	m.status = "syncing"
	m.err = nil
	m.jobs.Go(func() func() {
		err := syncDesk()
		var board Board
		if err == nil {
			board, err = fetchBoard()
		}
		return func() {
			m.syncing = false
			if err != nil {
				m.fail(err)
				return
			}
			m.replace(board)
			m.status = ""
			m.err = nil
		}
	})
}

// replace swaps in the desk's answer and keeps the workspace and row.
func (m *model) replace(board Board) {
	preset := m.presetID()
	m.board = board
	m.ws = workspaceIndex(board.Workspaces, preset)
	m.clamp()
}

func (m *model) fail(err error) {
	m.err = err
	m.status = err.Error()
}

// workspace is the open workspace as the table shows it: the desk's last
// answer without the rows still being marked read.
func (m *model) workspace() (Workspace, bool) {
	if m.ws < 0 || m.ws >= len(m.board.Workspaces) {
		return Workspace{}, false
	}
	ws := m.board.Workspaces[m.ws]
	if len(m.marking) == 0 {
		return ws, true
	}
	ws.DMs = m.unmarked(ws.Preset, ws.DMs)
	ws.Tagged = m.unmarked(ws.Preset, ws.Tagged)
	ws.Aware = m.unmarked(ws.Preset, ws.Aware)
	return ws, true
}

func (m *model) unmarked(preset string, items []Item) []Item {
	out := make([]Item, 0, len(items))
	for _, item := range items {
		if !m.marking[markKey(preset, item.MessageID)] {
			out = append(out, item)
		}
	}
	return out
}

func (m *model) presetID() string {
	ws, ok := m.workspace()
	if !ok {
		return ""
	}
	return ws.Preset
}

func (m *model) count() int {
	ws, ok := m.workspace()
	if !ok {
		return 0
	}
	return len(ws.items())
}

func (m *model) clamp() int {
	n := m.count()
	if m.cursor < 0 || n == 0 {
		m.cursor = 0
		return 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	return m.cursor
}

func (m *model) current() (Item, bool) {
	ws, ok := m.workspace()
	if !ok {
		return Item{}, false
	}
	items := ws.items()
	if len(items) == 0 {
		return Item{}, false
	}
	return items[m.clamp()], true
}
