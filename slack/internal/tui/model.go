// Package tui is the r2d2 slack screen.
package tui

import (
	"time"

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

type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

type keyWaiter interface {
	readUntil(deadline time.Time) (key string, ok bool, err error)
}

type model struct {
	board   Board
	ws      int
	cursor  int
	status  string
	err     error
	help    bool
	width   int
	height  int
	session *tuisession.Session
}

func newModel(board Board, sess *tuisession.Session) *model {
	return &model{board: board, session: sess}
}

func (m *model) loop(term screen) error {
	m.restore()
	if m.session != nil {
		stop := startSnapshots()
		defer stop()
	}
	next := now().Add(syncEvery)
	for {
		m.width, m.height = term.size()
		term.paint(m.render())
		key, ok, err := waitKey(term, m.readDeadline(next))
		if err != nil {
			return err
		}
		if !ok {
			if m.help {
				continue
			}
			m.paintSync(term)
			next = now().Add(syncEvery)
			continue
		}
		manual := key == "s" && !m.help
		if m.syncKey(term, key) {
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

func (m *model) paintSync(term screen) {
	m.status = "syncing"
	m.err = nil
	term.paint(m.render())
	m.sync()
}

func waitKey(term screen, deadline time.Time) (string, bool, error) {
	if waiter, ok := term.(keyWaiter); ok {
		return waiter.readUntil(deadline)
	}
	key, err := term.readKey()
	if err != nil {
		return "", false, err
	}
	return key, true, nil
}

func (m *model) syncKey(term screen, key string) bool {
	if key == "s" && !m.help {
		m.status = "syncing"
		m.err = nil
		term.paint(m.render())
	}
	return m.handleKey(key)
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

func (m *model) open() bool {
	item, ok := m.current()
	if !ok {
		return false
	}
	if err := openDeskItem(m.presetID(), item.MessageID); err != nil {
		m.fail(err)
		return false
	}
	return true
}

func (m *model) mark() {
	item, ok := m.current()
	if !ok {
		return
	}
	if err := readDeskItem(m.presetID(), item.MessageID); err != nil {
		m.fail(err)
		return
	}
	if err := m.reload(); err != nil {
		m.fail(err)
		return
	}
	m.status = "marked"
	m.err = nil
}

func (m *model) sync() {
	if err := syncDesk(); err != nil {
		m.fail(err)
		return
	}
	if err := m.reload(); err != nil {
		m.fail(err)
		return
	}
	m.status = ""
	m.err = nil
}

func (m *model) reload() error {
	preset := m.presetID()
	cursor := m.cursor
	board, err := fetchBoard()
	if err != nil {
		return err
	}
	m.board = board
	m.ws = workspaceIndex(board.Workspaces, preset)
	m.cursor = cursor
	m.clamp()
	return nil
}

func (m *model) fail(err error) {
	m.err = err
	m.status = err.Error()
}

func (m *model) workspace() (Workspace, bool) {
	if m.ws < 0 || m.ws >= len(m.board.Workspaces) {
		return Workspace{}, false
	}
	return m.board.Workspaces[m.ws], true
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
