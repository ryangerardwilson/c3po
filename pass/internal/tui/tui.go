// Package tui is the full-screen 1Password browser.
package tui

import (
	"fmt"
	"sort"
	"strings"

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
// so this process does not read the secret.
type Source interface {
	Vaults() ([]Vault, error)
	Items(vault string) ([]Item, error)
	Copy(vault, id string) error
	Delete(vault, id string) error
}

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;97m"
	styleMuted  = "\x1b[90m"
	styleReset  = "\x1b[0m"
)

type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

type row struct {
	vault    bool
	id       string
	vaultID  string
	name     string
	detail   string
	category string
}

type model struct {
	src       Source
	vaultName string
	rows      []row
	cursor    int
	status    string
	err       error
	width     int
	height    int
	help      bool
	armed     bool
	session   *tuisession.Session
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
	m.loadVaults()
	if body, ok := sess.Restored(); ok && body.Path != "" {
		m.loadItems(body.Path)
	}
	return m.loop(term)
}

func (m *model) loop(term screen) error {
	for {
		m.width, m.height = term.size()
		term.paint(m.view())
		key, err := term.readKey()
		if err != nil {
			return err
		}
		if m.handleKey(key) {
			m.noteSession()
			return m.err
		}
		m.noteSession()
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
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	case "j", "down":
		m.cursor++
	case "k", "up":
		m.cursor--
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.rows) - 1
	case "h", "left", "backspace":
		m.goUp()
	case "l", "right", "enter":
		m.activate(key)
	}
	m.cursor = m.clamp()
	return false
}

func (m *model) activate(key string) {
	if len(m.rows) == 0 {
		return
	}
	row := m.rows[m.clamp()]
	if row.vault {
		m.loadItems(row.name)
		return
	}
	if key == "enter" {
		m.copy(row)
	}
}

func (m *model) goUp() {
	if m.vaultName == "" {
		return
	}
	m.loadVaults()
}

func (m *model) loadVaults() {
	m.vaultName = ""
	m.cursor = 0
	vaults, err := m.src.Vaults()
	if err != nil {
		m.rows = nil
		m.fail(err)
		return
	}
	m.rows = vaultRows(vaults)
	m.err = nil
	m.status = ""
	if len(m.rows) == 0 {
		m.status = "no vaults"
	}
	m.restoreSession()
}

func (m *model) loadItems(name string) {
	m.vaultName = name
	m.cursor = 0
	items, err := m.src.Items(name)
	if err != nil {
		m.rows = nil
		m.fail(err)
		return
	}
	m.rows = itemRows(items)
	m.err = nil
	m.status = ""
	if len(m.rows) == 0 {
		m.status = "no items"
	}
	m.restoreSession()
}

func (m *model) noteSession() {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		_ = m.session.Place(m.vaultName, "")
		return
	}
	row := m.rows[m.cursor]
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

// copy asks nebula crypt to copy one item. The secret is not returned.
func (m *model) copy(row row) {
	if err := m.src.Copy(m.vaultName, row.id); err != nil {
		m.fail(err)
		return
	}
	m.err = nil
	m.status = "copied " + row.name
}

func (m *model) fail(err error) {
	m.err = err
	m.status = err.Error()
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

func (m *model) view() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	title := "pass - vaults"
	if m.vaultName != "" {
		title = "pass - " + m.vaultName
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
	if len(m.rows) == 0 {
		b.WriteString("Nothing here.\n")
		return
	}
	start, end := m.window()
	for i := start; i < end; i++ {
		row := m.rows[i]
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
func (m *model) window() (int, int) {
	rows := m.height - 4
	if m.height <= 0 || rows <= 0 || len(m.rows) <= rows {
		return 0, len(m.rows)
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
