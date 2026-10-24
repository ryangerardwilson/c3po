package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryangerardwilson/c3po/xl/internal/sheet"
)

func TestDDDeletesTheCurrentRow(t *testing.T) {
	s := sheet.Blank()
	s.Set(0, 0, "Rent")
	s.Set(1, 0, "10")
	s.Set(0, 1, "Food")
	s.Set(1, 1, "20")
	s.Set(1, 2, "=SUM(B1:B2)")
	m := ready(s)
	m = press(m, runeKey('d'))
	if m.flash != "d" || !m.deleteArmed || m.sheet.Source(0, 0) != "Rent" {
		t.Fatalf("arm flash %q armed %v %q", m.flash, m.deleteArmed, m.sheet.Source(0, 0))
	}
	m = press(m, runeKey('j'))
	if m.deleteArmed || m.row != 1 || m.sheet.Source(0, 0) != "Rent" || m.sheet.Source(0, 1) != "Food" {
		t.Fatalf("cancel row %d armed %v %q %q", m.row, m.deleteArmed, m.sheet.Source(0, 0), m.sheet.Source(0, 1))
	}
	m = press(m, runeKey('d'), tea.KeyMsg{Type: tea.KeyEsc})
	if m.deleteArmed || m.sheet.Source(0, 1) != "Food" {
		t.Fatalf("esc armed %v %q", m.deleteArmed, m.sheet.Source(0, 1))
	}
	m = press(m, runeKey('d'), runeKey('d'))
	if m.row != 1 || m.sheet.Source(0, 0) != "Rent" || m.sheet.Source(0, 1) != "" || m.sheet.Source(1, 1) != "=SUM(B1:B1)" || m.sheet.Display(1, 1) != "10" {
		t.Fatalf("deleted row %d %q %q sum %q display %s", m.row, m.sheet.Source(0, 0), m.sheet.Source(0, 1), m.sheet.Source(1, 1), m.sheet.Display(1, 1))
	}
	if m.deleteArmed {
		t.Fatal("arm stayed set")
	}

	m = press(m, runeKey('i'), runeKey('d'), tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal || m.sheet.Source(0, 1) != "d" || m.sheet.Source(0, 0) != "Rent" {
		t.Fatalf("typed mode %d %q rent %q", m.mode, m.sheet.Source(0, 1), m.sheet.Source(0, 0))
	}

	held := ready(sheet.Blank())
	held.sheet.Set(0, 0, "Rent")
	held = press(held, runeKey('v'), runeKey('d'), runeKey('d'))
	if held.mode != modeVisual || held.sheet.Source(0, 0) != "Rent" || held.deleteArmed {
		t.Fatalf("visual mode %d %q armed %v", held.mode, held.sheet.Source(0, 0), held.deleteArmed)
	}

	help := press(ready(sheet.Blank()), runeKey('?'))
	if !strings.Contains(help.View(), "delete the current row") {
		t.Fatalf("help:\n%s", help.View())
	}
}
