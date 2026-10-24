// Package tui is the route screen behind c3po tui tunnel. It uses only the
// standard library: raw mode on Linux, ANSI output, and a small line
// input for the name and port fields.
package tui

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
	"github.com/ryangerardwilson/c3po/tunnel/internal/store"
)

const (
	modeList    = "list"
	modeAdd     = "add"
	modeConfirm = "confirm"
)

type action int

const (
	actNone action = iota
	actQuit
	actSave
)

type model struct {
	home    string
	file    routes.File
	cursor  int
	mode    string
	name    input
	port    input
	notice  string
	service bool
	local   map[int]bool
	color   bool
	help    bool
}

func newModel(home string, file routes.File, color bool) model {
	return model{
		home:  home,
		file:  file,
		mode:  modeList,
		name:  input{limit: 63},
		port:  input{limit: 5},
		local: map[int]bool{},
		color: color,
	}
}

// key applies one key and reports what the screen loop must do next.
func (m *model) key(k string) action {
	switch m.mode {
	case modeAdd:
		return m.keyAdd(k)
	case modeConfirm:
		m.keyConfirm(k)
		return actNone
	default:
		return m.keyList(k)
	}
}

func (m *model) keyList(k string) action {
	if m.help {
		switch k {
		case "?", "esc":
			m.help = false
		case "q", "ctrl+c":
			return actQuit
		}
		return actNone
	}
	switch k {
	case "?":
		m.help = true
		return actNone
	case "q", "esc", "ctrl+c":
		return actQuit
	case "j", "down":
		if m.cursor < len(m.file.Routes)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "a":
		m.mode = modeAdd
		m.notice = ""
		m.port.focused = false
		m.name.focused = true
	case "d":
		if len(m.file.Routes) > 0 {
			m.mode = modeConfirm
		}
	}
	return actNone
}

func (m *model) keyAdd(k string) action {
	switch k {
	case "esc":
		m.mode = modeList
		m.notice = ""
	case "tab", "shift+tab":
		m.name.focused, m.port.focused = !m.name.focused, m.name.focused
	case "enter":
		if m.name.focused && strings.TrimSpace(m.port.value) == "" {
			m.name.focused, m.port.focused = false, true
			return actNone
		}
		m.notice = "saving"
		return actSave
	default:
		m.focusedInput().key(k)
	}
	return actNone
}

func (m *model) focusedInput() *input {
	if m.port.focused {
		return &m.port
	}
	return &m.name
}

func (m *model) keyConfirm(k string) {
	switch k {
	case "y", "Y":
		m.mode = modeList
		if len(m.file.Routes) > 0 {
			m.removeCurrent()
		}
	case "n", "N", "esc":
		m.mode = modeList
	}
}

func (m *model) removeCurrent() {
	name := m.file.Routes[m.cursor].Name
	next, err := store.Remove(m.home, name)
	if err != nil {
		m.notice = err.Error()
		return
	}
	m.file = next
	m.cursor = min(m.cursor, len(m.file.Routes)-1)
	m.cursor = max(m.cursor, 0)
	m.notice = "removed " + name
	m.scan()
}

// saved applies the result of an add that ran in the background.
func (m *model) saved(err error) {
	if err != nil {
		m.notice = err.Error()
		return
	}
	file, err := routes.Load(routes.Path(m.home))
	if err != nil {
		m.notice = err.Error()
		return
	}
	m.file = file
	m.mode = modeList
	m.notice = "saved"
	m.name.value = ""
	m.port.value = ""
	m.scan()
}

// scan refreshes the service state and which local ports answer.
func (m *model) scan() {
	m.service = serviceUp()
	m.local = map[int]bool{}
	for _, route := range m.file.Routes {
		m.local[route.Port] = listening(route.Port)
	}
}

func (m model) View() string {
	if m.help {
		return m.viewHelp()
	}
	var b strings.Builder
	b.WriteString(m.bold("tunnel") + "  " + m.dim(routes.Zone) + "\n")
	state := "service stopped"
	if m.service {
		state = "service running"
	}
	b.WriteString(m.dim(state) + "\n\n")
	m.viewRoutes(&b)
	b.WriteString("\n")
	m.viewFooter(&b)
	if m.notice != "" {
		b.WriteString("\n" + m.notice)
	}
	b.WriteString("\n")
	return b.String()
}

func (m model) viewHelp() string {
	lines := []string{
		m.bold("tunnel"),
		"",
		"j/k    move",
		"a      add",
		"d      remove",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m model) viewRoutes(b *strings.Builder) {
	if len(m.file.Routes) == 0 {
		b.WriteString(m.dim("no routes") + "\n")
	}
	for i, route := range m.file.Routes {
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		local := "down"
		if m.local[route.Port] {
			local = "up"
		}
		line := fmt.Sprintf("%s%-8s  %-32s  %5d  %s", mark, route.Name, routes.Hostname(m.file.Zone, route.Name), route.Port, local)
		if i == m.cursor {
			line = m.bold(line)
		}
		b.WriteString(line + "\n")
	}
}

func (m model) viewFooter(b *strings.Builder) {
	switch m.mode {
	case modeAdd:
		b.WriteString("name  " + m.name.view(m.color) + "\n")
		b.WriteString("port  " + m.port.view(m.color) + "\n")
		b.WriteString(m.dim("host  "+m.preview()) + "\n")
		b.WriteString(m.dim("enter save    esc cancel"))
	case modeConfirm:
		b.WriteString("remove " + m.file.Routes[m.cursor].Name + "? y or n")
	}
}

func (m model) preview() string {
	raw := strings.TrimSpace(m.name.value)
	if raw == "" {
		return "…"
	}
	_, hostname, err := routes.Normalize(m.file.Zone, raw)
	if err != nil {
		return "…"
	}
	return hostname
}

func (m model) bold(s string) string {
	if !m.color {
		return s
	}
	return "\x1b[1m" + s + "\x1b[22m"
}

func (m model) dim(s string) string {
	if !m.color {
		return s
	}
	return "\x1b[38;5;245m" + s + "\x1b[39m"
}

func serviceUp() bool {
	return exec.Command("systemctl", "--user", "--quiet", "is-active", "tunnel.service").Run() == nil
}

func listening(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 150*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
