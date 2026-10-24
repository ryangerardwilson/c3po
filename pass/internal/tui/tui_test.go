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
