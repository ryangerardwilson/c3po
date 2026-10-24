package tui

import (
	"io"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/pass/internal/op"
)

type fakeSource struct {
	vaults []op.Vault
	items  []op.Item
	secret string
	reads  int
}

func (f *fakeSource) Vaults() ([]op.Vault, error) { return f.vaults, nil }

func (f *fakeSource) List(string) ([]op.Item, error) { return f.items, nil }

func (f *fakeSource) Read(string, string, string) (string, error) {
	f.reads++
	return f.secret, nil
}

type fakeClip struct{ text string }

func (f *fakeClip) Copy(text string, _ io.Writer) error {
	f.text = text
	return nil
}

func sample() (*model, *fakeSource, *fakeClip) {
	item := op.Item{ID: "item-1", Title: "GH_EXAMPLE", Category: "LOGIN"}
	item.Vault.ID = "vault-1"
	item.AdditionalInformation = "ada"
	item.Fields = []op.Field{{Label: "password", Value: "secret-value"}}
	src := &fakeSource{
		vaults: []op.Vault{{ID: "vault-1", Name: "Root"}},
		items:  []op.Item{item},
		secret: "secret-value",
	}
	clip := &fakeClip{}
	m := &model{src: src, clip: clip, width: 80, height: 24}
	m.loadVaults()
	return m, src, clip
}

func TestEnterOpensAVaultThenCopiesTheItem(t *testing.T) {
	m, src, clip := sample()
	if m.handleKey("enter") || m.vaultName != "Root" || src.reads != 0 {
		t.Fatalf("vault %q reads %d", m.vaultName, src.reads)
	}
	if m.handleKey("l") || src.reads != 0 || clip.text != "" {
		t.Fatal("l copied an item")
	}
	if m.handleKey("enter") || clip.text != "secret-value" || m.status != "copied GH_EXAMPLE" {
		t.Fatalf("%q copied %d bytes", m.status, len(clip.text))
	}
	if strings.Contains(m.view(), "secret-value") {
		t.Fatal("frame printed the secret")
	}
}

func TestFrameShowsTheUsernameAndHidesThePasswordField(t *testing.T) {
	m, _, _ := sample()
	m.handleKey("enter")
	frame := m.view()
	if !strings.Contains(frame, "GH_EXAMPLE") || !strings.Contains(frame, "ada") || strings.Contains(frame, "secret-value") {
		t.Fatal(frame)
	}
}

func TestHReturnsToTheVaultList(t *testing.T) {
	m, _, _ := sample()
	m.handleKey("enter")
	if m.handleKey("h") || m.vaultName != "" || len(m.rows) != 1 || !m.rows[0].vault {
		t.Fatalf("%q %#v", m.vaultName, m.rows)
	}
}

func TestShortcutsStayBehindQuestionMark(t *testing.T) {
	m, _, _ := sample()
	if strings.Contains(m.view(), "j/k move") || !strings.Contains(m.view(), "Root") {
		t.Fatalf("list\n%s", m.view())
	}
	if m.handleKey("?") || !strings.Contains(m.view(), "open") || !strings.Contains(m.view(), "copy") {
		t.Fatalf("help\n%s", m.view())
	}
	if m.handleKey("j") || m.cursor != 0 || !m.help {
		t.Fatalf("cursor %d", m.cursor)
	}
	if m.handleKey("esc") || m.help || !strings.Contains(m.view(), "Root") || strings.Contains(m.view(), "j/k move") {
		t.Fatalf("esc\n%s", m.view())
	}
	if m.handleKey("?") || !m.handleKey("q") {
		t.Fatal("q")
	}
}

func TestCopiedItemStaysOnScreenWhileJKMoves(t *testing.T) {
	items := make([]op.Item, 40)
	for i := range items {
		title := itemTitle(i)
		items[i] = op.Item{ID: "id-" + title[:7], Title: title, Category: "LOGIN"}
		items[i].Vault.ID = "vault-1"
	}
	src := &fakeSource{
		vaults: []op.Vault{{ID: "vault-1", Name: "Wiom"}},
		items:  items,
		secret: "secret-value",
	}
	m := &model{src: src, clip: &fakeClip{}, width: 84, height: 37}
	m.loadVaults()
	if m.handleKey("enter") || m.vaultName != "Wiom" {
		t.Fatalf("vault %q", m.vaultName)
	}
	if m.handleKey("enter") || !strings.HasPrefix(m.status, "copied ") {
		t.Fatal(m.status)
	}
	for i := 0; i < len(m.rows); i++ {
		if m.cursor != i {
			t.Fatalf("cursor %d want %d", m.cursor, i)
		}
		line := cursorLine(m.view())
		if !strings.Contains(line, m.rows[i].name[:7]) {
			t.Fatalf("row %d line %q", i, line)
		}
		assertFrame(t, m)
		if i+1 < len(m.rows) && m.handleKey("j") {
			t.Fatal("j quit")
		}
	}
	if m.handleKey("k") || m.cursor != len(m.rows)-2 {
		t.Fatalf("k cursor %d", m.cursor)
	}
	line := cursorLine(m.view())
	if !strings.Contains(line, m.rows[m.cursor].name[:7]) {
		t.Fatalf("k line %q", line)
	}
	assertFrame(t, m)
}

func TestPaintPinsEachLineToItsRow(t *testing.T) {
	frame := "pass - Wiom\n\n> ITEM\ncopied ITEM_00\n"
	got := paintBytes(frame, 37)
	if strings.Contains(got, "\n") {
		t.Fatal("paint wrote a newline")
	}
	for _, seq := range []string{"\x1b[?7l", "\x1b[1;1H", "\x1b[2;1H", "\x1b[4;1H", "\x1b[5;1H\x1b[J", "\x1b[?7h"} {
		if !strings.Contains(got, seq) {
			t.Fatalf("missing %q", seq)
		}
	}
}

func itemTitle(i int) string {
	return "ITEM_" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + "_" + strings.Repeat("LongTitle_", 8)
}

func assertFrame(t *testing.T, m *model) {
	t.Helper()
	frame := m.view()
	if strings.Contains(frame, "secret-value") {
		t.Fatal("frame printed the secret")
	}
	lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("%d lines in a %d-row terminal", len(lines), m.height)
	}
	for _, line := range lines {
		if n := len(stripANSI(line)); n >= m.width {
			t.Fatalf("line width %d", n)
		}
	}
}

func cursorLine(frame string) string {
	for _, line := range strings.Split(frame, "\n") {
		plain := stripANSI(line)
		if strings.HasPrefix(plain, "> ") {
			return plain
		}
	}
	return ""
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			continue
		}
		for i < len(s) && s[i] != 'm' {
			i++
		}
	}
	return b.String()
}
