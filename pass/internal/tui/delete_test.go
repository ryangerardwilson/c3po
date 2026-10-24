package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/pass/internal/op"
)

func loginItem(id, title string) op.Item {
	item := op.Item{ID: id, Title: title, Category: "LOGIN"}
	item.Vault.ID = "vault-1"
	item.Fields = []op.Field{{Label: "password", Value: "secret-value"}}
	return item
}

func openVault(items ...op.Item) (*model, *fakeSource) {
	src := &fakeSource{items: items, secret: "secret-value"}
	m := &model{src: src, clip: &fakeClip{}, width: 80, height: 24}
	m.loadItems("Root")
	return m, src
}

func TestDDDeletesTheSelectedItem(t *testing.T) {
	m, src := openVault(loginItem("a", "alpha"), loginItem("b", "bravo"), loginItem("c", "charlie"))
	m.cursor = 1
	if m.handleKey("d") || m.status != "d" || !m.armed {
		t.Fatalf("arm %v %q", m.armed, m.status)
	}
	if m.handleKey("d") || m.status != "deleted bravo" || m.cursor != 1 || len(m.rows) != 2 || m.rows[1].name != "charlie" {
		t.Fatalf("status %q cursor %d rows %#v", m.status, m.cursor, m.rows)
	}
	if len(src.deleted) != 1 || src.deleted[0].vault != "Root" || src.deleted[0].id != "b" {
		t.Fatalf("%#v", src.deleted)
	}
	if strings.Contains(m.view(), "secret-value") {
		t.Fatal(m.view())
	}
}

func TestDThenMotionDoesNotDelete(t *testing.T) {
	m, src := openVault(loginItem("a", "alpha"), loginItem("b", "bravo"))
	if m.handleKey("d") || m.handleKey("j") || m.cursor != 1 || m.armed || m.status != "" || len(src.deleted) != 0 {
		t.Fatalf("cursor %d armed %v %q %#v", m.cursor, m.armed, m.status, src.deleted)
	}
}

func TestEscCancelsDelete(t *testing.T) {
	m, src := openVault(loginItem("a", "alpha"))
	m.handleKey("d")
	if m.handleKey("esc") || m.armed || m.status != "" || len(src.deleted) != 0 {
		t.Fatalf("armed %v %q %#v", m.armed, m.status, src.deleted)
	}
}

func TestQAfterDQuitsWithoutDeleting(t *testing.T) {
	m, src := openVault(loginItem("a", "alpha"))
	if m.handleKey("d") || !m.handleKey("q") || len(src.deleted) != 0 {
		t.Fatalf("deleted %#v", src.deleted)
	}
}

func TestDDLeavesAVault(t *testing.T) {
	m, _, _ := sample()
	src := m.src.(*fakeSource)
	if m.handleKey("d") || m.handleKey("d") || m.status != "dd deletes an item" || len(src.deleted) != 0 || !m.rows[0].vault {
		t.Fatalf("%q %#v", m.status, src.deleted)
	}
}

func TestDDOnAnEmptyItemList(t *testing.T) {
	m, src := openVault()
	if m.handleKey("d") || m.handleKey("d") || m.status != "no items" || len(src.deleted) != 0 {
		t.Fatalf("%q %#v", m.status, src.deleted)
	}
}

func TestDDOnAnEmptyVaultList(t *testing.T) {
	src := &fakeSource{}
	m := &model{src: src, clip: &fakeClip{}, width: 80, height: 24}
	m.loadVaults()
	if m.handleKey("d") || m.handleKey("d") || m.status != "no vaults" || len(src.deleted) != 0 {
		t.Fatalf("%q %#v", m.status, src.deleted)
	}
}

func TestFailedDeleteLeavesTheItem(t *testing.T) {
	m, src := openVault(loginItem("a", "alpha"))
	src.deleteErr = errors.New("Unable to delete the 1Password item.")
	if m.handleKey("d") || m.handleKey("d") || m.status != src.deleteErr.Error() || len(m.rows) != 1 || m.rows[0].name != "alpha" {
		t.Fatalf("%q %#v", m.status, m.rows)
	}
	if strings.Contains(m.view(), "secret-value") {
		t.Fatal(m.view())
	}
}

func TestHelpIgnoresD(t *testing.T) {
	m, src := openVault(loginItem("a", "alpha"))
	if m.handleKey("?") || m.handleKey("d") || m.handleKey("d") || !m.help || len(src.deleted) != 0 || len(m.rows) != 1 {
		t.Fatalf("help %v %#v", m.help, src.deleted)
	}
}

func TestQuestionMarkClearsTheArm(t *testing.T) {
	m, src := openVault(loginItem("a", "alpha"))
	if m.handleKey("d") || m.handleKey("?") || m.armed || m.status != "" || !m.help || len(src.deleted) != 0 {
		t.Fatalf("armed %v help %v %q", m.armed, m.help, m.status)
	}
}
