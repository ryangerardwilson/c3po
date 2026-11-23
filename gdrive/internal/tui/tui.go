// Package tui is the full-screen Drive navigator.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
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

// Each Drive call runs on the screen's worker with its own bound.
const (
	connectWait = 30 * time.Second
	listWait    = 30 * time.Second
	trashWait   = 30 * time.Second
	uploadWait  = 30 * time.Minute
)

// jobPoll is how often the loop looks for a finished job while one runs.
const jobPoll = 50 * time.Millisecond

// leaveWait bounds the wait for a delete or upload still running when the
// screen leaves.
const leaveWait = 10 * time.Second

// Connect reaches Drive within ctx: it refreshes the account's token and
// returns the client.
type Connect func(ctx context.Context) (Drive, error)

// screen is the terminal the navigator draws on and reads keys from.
type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

// keyWaiter is a terminal that can stop waiting for a key at a deadline.
type keyWaiter interface {
	readUntil(deadline time.Time) (key string, ok bool, err error)
}

// model is the screen's UI state. parentStack, parentID, pathLabel, and
// entries are a cache of the folder Drive listed last and are only ever
// replaced whole. next is the folder asked for last while visits are in
// flight; the frame shows it loading. trashing holds the files with a
// delete in flight; the list hides them. writes counts deletes and uploads
// not finished.
type model struct {
	connect     Connect
	client      Drive
	parentStack []stackItem
	parentID    string
	pathLabel   string
	entries     []driveapi.NavEntry
	next        folder
	visits      int
	trashing    map[string]bool
	writes      int
	cursor      int
	status      string
	err         error
	width       int
	height      int
	open        func(string) error
	pick        func() (string, error)
	armed       bool
	help        bool
	session     *tuisession.Session
	jobs        *tuijob.Queue
}

type stackItem struct {
	ID    string
	Label string
}

// folder is one place in Drive: its id, its label, and the folders above
// it.
type folder struct {
	id    string
	label string
	stack []stackItem
}

func Browse(connect Connect) (int, error) {
	return browse(connect, nil)
}

// BrowseSession opens Drive and restores the saved folder when sess is set.
func BrowseSession(connect Connect, sess *tuisession.Session) (int, error) {
	return browse(connect, sess)
}

func browse(connect Connect, sess *tuisession.Session) (int, error) {
	if err := run(connect, sess); err != nil {
		return 1, err
	}
	return 0, nil
}

func run(connect Connect, sess *tuisession.Session) error {
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	m := newModel(connect)
	m.session = sess
	m.pick = func() (string, error) { return pickLocal(term) }
	if body, ok := sess.Restored(); ok && body.Path != "" {
		m.parentID = body.Path
	}
	return m.loop(term)
}

func newModel(connect Connect) *model {
	return &model{connect: connect, parentID: "root", pathLabel: "My Drive", open: openBrowser}
}

// loop paints, then waits for a key or a finished job. The first frame
// says loading before anything reaches Google; no key waits on Drive
// before the next paint.
func (m *model) loop(term screen) error {
	m.jobs = tuijob.New()
	defer m.jobs.Close()
	if m.client == nil && m.connect != nil {
		m.start()
	}
	for {
		m.width, m.height = term.size()
		term.paint(m.View())
		key, ok, err := m.waitKey(term)
		if err != nil {
			return err
		}
		m.jobs.Apply()
		if ok && m.handleKey(key) {
			m.drain(term)
			m.noteSession()
			return m.err
		}
		m.noteSession()
	}
}

// waitKey reads the next key. While a job runs it wakes every jobPoll so
// the result is painted when it lands. A terminal that cannot wait with a
// deadline lets the job finish before it reads a key.
func (m *model) waitKey(term screen) (string, bool, error) {
	if m.jobs.Busy() {
		if waiter, ok := term.(keyWaiter); ok {
			return waiter.readUntil(time.Now().Add(jobPoll))
		}
		m.jobs.Wait(uploadWait)
		return "", false, nil
	}
	key, err := term.readKey()
	if err != nil {
		return "", false, err
	}
	return key, true, nil
}

// drain lets a delete or upload still running end before the screen
// leaves.
func (m *model) drain(term screen) {
	if m.writes == 0 {
		return
	}
	m.status = fmt.Sprintf("saving %d", m.writes)
	m.err = nil
	term.paint(m.View())
	m.jobs.Wait(leaveWait)
	m.jobs.Apply()
}

// handleKey applies one key and reports whether the navigator quits.
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
			return m.deleteItem()
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
	return m.applyMotion(key)
}

func (m *model) applyMotion(key string) bool {
	n := len(m.rows())
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	case "j", "down":
		m.cursor = cycled(m.cursor, 1, n)
	case "k", "up":
		m.cursor = cycled(m.cursor, -1, n)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = n - 1
	case "h", "left", "backspace":
		m.goUp()
	case "l", "right":
		m.descend()
	case "enter":
		return m.openItem()
	case "u":
		return m.upload()
	}
	m.cursor = m.clampedCursor()
	return false
}

func (m *model) goUp() {
	at := m.place()
	n := len(at.stack)
	if n == 0 {
		return
	}
	last := at.stack[n-1]
	m.visit(folder{id: last.ID, label: last.Label, stack: at.stack[: n-1 : n-1]})
}

func (m *model) descend() {
	rows := m.rows()
	if len(rows) == 0 {
		return
	}
	entry := rows[m.clampedCursor()]
	if !entry.IsDir() {
		return
	}
	stack := make([]stackItem, len(m.parentStack), len(m.parentStack)+1)
	copy(stack, m.parentStack)
	stack = append(stack, stackItem{ID: m.parentID, Label: m.pathLabel})
	m.visit(folder{id: entry.ID, label: strings.Trim(m.pathLabel+"/"+entry.Name, "/"), stack: stack})
}

func (m *model) openItem() bool {
	rows := m.rows()
	if len(rows) == 0 {
		return false
	}
	entry := rows[m.clampedCursor()]
	if err := m.open(itemURL(entry)); err != nil {
		m.err = err
		m.status = err.Error()
		return false
	}
	m.err = nil
	m.status = "opened " + entry.Name
	return false
}

// start reaches Drive and lists the first folder on the worker. The frame
// says loading meanwhile.
func (m *model) start() {
	connect := m.connect
	to := m.place()
	m.next = to
	m.visits++
	m.jobs.Go(func() func() {
		client, err := connectWithin(connect)
		var entries []driveapi.NavEntry
		if err == nil {
			entries, err = listWithin(client, to.id)
		}
		return func() {
			if client != nil {
				m.client = client
			}
			m.landed(to, entries, err)
		}
	})
}

// visit shows the folder loading at once and asks Drive to list it.
// Listings land in the order they were asked; a failure leaves the last
// folder.
func (m *model) visit(to folder) {
	client := m.client
	if client == nil {
		return
	}
	m.next = to
	m.visits++
	m.cursor = 0
	m.err = nil
	m.status = ""
	m.jobs.Go(func() func() {
		entries, err := listWithin(client, to.id)
		return func() { m.landed(to, entries, err) }
	})
}

// landed takes one folder listing as the cache.
func (m *model) landed(to folder, entries []driveapi.NavEntry, err error) {
	m.visits--
	if err != nil {
		m.fail(err)
		return
	}
	m.parentStack, m.parentID, m.pathLabel = to.stack, to.id, to.label
	m.entries = entries
	m.cursor = 0
	m.restoreSession()
}

// relist takes a listing of parent as the cache while the screen still
// holds that folder.
func (m *model) relist(parent string, entries []driveapi.NavEntry) {
	if m.parentID != parent {
		return
	}
	m.entries = entries
	m.cursor = m.clampedCursor()
}

func (m *model) fail(err error) {
	m.err = err
	m.status = err.Error()
}

// place is the folder the frame names: the one asked for last.
func (m *model) place() folder {
	if m.visits > 0 {
		return m.next
	}
	return folder{id: m.parentID, label: m.pathLabel, stack: m.parentStack}
}

// rows is the folder as the frame draws it: nothing while a folder loads,
// else Drive's last listing without the files being deleted.
func (m *model) rows() []driveapi.NavEntry {
	if m.visits > 0 {
		return nil
	}
	if len(m.trashing) == 0 {
		return m.entries
	}
	out := make([]driveapi.NavEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		if !m.trashing[entry.ID] {
			out = append(out, entry)
		}
	}
	return out
}

func connectWithin(connect Connect) (Drive, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectWait)
	defer cancel()
	return connect(ctx)
}

func listWithin(client Lister, parentID string) ([]driveapi.NavEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listWait)
	defer cancel()
	return client.ListChildren(ctx, parentID)
}

// noteSession keeps the place in the session. It waits while a folder
// loads, so a restored row is not dropped before its folder arrives.
func (m *model) noteSession() {
	if m.visits > 0 {
		return
	}
	rows := m.rows()
	cursor := ""
	if m.cursor >= 0 && m.cursor < len(rows) {
		cursor = rows[m.cursor].ID
	}
	_ = m.session.Place(m.parentID, cursor)
}

func (m *model) restoreSession() {
	if m.session == nil {
		return
	}
	keys := make([]string, len(m.entries))
	for i, entry := range m.entries {
		keys[i] = entry.ID
	}
	if i, ok := m.session.Index(keys); ok {
		m.cursor = i
	}
}

func (m *model) clampedCursor() int {
	n := len(m.rows())
	if n == 0 || m.cursor < 0 {
		return 0
	}
	if m.cursor >= n {
		return n - 1
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

func (m *model) View() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", paintStyle(styleTitle, "gdrive browse - "+m.place().label))
	m.writeEntries(&b)
	if m.status != "" {
		style := styleMuted
		if m.err != nil || strings.Contains(strings.ToLower(m.status), "error") {
			style = styleError
		}
		fmt.Fprintf(&b, "\n%s\n", paintStyle(style, m.status))
	}
	return b.String()
}

func (m *model) helpView() string {
	lines := []string{
		paintStyle(styleTitle, "gdrive"),
		"",
		"j/k    move",
		"l      open a folder",
		"h      up",
		"enter  open in the browser",
		"u      upload",
		"dd     delete the selected file",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) writeEntries(b *strings.Builder) {
	if m.visits > 0 {
		b.WriteString(paintStyle(styleMuted, "loading") + "\n")
		return
	}
	rows := m.rows()
	if len(rows) == 0 {
		b.WriteString("No files.\n")
		return
	}
	start, end := m.window(len(rows))
	for idx := start; idx < end; idx++ {
		entry := rows[idx]
		prefix, style := "  ", ""
		if idx == m.cursor {
			prefix, style = "> ", styleActive
		}
		icon := "file"
		if entry.IsDir() {
			icon = "dir "
		}
		line := truncate(fmt.Sprintf("%s%s  %s", prefix, icon, entry.Name), m.width)
		b.WriteString(paintStyle(style, line))
		b.WriteByte('\n')
	}
}

// window is the slice of entries that fits under the header and above
// the status line, kept around the cursor.
func (m *model) window(n int) (int, int) {
	rows := m.height - 4
	if m.height <= 0 || rows <= 0 || n <= rows {
		return 0, n
	}
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	return start, start + rows
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

var escapeKeys = map[string]string{
	"[A": "up", "OA": "up",
	"[B": "down", "OB": "down",
	"[C": "right", "OC": "right",
	"[D": "left", "OD": "left",
	"[H": "home", "OH": "home", "[1~": "home", "[7~": "home",
	"[F": "end", "OF": "end", "[4~": "end", "[8~": "end",
}

// decodeKey names the key in one terminal read. An unknown escape
// sequence is the empty name and does nothing.
func decodeKey(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if data[0] == 0x1b && len(data) > 1 {
		return escapeKeys[string(data[1:])]
	}
	switch data[0] {
	case 0x03:
		return "ctrl+c"
	case 0x1b:
		return "esc"
	case '\r', '\n':
		return "enter"
	case 0x7f, 0x08:
		return "backspace"
	default:
		return string(data[:1])
	}
}
