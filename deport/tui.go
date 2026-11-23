package deport

import (
	"fmt"
	"strconv"
	"strings"
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

// jobPoll is how often the loop looks for a finished job while one runs.
const jobPoll = 50 * time.Millisecond

// leaveWait bounds the wait for a kill still running when the screen
// leaves. One kill takes about 1.3 s at most.
const leaveWait = 5 * time.Second

// refreshEvery is how long the list waits without a key before it reads
// the ports again.
var refreshEvery = 2 * time.Second

// screen is the terminal the list draws on and reads keys from.
type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

// keyWaiter is a terminal that can stop waiting for a key at a deadline.
type keyWaiter interface {
	readUntil(deadline time.Time) (key string, ok bool, err error)
}

// model is the screen's UI state. rows is a cache of the last port read
// and is only ever replaced whole. freeing holds the ports with a kill in
// flight; the list marks those rows until the kill ends.
type model struct {
	rows    []occupant
	freeing map[int]bool
	cursor  int
	armed   bool
	status  string
	err     error
	width   int
	height  int
	load    func() ([]occupant, error)
	release func(port int) ([]proc, error)
	help    bool
	session *tuisession.Session
	jobs    *tuijob.Queue
}

func runScreen(e env, sess *tuisession.Session) error {
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	m := &model{
		load: func() ([]occupant, error) { return occupied(e.procRoot) },
		release: func(port int) ([]proc, error) {
			return free(e.procRoot, port, e.killer, e.sleep)
		},
		session: sess,
	}
	return m.loop(term)
}

// loop paints, then waits for a key or a finished job. A key only changes
// UI state and may queue a kill; no key waits on a signal or a port read
// before the next paint. The ports are read again on the worker after a
// kill and after refreshEvery without a key.
func (m *model) loop(term screen) error {
	m.jobs = tuijob.New()
	defer m.jobs.Close()
	m.refresh()
	next := time.Now().Add(refreshEvery)
	for {
		m.width, m.height = term.size()
		term.paint(m.view())
		key, ok, err := m.waitKey(term, next)
		if err != nil {
			return err
		}
		m.jobs.Apply()
		if !ok {
			if m.jobs.Busy() || time.Now().Before(next) {
				continue
			}
			m.reload()
			next = time.Now().Add(refreshEvery)
			continue
		}
		if m.handleKey(key) {
			m.drain(term)
			m.noteSession()
			return nil
		}
		m.noteSession()
		next = time.Now().Add(refreshEvery)
	}
}

// waitKey waits for a key until the next port read is due. While a job
// runs it wakes every jobPoll so the result is painted when it lands. A
// terminal that cannot wait with a deadline lets the job finish before it
// reads a key.
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
		return waiter.readUntil(time.Now().Add(jobPoll))
	}
	return waiter.readUntil(next)
}

// drain lets a kill still running end before the screen leaves.
func (m *model) drain(term screen) {
	n := len(m.freeing)
	if n == 0 {
		return
	}
	m.status = "freeing 1 port"
	if n > 1 {
		m.status = fmt.Sprintf("freeing %d ports", n)
	}
	m.err = nil
	term.paint(m.view())
	m.jobs.Wait(leaveWait)
	m.jobs.Apply()
}

// refresh reads the ports before the first paint.
func (m *model) refresh() {
	if m.load == nil {
		return
	}
	rows, err := m.load()
	m.landed(rows, err)
}

// reload reads the ports on the worker.
func (m *model) reload() {
	load := m.load
	if load == nil {
		return
	}
	m.jobs.Go(func() func() {
		rows, err := load()
		return func() { m.landed(rows, err) }
	})
}

// landed replaces the list with a port read and keeps the cursor on the
// same port when it is still there.
func (m *model) landed(rows []occupant, err error) {
	if err != nil {
		m.err = err
		m.status = err.Error()
		m.rows = nil
		m.cursor = 0
		return
	}
	port := -1
	if len(m.rows) > 0 {
		port = m.rows[m.clamp()].port
	}
	m.rows = rows
	m.cursor = m.clamp()
	for i, row := range rows {
		if row.port == port {
			m.cursor = i
		}
	}
	m.restoreSession()
}

func (m *model) noteSession() {
	cursor := ""
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		cursor = strconv.Itoa(m.rows[m.cursor].port)
	}
	_ = m.session.Place("", cursor)
}

func (m *model) restoreSession() {
	if m.session == nil {
		return
	}
	keys := make([]string, len(m.rows))
	for i, row := range m.rows {
		keys[i] = strconv.Itoa(row.port)
	}
	if i, ok := m.session.Index(keys); ok {
		m.cursor = i
	}
}

// handleKey applies one key and reports whether the screen quits.
func (m *model) handleKey(key string) bool {
	if m.help {
		switch key {
		case "?", "esc":
			m.help = false
		case "ctrl+c", "q":
			return true
		}
		return false
	}
	if key == "?" {
		m.help = true
		m.armed = false
		m.status = ""
		return false
	}
	if m.armed {
		m.armed = false
		if key == "d" {
			m.freeSelected()
			return false
		}
		if key == "esc" {
			m.status = ""
			return false
		}
		m.status = ""
	} else if key == "d" {
		m.armed = true
		m.status = "d"
		return false
	}
	return m.motion(key)
}

func (m *model) motion(key string) bool {
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	case "j", "down":
		m.cursor = cycled(m.cursor, 1, len(m.rows))
	case "k", "up":
		m.cursor = cycled(m.cursor, -1, len(m.rows))
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.rows) - 1
	}
	m.cursor = m.clamp()
	return false
}

// freeSelected marks the row at once and kills its holders on the worker.
// The port read after the kill replaces the list; a failure keeps the row
// and says why.
func (m *model) freeSelected() {
	if len(m.rows) == 0 {
		m.status = "no occupied external ports"
		return
	}
	row := m.rows[m.clamp()]
	if m.freeing[row.port] {
		return
	}
	if m.freeing == nil {
		m.freeing = map[int]bool{}
	}
	m.freeing[row.port] = true
	m.err = nil
	m.status = fmt.Sprintf("freeing %d", row.port)
	release, load := m.release, m.load
	m.jobs.Go(func() func() {
		_, err := release(row.port)
		var rows []occupant
		var loadErr error
		if load != nil {
			rows, loadErr = load()
		}
		return func() {
			delete(m.freeing, row.port)
			if load != nil {
				m.landed(rows, loadErr)
			}
			if err != nil {
				m.err = err
				m.status = err.Error()
				return
			}
			if loadErr == nil {
				m.err = nil
				m.status = fmt.Sprintf("%d is free", row.port)
			}
		}
	})
}

func (m *model) clamp() int {
	if len(m.rows) == 0 || m.cursor < 0 {
		return 0
	}
	if m.cursor >= len(m.rows) {
		return len(m.rows) - 1
	}
	return m.cursor
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

func (m *model) view() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", paintStyle(styleTitle, "deport"))
	if len(m.rows) == 0 {
		b.WriteString("no occupied external ports\n")
	} else {
		start, end := m.window()
		for i := start; i < end; i++ {
			prefix, style := "  ", ""
			if i == m.clamp() {
				prefix, style = "> ", styleActive
			}
			text := formatOccupant(m.rows[i])
			if m.freeing[m.rows[i].port] {
				text += "  freeing"
			}
			line := truncate(prefix+text, m.width)
			b.WriteString(paintStyle(style, line))
			b.WriteByte('\n')
		}
	}
	if m.status != "" {
		style := styleMuted
		if m.err != nil {
			style = styleError
		}
		fmt.Fprintf(&b, "\n%s\n", paintStyle(style, m.status))
	}
	return b.String()
}

func (m *model) helpView() string {
	lines := []string{
		paintStyle(styleTitle, "deport"),
		"",
		"j/k    move",
		"dd     free the selected port",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) window() (int, int) {
	rows := m.height - 3
	if m.height <= 0 || rows < 1 || len(m.rows) <= rows {
		return 0, len(m.rows)
	}
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	end := start + rows
	if end > len(m.rows) {
		end = len(m.rows)
	}
	return start, end
}

func paintStyle(style, text string) string {
	if style == "" {
		return text
	}
	return style + text + styleReset
}

func truncate(value string, width int) string {
	if width <= 4 || len(value) <= width {
		return value
	}
	return value[:width-3] + "..."
}
