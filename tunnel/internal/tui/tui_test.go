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
	for _, want := range []string{"lab", "lab.ryangerardwilson.com", "9090", "up", "service running", "a add"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q\n%s", want, view)
		}
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
