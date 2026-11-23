// Package tui is the full-screen 1Password browser.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/internal/tuijob"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

// Vault is one 1Password vault named by nebula crypt.
type Vault struct {
	ID   string
	Name string
}

// Item is one 1Password item. It does not carry a concealed value.
type Item struct {
	ID       string
	VaultID  string
	Title    string
	Category string
	Username string
}

// Source lists vaults and items. Copy and Delete go through nebula crypt,
// so this process does not read the secret. Each call ends when ctx does.
type Source interface {
	Vaults(ctx context.Context) ([]Vault, error)
	Items(ctx context.Context, vault string) ([]Item, error)
	Copy(ctx context.Context, vault, id string) error
	Delete(ctx context.Context, vault, id string) error
}

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;97m"
	styleMuted  = "\x1b[90m"
	styleReset  = "\x1b[0m"
)

// cryptWait bounds one nebula crypt call. 1Password may ask to be unlocked
// first.
const cryptWait = time.Minute

// jobPoll is how often the loop looks for a finished job while one runs.
const jobPoll = 50 * time.Millisecond

// leaveWait bounds the wait for a copy or delete still running when the
// screen leaves.
const leaveWait = 10 * time.Second

type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

// keyWaiter is a terminal that can stop waiting for a key at a deadline.
type keyWaiter interface {
	readUntil(deadline time.Time) (key string, ok bool, err error)
}

type row struct {
	vault    bool
	id       string
	vaultID  string
	name     string
	detail   string
	category string
}

// model is the screen's UI state. vaultName and rows are a cache of
// nebula's last list: the vaults when vaultName is empty, else that vault's
// items. They are only ever replaced whole. next is the list asked for last
// while visits are in flight; the frame shows it loading. deleting holds
// the items with a delete in flight; the list hides them.
type model struct {
	src       Source
	vaultName string
	rows      []row
	next      string
	visits    int
	deleting  map[string]bool
	writes    int
	cursor    int
	status    string
	err       error
	width     int
	height    int
	help      bool
	armed     bool
	session   *tuisession.Session
	jobs      *tuijob.Queue
}

// Run opens the vault list. q leaves it.
func Run(src Source) error {
	return RunSession(src, nil)
}

// RunSession opens the vault list and restores a saved row when sess is set.
func RunSession(src Source, sess *tuisession.Session) error {
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	m := &model{src: src, session: sess}
	return m.loop(term)
}

// loop paints, then waits for a key or a finished job. The first frame
// says loading; no key waits on nebula before the next paint.
func (m *model) loop(term screen) error {
	m.jobs = tuijob.New()
	defer m.jobs.Close()
	m.loadVaults()
	if body, ok := m.session.Restored(); ok && body.Path != "" {
		m.loadItems(body.Path)
	}
	for {
		m.width, m.height = term.size()
		term.paint(m.view())
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
		m.jobs.Wait(cryptWait)
		return "", false, nil
	}
	key, err := term.readKey()
	if err != nil {
		return "", false, err
	}
	return key, true, nil
}

// drain lets a copy or delete still running end before the screen leaves.
func (m *model) drain(term screen) {
	if m.writes == 0 {
		return
	}
	m.status = fmt.Sprintf("saving %d", m.writes)
	m.err = nil
	term.paint(m.view())
	m.jobs.Wait(leaveWait)
	m.jobs.Apply()
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
	n := len(m.shown())
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
	case "l", "right", "enter":
		m.activate(key)
	}
	m.cursor = m.clamp()
	return false
}

func (m *model) activate(key string) {
	rows := m.shown()
	if len(rows) == 0 {
		return
	}
	row := rows[m.clamp()]
	if row.vault {
		m.loadItems(row.name)
		return
	}
	if key == "enter" {
		m.copy(row)
	}
}

func (m *model) goUp() {
	if m.place() == "" {
		return
	}
	m.loadVaults()
}

// loadVaults asks nebula for the vault list on the worker.
func (m *model) loadVaults() {
	m.visit("")
}

// loadItems asks nebula for one vault's items on the worker.
func (m *model) loadItems(name string) {
	m.visit(name)
}

// visit shows the list loading at once and asks nebula for it. Answers
// land in the order they were asked; a failure leaves the last list.
func (m *model) visit(vault string) {
	src := m.src
	m.next = vault
	m.visits++
	m.cursor = 0
	m.err = nil
	m.status = ""
	m.jobs.Go(func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), cryptWait)
		defer cancel()
		rows, err := listRows(ctx, src, vault)
		return func() { m.landed(vault, rows, err) }
	})
}

func listRows(ctx context.Context, src Source, vault string) ([]row, error) {
	if vault == "" {
		vaults, err := src.Vaults(ctx)
		if err != nil {
			return nil, err
		}
		return vaultRows(vaults), nil
	}
	items, err := src.Items(ctx, vault)
	if err != nil {
		return nil, err
	}
	return itemRows(items), nil
}

// landed takes one list answer as the cache.
func (m *model) landed(vault string, rows []row, err error) {
	m.visits--
	if err != nil {
		m.fail(err)
		return
	}
	m.vaultName = vault
	m.rows = rows
	m.cursor = 0
	m.err = nil
	m.status = ""
	if len(rows) == 0 {
		m.status = "no vaults"
		if vault != "" {
			m.status = "no items"
		}
	}
	m.restoreSession()
}

// place is the list the frame names: the one asked for last.
func (m *model) place() string {
	if m.visits > 0 {
		return m.next
	}
	return m.vaultName
}

// shown is the list as the frame draws it: nothing while a list loads,
// else nebula's last answer without the items being deleted.
func (m *model) shown() []row {
	if m.visits > 0 {
		return nil
	}
	if len(m.deleting) == 0 {
		return m.rows
	}
	out := make([]row, 0, len(m.rows))
	for _, r := range m.rows {
		if r.vault || !m.deleting[r.id] {
			out = append(out, r)
		}
	}
	return out
}

// noteSession stores the place once the list it points into has loaded. A
// list still loading has no row yet, so storing then would wipe the place a
// --resume is restoring.
func (m *model) noteSession() {
	if m.visits > 0 {
		return
	}
	rows := m.shown()
	if m.cursor < 0 || m.cursor >= len(rows) {
		_ = m.session.Place(m.place(), "")
		return
	}
	row := rows[m.cursor]
	if row.vault {
		_ = m.session.Place("", row.id)
		return
	}
	_ = m.session.Place(m.vaultName, row.id)
}

func (m *model) restoreSession() {
	if m.session == nil {
		return
	}
	keys := make([]string, len(m.rows))
	for i, row := range m.rows {
		keys[i] = row.id
	}
	if i, ok := m.session.Index(keys); ok {
		m.cursor = i
	}
}

// copy asks nebula crypt to copy one item on the worker. The secret is not
// returned.
func (m *model) copy(row row) {
	src, vault := m.src, m.vaultName
	m.writes++
	m.err = nil
	m.status = "copying " + row.name
	m.jobs.Go(func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), cryptWait)
		defer cancel()
		err := src.Copy(ctx, vault, row.id)
		return func() {
			m.writes--
			if err != nil {
				m.fail(err)
				return
			}
			m.err = nil
			m.status = "copied " + row.name
		}
	})
}

func (m *model) fail(err error) {
	m.err = err
	m.status = err.Error()
}

func (m *model) clamp() int {
	n := len(m.shown())
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

func (m *model) view() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	title := "pass - vaults"
	if place := m.place(); place != "" {
		title = "pass - " + place
	}
	fmt.Fprintf(&b, "%s\n\n", paint(styleTitle, title))
	m.writeRows(&b)
	if m.status != "" {
		fmt.Fprintf(&b, "\n%s\n", paint(styleMuted, truncate(oneLine(m.status), m.width)))
	}
	return b.String()
}

func (m *model) helpView() string {
	lines := []string{
		paint(styleTitle, "pass"),
		"",
		"j/k    move",
		"l      open",
		"h      up",
		"enter  copy",
		"dd     delete",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) writeRows(b *strings.Builder) {
	if m.visits > 0 {
		b.WriteString(paint(styleMuted, "loading") + "\n")
		return
	}
	rows := m.shown()
	if len(rows) == 0 {
		b.WriteString("Nothing here.\n")
		return
	}
	start, end := m.window(len(rows))
	for i := start; i < end; i++ {
		row := rows[i]
		prefix, style := "  ", ""
		if i == m.cursor {
			prefix, style = "> ", styleActive
		}
		text := oneLine(row.name)
		if row.detail != "" {
			text += "  " + oneLine(row.detail)
		}
		b.WriteString(paint(style, truncate(prefix+text, m.width)))
		b.WriteByte('\n')
	}
}

// window is the rows that fit under the title and above the copy status.
// Those four lines stay reserved while the status is empty, so the list
// does not jump when a copy lands.
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

func vaultRows(vaults []Vault) []row {
	rows := make([]row, 0, len(vaults))
	for _, item := range vaults {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		rows = append(rows, row{vault: true, id: item.ID, name: name})
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].name) < strings.ToLower(rows[j].name)
	})
	return rows
}

func itemRows(items []Item) []row {
	rows := make([]row, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if item.ID != "" && seen[item.ID] {
			continue
		}
		if item.ID != "" {
			seen[item.ID] = true
		}
		name := strings.TrimSpace(item.Title)
		if name == "" {
			name = "(untitled)"
		}
		rows = append(rows, row{
			id:       item.ID,
			vaultID:  item.VaultID,
			name:     name,
			detail:   item.Username,
			category: item.Category,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].name) < strings.ToLower(rows[j].name)
	})
	return rows
}

func paint(style, text string) string {
	if style == "" {
		return text
	}
	return style + text + styleReset
}

func truncate(value string, width int) string {
	// A line as wide as the terminal wraps and scrolls the screen, which
	// pushes the cursor row out of the visible frame.
	if width <= 4 {
		return value
	}
	limit := width - 1
	if len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}

func oneLine(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, "\n", " ")
}
