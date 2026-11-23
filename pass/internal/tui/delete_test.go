package tui

import (
	"errors"
	"strings"
	"testing"
)

func loginItem(id, title string) Item {
	return Item{ID: id, VaultID: "vault-1", Title: title, Category: "LOGIN", Username: "ada"}
}

func openVault(t *testing.T, items ...Item) (*model, *fakeSource) {
	t.Helper()
	src := &fakeSource{items: items}
	m := testModel(t, src)
	m.loadItems("Root")
	settle(t, m)
	return m, src
}

func TestDDDeletesTheSelectedItem(t *testing.T) {
	m, src := openVault(t, loginItem("a", "alpha"), loginItem("b", "bravo"), loginItem("c", "charlie"))
	m.cursor = 1
	if press(t, m, "d") || m.status != "d" || !m.armed {
		t.Fatalf("arm %v %q", m.armed, m.status)
	}
	if press(t, m, "d") || m.status != "deleted bravo" || m.cursor != 1 || len(m.rows) != 2 || m.rows[1].name != "charlie" {
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
	m, src := openVault(t, loginItem("a", "alpha"), loginItem("b", "bravo"))
	if press(t, m, "d") || press(t, m, "j") || m.cursor != 1 || m.armed || m.status != "" || len(src.deleted) != 0 {
		t.Fatalf("cursor %d armed %v %q %#v", m.cursor, m.armed, m.status, src.deleted)
	}
}

func TestEscCancelsDelete(t *testing.T) {
	m, src := openVault(t, loginItem("a", "alpha"))
	press(t, m, "d")
	if press(t, m, "esc") || m.armed || m.status != "" || len(src.deleted) != 0 {
		t.Fatalf("armed %v %q %#v", m.armed, m.status, src.deleted)
	}
}

func TestQAfterDQuitsWithoutDeleting(t *testing.T) {
	m, src := openVault(t, loginItem("a", "alpha"))
	if press(t, m, "d") || !press(t, m, "q") || len(src.deleted) != 0 {
		t.Fatalf("deleted %#v", src.deleted)
	}
}

func TestDDLeavesAVault(t *testing.T) {
	m, _ := sample(t)
	src := m.src.(*fakeSource)
	if press(t, m, "d") || press(t, m, "d") || m.status != "dd deletes an item" || len(src.deleted) != 0 || !m.rows[0].vault {
		t.Fatalf("%q %#v", m.status, src.deleted)
	}
}

func TestDDOnAnEmptyItemList(t *testing.T) {
	m, src := openVault(t)
	if press(t, m, "d") || press(t, m, "d") || m.status != "no items" || len(src.deleted) != 0 {
		t.Fatalf("%q %#v", m.status, src.deleted)
	}
}

func TestDDOnAnEmptyVaultList(t *testing.T) {
	src := &fakeSource{}
	m := testModel(t, src)
	m.loadVaults()
	settle(t, m)
	if press(t, m, "d") || press(t, m, "d") || m.status != "no vaults" || len(src.deleted) != 0 {
		t.Fatalf("%q %#v", m.status, src.deleted)
	}
}

func TestFailedDeleteLeavesTheItem(t *testing.T) {
	m, src := openVault(t, loginItem("a", "alpha"))
	src.deleteErr = errors.New("Unable to delete the 1Password item.")
	if press(t, m, "d") || press(t, m, "d") || m.status != src.deleteErr.Error() || len(m.rows) != 1 || m.rows[0].name != "alpha" {
		t.Fatalf("%q %#v", m.status, m.rows)
	}
	if strings.Contains(m.view(), "secret-value") {
		t.Fatal(m.view())
	}
}

func TestHelpIgnoresD(t *testing.T) {
	m, src := openVault(t, loginItem("a", "alpha"))
	if press(t, m, "?") || press(t, m, "d") || press(t, m, "d") || !m.help || len(src.deleted) != 0 || len(m.rows) != 1 {
		t.Fatalf("help %v %#v", m.help, src.deleted)
	}
}

func TestQuestionMarkClearsTheArm(t *testing.T) {
	m, src := openVault(t, loginItem("a", "alpha"))
	if press(t, m, "d") || press(t, m, "?") || m.armed || m.status != "" || !m.help || len(src.deleted) != 0 {
		t.Fatalf("armed %v help %v %q", m.armed, m.help, m.status)
	}
}
