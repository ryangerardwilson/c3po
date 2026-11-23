package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/tuijob"
)

type deletedCall struct {
	vault string
	id    string
}

type copyCall struct {
	vault string
	id    string
}

type fakeSource struct {
	vaults    []Vault
	items     []Item
	copies    []copyCall
	deleted   []deletedCall
	deleteErr error
}

func (f *fakeSource) Vaults(context.Context) ([]Vault, error) { return f.vaults, nil }

func (f *fakeSource) Items(context.Context, string) ([]Item, error) { return f.items, nil }

func (f *fakeSource) Copy(_ context.Context, vault, id string) error {
	f.copies = append(f.copies, copyCall{vault: vault, id: id})
	return nil
}

func (f *fakeSource) Delete(_ context.Context, vault, id string) error {
	f.deleted = append(f.deleted, deletedCall{vault: vault, id: id})
	if f.deleteErr != nil {
		return f.deleteErr
	}
	kept := make([]Item, 0, len(f.items))
	for _, item := range f.items {
		if item.ID != id {
			kept = append(kept, item)
		}
	}
	f.items = kept
	return nil
}

// testModel is a model with its worker, as the loop makes it.
func testModel(t *testing.T, src Source) *model {
	t.Helper()
	m := &model{src: src, width: 80, height: 24, jobs: tuijob.New()}
	t.Cleanup(m.jobs.Close)
	return m
}

// settle lets the queued nebula calls finish and applies their results, as
// the loop does before it reads a key from a terminal without a deadline.
func settle(t *testing.T, m *model) {
	t.Helper()
	if !m.jobs.Wait(time.Second) {
		t.Fatal("nebula calls did not finish")
	}
	m.jobs.Apply()
}

// press applies one key and settles what it queued.
func press(t *testing.T, m *model, key string) bool {
	t.Helper()
	quit := m.handleKey(key)
	settle(t, m)
	return quit
}

func sample(t *testing.T) (*model, *fakeSource) {
	t.Helper()
	item := Item{ID: "item-1", VaultID: "vault-1", Title: "GH_EXAMPLE", Category: "LOGIN", Username: "ada"}
	src := &fakeSource{
		vaults: []Vault{{ID: "vault-1", Name: "Root"}},
		items:  []Item{item},
	}
	m := testModel(t, src)
	m.loadVaults()
	settle(t, m)
	return m, src
}

func TestEnterOpensAVaultThenCopiesTheItem(t *testing.T) {
	m, src := sample(t)
	if press(t, m, "enter") || m.vaultName != "Root" || len(src.copies) != 0 {
		t.Fatalf("vault %q copies %d", m.vaultName, len(src.copies))
	}
	if press(t, m, "l") || len(src.copies) != 0 {
		t.Fatal("l copied an item")
	}
	if press(t, m, "enter") || len(src.copies) != 1 || src.copies[0].vault != "Root" || src.copies[0].id != "item-1" || m.status != "copied GH_EXAMPLE" {
		t.Fatalf("%q %#v", m.status, src.copies)
	}
	if strings.Contains(m.view(), "secret-value") {
		t.Fatal("frame printed the secret")
	}
}

func TestFrameShowsTheUsernameAndHidesThePasswordField(t *testing.T) {
	m, _ := sample(t)
	press(t, m, "enter")
	frame := m.view()
	if !strings.Contains(frame, "GH_EXAMPLE") || !strings.Contains(frame, "ada") || strings.Contains(frame, "secret-value") {
		t.Fatal(frame)
	}
}

func TestHReturnsToTheVaultList(t *testing.T) {
	m, _ := sample(t)
	press(t, m, "enter")
	if press(t, m, "h") || m.vaultName != "" || len(m.rows) != 1 || !m.rows[0].vault {
		t.Fatalf("%q %#v", m.vaultName, m.rows)
	}
}

func TestShortcutsStayBehindQuestionMark(t *testing.T) {
	m, _ := sample(t)
	if strings.Contains(m.view(), "j/k move") || !strings.Contains(m.view(), "Root") {
		t.Fatalf("list\n%s", m.view())
	}
	if press(t, m, "?") || !strings.Contains(m.view(), "open") || !strings.Contains(m.view(), "copy") || !strings.Contains(m.view(), "dd") {
		t.Fatalf("help\n%s", m.view())
	}
	if press(t, m, "j") || m.cursor != 0 || !m.help {
		t.Fatalf("cursor %d", m.cursor)
	}
	if press(t, m, "esc") || m.help || !strings.Contains(m.view(), "Root") || strings.Contains(m.view(), "j/k move") {
		t.Fatalf("esc\n%s", m.view())
	}
	if press(t, m, "?") || !press(t, m, "q") {
		t.Fatal("q")
	}
}

func TestCopiedItemStaysOnScreenWhileJKMoves(t *testing.T) {
	items := make([]Item, 40)
	for i := range items {
		title := itemTitle(i)
		items[i] = Item{ID: "id-" + title[:7], VaultID: "vault-1", Title: title, Category: "LOGIN"}
	}
	src := &fakeSource{
		vaults: []Vault{{ID: "vault-1", Name: "Wiom"}},
		items:  items,
	}
	m := testModel(t, src)
	m.width, m.height = 84, 37
	m.loadVaults()
	settle(t, m)
	if press(t, m, "enter") || m.vaultName != "Wiom" {
		t.Fatalf("vault %q", m.vaultName)
	}
	if press(t, m, "enter") || !strings.HasPrefix(m.status, "copied ") {
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
		if i+1 < len(m.rows) && press(t, m, "j") {
			t.Fatal("j quit")
		}
	}
	if press(t, m, "k") || m.cursor != len(m.rows)-2 {
		t.Fatalf("k cursor %d", m.cursor)
	}
	line := cursorLine(m.view())
	if !strings.Contains(line, m.rows[m.cursor].name[:7]) {
		t.Fatalf("k line %q", line)
	}
	assertFrame(t, m)
}

func TestJKWrapsTheItemList(t *testing.T) {
	m := &model{rows: []row{{name: "alpha"}, {name: "bravo"}}}
	if press(t, m, "k") || m.cursor != 1 {
		t.Fatalf("k %d", m.cursor)
	}
	if press(t, m, "j") || m.cursor != 0 {
		t.Fatalf("j %d", m.cursor)
	}
	if press(t, m, "G") || m.cursor != 1 || press(t, m, "g") || m.cursor != 0 {
		t.Fatalf("ends %d", m.cursor)
	}
	one := &model{rows: []row{{name: "only"}}}
	one.handleKey("j")
	one.handleKey("k")
	if one.cursor != 0 {
		t.Fatalf("one %d", one.cursor)
	}
	empty := &model{}
	empty.handleKey("j")
	empty.handleKey("k")
	if empty.cursor != 0 {
		t.Fatalf("empty %d", empty.cursor)
	}
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
