// Package tui is the full-screen Drive navigator for browse and upload.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/gdrive/internal/transfer"
)

type Mode string

const (
	ModeBrowse Mode = "browse"
	ModeUpload Mode = "upload"
)

type Result struct {
	UploadSummary *transfer.UploadSummary
	UploadTarget  string
}

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;92m"
	styleMuted  = "\x1b[90m"
	styleError  = "\x1b[91m"
	styleReset  = "\x1b[0m"
)

// screen is the terminal the navigator draws on and reads keys from.
type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

type model struct {
	client      driveapi.Client
	mode        Mode
	downloadDir string
	uploadPaths []string
	parentStack []stackItem
	parentID    string
	pathLabel   string
	entries     []driveapi.NavEntry
	cursor      int
	status      string
	result      Result
	err         error
	width       int
	height      int
}

type stackItem struct {
	ID    string
	Label string
}

type actionMsg struct {
	status string
	result Result
	err    error
	quit   bool
}

func Browse(client driveapi.Client, downloadDir string) (int, error) {
	_, err := run(client, ModeBrowse, downloadDir, nil)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func Upload(client driveapi.Client, downloadDir string, uploadPaths []string) (Result, int, error) {
	result, err := run(client, ModeUpload, downloadDir, uploadPaths)
	if err != nil {
		return result, 1, err
	}
	return result, 0, nil
}

func run(client driveapi.Client, mode Mode, downloadDir string, uploadPaths []string) (Result, error) {
	term, err := openTerminal()
	if err != nil {
		return Result{}, err
	}
	defer term.close()
	m := newModel(client, mode, downloadDir, uploadPaths)
	m.load("root")
	return m.loop(term)
}

func newModel(client driveapi.Client, mode Mode, downloadDir string, uploadPaths []string) *model {
	return &model{client: client, mode: mode, downloadDir: downloadDir, uploadPaths: uploadPaths, parentID: "root", pathLabel: "My Drive"}
}

func (m *model) loop(term screen) (Result, error) {
	for {
		m.width, m.height = term.size()
		term.paint(m.View())
		key, err := term.readKey()
		if err != nil {
			return Result{}, err
		}
		if m.handleKey(key) {
			return m.result, m.err
		}
	}
}

// handleKey applies one key and reports whether the navigator quits.
func (m *model) handleKey(key string) bool {
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
		m.cursor = len(m.entries) - 1
	case "h", "left", "backspace":
		m.goUp()
	case "l", "right", "enter":
		return m.activate(key)
	}
	m.cursor = m.clampedCursor()
	return false
}

func (m *model) goUp() {
	if len(m.parentStack) == 0 {
		return
	}
	last := m.parentStack[len(m.parentStack)-1]
	m.parentStack = m.parentStack[:len(m.parentStack)-1]
	m.pathLabel = last.Label
	m.load(last.ID)
}

func (m *model) activate(key string) bool {
	if len(m.entries) == 0 {
		return false
	}
	entry := m.entries[m.clampedCursor()]
	if entry.IsDir() {
		if m.mode == ModeUpload && key == "enter" {
			return m.apply(m.upload(entry))
		}
		m.parentStack = append(m.parentStack, stackItem{ID: m.parentID, Label: m.pathLabel})
		m.pathLabel = strings.Trim(m.pathLabel+"/"+entry.Name, "/")
		m.load(entry.ID)
		return false
	}
	if m.mode == ModeUpload {
		return m.apply(m.uploadToParent(entry))
	}
	return m.apply(m.download(entry))
}

func (m *model) apply(msg actionMsg) bool {
	if msg.err != nil {
		m.status = msg.err.Error()
		return false
	}
	m.status = msg.status
	m.result = msg.result
	return msg.quit
}

func (m *model) load(parentID string) {
	entries, err := m.client.ListChildren(parentID)
	if err != nil {
		m.err = err
		m.status = err.Error()
		return
	}
	m.parentID = parentID
	m.entries = entries
	m.cursor = 0
}

func (m *model) download(entry driveapi.NavEntry) actionMsg {
	target := filepath.Join(m.downloadDir, entry.Name)
	var downloaded string
	var err error
	if entry.IsDir() {
		downloaded, err = transfer.DownloadDirectoryAsZip(m.client, entry, target)
	} else {
		downloaded, err = m.client.DownloadEntry(entry, target)
	}
	if err != nil {
		return actionMsg{err: err}
	}
	return actionMsg{status: "downloaded " + downloaded}
}

func (m *model) upload(entry driveapi.NavEntry) actionMsg {
	summary, err := transfer.UploadLocalPaths(m.client, entry.ID, m.uploadPaths)
	if err != nil {
		return actionMsg{err: err}
	}
	return actionMsg{status: "uploaded", result: Result{UploadSummary: &summary, UploadTarget: m.pathLabel + "/" + entry.Name}, quit: true}
}

func (m *model) uploadToParent(entry driveapi.NavEntry) actionMsg {
	summary, err := transfer.UploadLocalPaths(m.client, entry.ParentID, m.uploadPaths)
	if err != nil {
		return actionMsg{err: err}
	}
	return actionMsg{status: "uploaded", result: Result{UploadSummary: &summary, UploadTarget: m.pathLabel}, quit: true}
}

func (m *model) clampedCursor() int {
	if len(m.entries) == 0 || m.cursor < 0 {
		return 0
	}
	if m.cursor >= len(m.entries) {
		return len(m.entries) - 1
	}
	return m.cursor
}

func (m *model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", paintStyle(styleTitle, fmt.Sprintf("gdrive %s - %s", m.mode, m.pathLabel)))
	fmt.Fprintf(&b, "%s\n\n", paintStyle(styleMuted, "j/k move  l enter  h up  enter download/upload  q quit"))
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

func (m *model) writeEntries(b *strings.Builder) {
	if len(m.entries) == 0 {
		b.WriteString("No files.\n")
		return
	}
	start, end := m.window()
	for idx := start; idx < end; idx++ {
		entry := m.entries[idx]
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
func (m *model) window() (int, int) {
	rows := m.height - 5
	if m.height <= 0 || rows <= 0 || len(m.entries) <= rows {
		return 0, len(m.entries)
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
