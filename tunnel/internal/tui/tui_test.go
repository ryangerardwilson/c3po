package tui

import (
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
)

func TestViewListsRoute(t *testing.T) {
	m := model{
		file: routes.File{Zone: routes.Zone, Routes: []routes.Route{{Name: "lab", Port: 9090}}},
		mode: "list",
		local: map[int]bool{
			9090: true,
		},
		service: true,
	}
	view := m.View()
	for _, want := range []string{"lab", "lab.ryangerardwilson.com", "9090", "up", "service running"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q\n%s", want, view)
		}
	}
	if strings.Contains(view, "a add") {
		t.Fatalf("legend on the list\n%s", view)
	}
	if strings.Contains(view, "\x1b") {
		t.Fatalf("plain view has escapes\n%q", view)
	}
}

func TestAddModeTypesAndPreviews(t *testing.T) {
	m := newModel(t.TempDir(), routes.Default(routes.Zone), false)
	m.key("a")
	if m.mode != modeAdd || !m.name.focused {
		t.Fatalf("mode %s", m.mode)
	}
	for _, k := range []string{"q", "x", "backspace", "N", "o", "t", "e", "s"} {
		if m.key(k) != actNone {
			t.Fatalf("key %q acted", k)
		}
	}
	if m.name.value != "qNotes" {
		t.Fatalf("name %q", m.name.value)
	}
	m.key("ctrl+u")
	for _, k := range []string{"n", "o", "t", "e", "s"} {
		m.key(k)
	}
	if !strings.Contains(m.View(), "host  notes.ryangerardwilson.com") {
		t.Fatalf("preview\n%s", m.View())
	}
	if m.key("enter") != actNone || !m.port.focused || m.name.focused {
		t.Fatal("enter with empty port should move to port")
	}
	for _, k := range []string{"3", "0", "0", "0", "0", "0"} {
		m.key(k)
	}
	if m.port.value != "30000" {
		t.Fatalf("port limit %q", m.port.value)
	}
	m.key("tab")
	if !m.name.focused || m.port.focused {
		t.Fatal("tab did not move focus")
	}
	if m.key("enter") != actSave || m.notice != "saving" {
		t.Fatal("enter with a port should save")
	}
	m.key("esc")
	if m.mode != modeList {
		t.Fatal("esc did not leave add")
	}
	if m.key("q") != actQuit {
		t.Fatal("q did not quit the list")
	}
}

func TestConfirmRemovesRoute(t *testing.T) {
	home := t.TempDir()
	file, _, err := routes.Upsert(routes.Default(routes.Zone), "lab", 9090)
	if err != nil {
		t.Fatal(err)
	}
	if err := routes.Save(routes.Path(home), file); err != nil {
		t.Fatal(err)
	}
	m := newModel(home, file, false)
	m.key("j")
	m.key("d")
	if m.mode != modeConfirm || !strings.Contains(m.View(), "remove lab? y or n") {
		t.Fatalf("confirm\n%s", m.View())
	}
	m.key("n")
	if m.mode != modeList || len(m.file.Routes) != 2 {
		t.Fatal("n removed the route")
	}
	m.key("d")
	m.key("y")
	if m.notice != "removed lab" || len(m.file.Routes) != 1 || m.cursor != 0 {
		t.Fatalf("remove %q %+v %d", m.notice, m.file, m.cursor)
	}
	saved, err := routes.Load(routes.Path(home))
	if err != nil || len(saved.Routes) != 1 || saved.Routes[0].Name != routes.Apex {
		t.Fatalf("saved %+v %v", saved, err)
	}
}

func TestDecodeKeys(t *testing.T) {
	cases := map[string]string{
		decode(0x1b, ""):   "esc",
		decode(0x1b, "[A"): "up",
		decode(0x1b, "OB"): "down",
		decode(0x1b, "[Z"): "shift+tab",
		decode('\r', ""):   "enter",
		decode('\t', ""):   "tab",
		decode(0x7f, ""):   "backspace",
		decode(3, ""):      "ctrl+c",
		decode('j', ""):    "j",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("decode %q want %q", got, want)
		}
	}
}

func TestShortcutsStayBehindQuestionMark(t *testing.T) {
	m := model{
		file: routes.File{Zone: routes.Zone, Routes: []routes.Route{
			{Name: "lab", Port: 9090},
			{Name: "notes", Port: 9091},
		}},
		mode: modeList,
	}
	if strings.Contains(m.View(), "a add") || !strings.Contains(m.View(), "lab") {
		t.Fatalf("list\n%s", m.View())
	}
	if m.key("?") != actNone || !strings.Contains(m.View(), "add") || !strings.Contains(m.View(), "remove") || !strings.Contains(m.View(), "quit") {
		t.Fatalf("help\n%s", m.View())
	}
	if m.key("j") != actNone || m.cursor != 0 || !m.help {
		t.Fatalf("cursor %d", m.cursor)
	}
	if m.key("esc") != actNone || m.help || !strings.Contains(m.View(), "lab") || strings.Contains(m.View(), "a add") {
		t.Fatalf("esc\n%s", m.View())
	}
	m.key("?")
	if m.key("q") != actQuit {
		t.Fatal("q")
	}
	add := newModel(t.TempDir(), routes.File{Zone: routes.Zone, Routes: []routes.Route{{Name: "lab", Port: 9090}}}, false)
	add.key("a")
	if add.mode != modeAdd || !strings.Contains(add.View(), "enter save") || !strings.Contains(add.View(), "esc cancel") {
		t.Fatalf("add\n%s", add.View())
	}
	add.key("esc")
	add.key("d")
	if !strings.Contains(add.View(), "remove lab? y or n") {
		t.Fatalf("confirm\n%s", add.View())
	}
}
