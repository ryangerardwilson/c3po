package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryangerardwilson/c3po/xl/internal/sheet"
)

func TestHJLKMovesInNormalAndTypesInInsert(t *testing.T) {
	m := ready(sheet.Blank())
	m = press(m, runeKey('l'), runeKey('j'))
	if m.col != 1 || m.row != 1 || m.mode != modeNormal {
		t.Fatalf("cursor %d,%d mode %d", m.col, m.row, m.mode)
	}
	if m.sheet.Source(0, 0) != "" || m.sheet.Source(1, 1) != "" {
		t.Fatal("hjkl wrote into a cell")
	}
	m = press(m, runeKey('h'), runeKey('k'))
	if m.col != 0 || m.row != 0 {
		t.Fatalf("back to origin, at %d,%d", m.col, m.row)
	}

	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeInsert || m.row != 0 || m.col != 0 || m.sheet.Source(0, 0) != "" {
		t.Fatalf("enter mode %d at %d,%d source %q", m.mode, m.col, m.row, m.sheet.Source(0, 0))
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc}, runeKey('i'))
	if m.mode != modeInsert {
		t.Fatal("i did not enter insert")
	}
	m = press(m, runeKey('h'), runeKey('j'))
	if string(m.edit) != "hj" || m.sheet.Source(0, 0) != "" {
		t.Fatalf("insert buffer %q source %q", string(m.edit), m.sheet.Source(0, 0))
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal {
		t.Fatal("esc did not return to normal")
	}
	if m.sheet.Source(0, 0) != "hj" || m.col != 0 || m.row != 0 {
		t.Fatalf("esc source %q at %d,%d", m.sheet.Source(0, 0), m.col, m.row)
	}
}

func TestEnterMovesDownAndLeavesInsert(t *testing.T) {
	m := ready(sheet.Blank())
	m = press(m, runeKey('i'), runes("Rent"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.sheet.Source(0, 0) != "Rent" || m.sheet.Source(0, 1) != "" {
		t.Fatalf("A1 %q A2 %q", m.sheet.Source(0, 0), m.sheet.Source(0, 1))
	}
	if m.mode != modeNormal || m.row != 1 || m.col != 0 {
		t.Fatalf("mode %d at %d,%d", m.mode, m.col, m.row)
	}
	m = press(m, runeKey('l'))
	if m.col != 1 || m.sheet.Source(0, 1) != "" {
		t.Fatalf("l typed into the cell, col %d A2 %q", m.col, m.sheet.Source(0, 1))
	}
}

func TestFormulaAndSelectionStats(t *testing.T) {
	s := sheet.Blank()
	s.Set(1, 0, "1200")
	s.Set(1, 1, "300")
	m := ready(s)
	m = press(m, runeKey('l'), runeKey('j'), runeKey('j'), runeKey('i'), runes("=SUM(B1:B2)"))
	if m.mode != modeInsert || m.sheet.Source(1, 2) != "" || !strings.Contains(m.View(), "|=SUM(B1:B2)") {
		t.Fatalf("live edit source %q\n%s", m.sheet.Source(1, 2), m.View())
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.sheet.Display(1, 2) != "1500" {
		t.Fatalf("display %s source %s", m.sheet.Display(1, 2), m.sheet.Source(1, 2))
	}
	if strings.Contains(m.View(), "Average") {
		t.Fatalf("stats without a selection:\n%s", m.View())
	}
	m = press(m, runeKey('k'), runeKey('k'), runeKey('v'), runeKey('j'))
	view := m.View()
	if !strings.Contains(view, "B1:B2") || !strings.Contains(view, "Sum 1,500") || !strings.Contains(view, "Average 750") || !strings.Contains(view, "Median 750") || !strings.Contains(view, "Mode none") {
		t.Fatalf("stats missing:\n%s", view)
	}
	if m.mode != modeVisual {
		t.Fatal("selection left visual mode")
	}
	m = press(m, runeKey('v'), runeKey('h'), runeKey('k'), runeKey('i'), runes("Rent"), tea.KeyMsg{Type: tea.KeyEsc}, runeKey('v'), runeKey('l'))
	if strings.Contains(m.View(), "Average") {
		t.Fatalf("stats on text:\n%s", m.View())
	}
}

func TestTypingCursorSitsOnTheShade(t *testing.T) {
	m := ready(sheet.Blank())
	m.color = true
	m = press(m, runeKey('i'), runes("Rent"))
	view := m.View()
	if m.sheet.Source(0, 0) != "" || !strings.Contains(view, "Rent") {
		t.Fatalf("source %q\n%s", m.sheet.Source(0, 0), view)
	}
	focus := viewLine(view, 1)
	if strings.Contains(view, "\033[1;7m") || !shadedCell(focus) || !strings.Contains(focus, ansiCursor) || !strings.Contains(focus, "Rent") {
		t.Fatalf("focus style:\n%s", view)
	}
	if strings.Contains(view, "[Sheet1]") || !strings.Contains(view, ansiCyan+"Sheet1!A1") || !strings.Contains(view, ansiYellow) {
		t.Fatalf("chrome color:\n%s", view)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc}, runeKey('v'), runeKey('j'))
	view = m.View()
	if !strings.Contains(view, ansiShade) || !strings.Contains(view, ansiSelect) {
		t.Fatalf("selection color:\n%s", view)
	}
}

func TestQuitDirtyThenDiscard(t *testing.T) {
	m := ready(sheet.Blank())
	m = press(m, runeKey('i'), runes("Rent"), tea.KeyMsg{Type: tea.KeyEsc}, runeKey('q'))
	if m.mode != modeQuit {
		t.Fatal("q did not ask")
	}
	next, cmd := m.Update(runeKey('n'))
	if cmd == nil {
		t.Fatal("discard did not quit")
	}
	if next.(*model).sheet.Dirty == false {
		t.Fatal("discard saved")
	}
}

func TestEnterFromNormalEditsThisCell(t *testing.T) {
	s := sheet.Blank()
	s.Set(0, 0, "Rent")
	m := ready(s)
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeInsert || m.row != 0 || m.col != 0 || string(m.edit) != "Rent" || m.sheet.Source(0, 0) != "Rent" {
		t.Fatalf("normal enter mode %d at %d,%d edit %q source %q", m.mode, m.col, m.row, string(m.edit), m.sheet.Source(0, 0))
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc}, runeKey('v'), runeKey('j'), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeNormal || m.row != 2 || m.anchorR != 2 || m.anchorC != 0 {
		t.Fatalf("visual enter mode %d row %d anchor %d,%d", m.mode, m.row, m.anchorC, m.anchorR)
	}
	m = press(m, runeKey('?'), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeHelp || m.row != 2 {
		t.Fatalf("help enter mode %d row %d", m.mode, m.row)
	}
}

func TestSheetsSwitchCreateAndRename(t *testing.T) {
	m := ready(sheet.Blank())
	m = press(m, runeKey('l'), runeKey('i'), runes("10"), tea.KeyMsg{Type: tea.KeyEsc})
	if m.sheet.Name != "Sheet1" || m.sheet.Source(1, 0) != "10" || m.col != 1 || m.row != 0 || m.mode != modeNormal {
		t.Fatalf("sheet1 %s %q at %d,%d mode %d", m.sheet.Name, m.sheet.Source(1, 0), m.col, m.row, m.mode)
	}
	m = press(m, runeKey('n'))
	if m.sheet.Name != "Sheet2" || m.col != 0 || m.row != 0 || m.mode != modeNormal {
		t.Fatalf("new %s at %d,%d mode %d", m.sheet.Name, m.col, m.row, m.mode)
	}
	m = press(m, runeKey('i'), runes("=Sheet1!B1"), tea.KeyMsg{Type: tea.KeyEsc})
	if m.sheet.Display(0, 0) != "10" || m.mode != modeNormal || m.row != 0 {
		t.Fatalf("display %s mode %d row %d source %q", m.sheet.Display(0, 0), m.mode, m.row, m.sheet.Source(0, 0))
	}
	view := m.View()
	if strings.Contains(view, "[Sheet2]") || !strings.Contains(view, "Sheet2!A1") || !strings.Contains(view, "=Sheet1!B1") {
		t.Fatalf("chrome:\n%s", view)
	}
	if strings.Contains(view, "hjkl move") || strings.Contains(view, "[ ] sheet") || strings.Contains(view, "csvking") {
		t.Fatalf("shortcuts on the grid:\n%s", view)
	}
	m = press(m, runeKey('['))
	if m.sheet.Name != "Sheet1" || m.col != 1 || m.row != 0 || m.sheet.Source(1, 0) != "10" {
		t.Fatalf("back %s at %d,%d %q", m.sheet.Name, m.col, m.row, m.sheet.Source(1, 0))
	}
	m = press(m, runeKey(']'))
	if m.sheet.Name != "Sheet2" || m.col != 0 || m.row != 0 || m.sheet.Display(0, 0) != "10" {
		t.Fatalf("forward %s at %d,%d display %s", m.sheet.Name, m.col, m.row, m.sheet.Display(0, 0))
	}
	m = press(m, runeKey('R'), tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal || m.sheet.Name != "Sheet2" || m.col != 0 || m.row != 0 {
		t.Fatalf("cancel %s mode %d at %d,%d", m.sheet.Name, m.mode, m.col, m.row)
	}
	m = press(m, runeKey('R'))
	if m.mode != modeRename {
		t.Fatal("rename did not open")
	}
	for range 6 {
		m = press(m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m = press(m, runes("Income"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeNormal || m.sheet.Name != "Income" || m.col != 0 || m.row != 0 {
		t.Fatalf("renamed %s mode %d at %d,%d", m.sheet.Name, m.mode, m.col, m.row)
	}
	if m.sheet.Display(0, 0) != "10" || m.sheet.Source(0, 0) != "=Sheet1!B1" {
		t.Fatalf("after rename display %s source %q", m.sheet.Display(0, 0), m.sheet.Source(0, 0))
	}
	view = m.View()
	if strings.Contains(view, "[Income]") || !strings.Contains(view, "Income!A1") {
		t.Fatalf("renamed chrome:\n%s", view)
	}
	m = press(m, runeKey('?'))
	help := m.View()
	if m.mode != modeHelp || m.row != 0 || !strings.Contains(help, "shortcuts") || !strings.Contains(help, "new sheet") || !strings.Contains(help, "=Income!B1") || !strings.Contains(help, "? or esc") {
		t.Fatalf("help mode %d row %d\n%s", m.mode, m.row, help)
	}
	m = press(m, runeKey('?'))
	if m.mode != modeNormal || m.row != 0 || strings.Contains(m.View(), "shortcuts") {
		t.Fatalf("help stayed open mode %d\n%s", m.mode, m.View())
	}
}

func TestFocusRowIsTheSameHeight(t *testing.T) {
	m := ready(sheet.Blank())
	m.color = true
	first := m.View()
	row1 := rowLine(first, 1)
	row2 := rowLine(first, 2)
	if row1 < 0 || row2 != row1+1 {
		t.Fatalf("packed rows %d %d\n%s", row1, row2, stripANSI(first))
	}
	if !shadedCell(viewLine(first, 1)) {
		t.Fatalf("shade\n%s", viewLine(first, 1))
	}
	m = press(m, runeKey('j'), runeKey('i'), runes("Rent"))
	second := m.View()
	if rowLine(second, 1) != row1 || rowLine(second, 2) != row2 {
		t.Fatalf("focus moved the rows to %d %d", rowLine(second, 1), rowLine(second, 2))
	}
	if len(strings.Split(first, "\n")) != len(strings.Split(second, "\n")) {
		t.Fatalf("lines %d %d", len(strings.Split(first, "\n")), len(strings.Split(second, "\n")))
	}
	focus := viewLine(second, 2)
	if !shadedCell(focus) || !strings.Contains(focus, "Rent") || !strings.Contains(focus, ansiCursor) {
		t.Fatalf("edit shade\n%s", focus)
	}
	plain := ready(sheet.Blank())
	plain = press(plain, runeKey('i'), runes("Rent"))
	text := stripANSI(plain.View())
	if rowLine(plain.View(), 2) != rowLine(plain.View(), 1)+1 || !strings.Contains(text, "|Rent") || strings.Contains(text, "┌") {
		t.Fatalf("plain\n%s", text)
	}
}

func viewLine(view string, row int) string {
	n := rowLine(view, row)
	lines := strings.Split(view, "\n")
	if n < 0 || n >= len(lines) {
		return ""
	}
	return lines[n]
}

func rowLine(view string, n int) int {
	prefix := fmt.Sprintf("%2d ", n)
	for i, line := range strings.Split(stripANSI(view), "\n") {
		if strings.HasPrefix(line, prefix) {
			return i
		}
	}
	return -1
}

func shadedCell(line string) bool {
	if ansiShade == ansiCursor || !strings.Contains(ansiShade, "38;2;0;0;0") || !strings.Contains(ansiCursor, "38;2;255;255;255") || !strings.Contains(ansiCursor, "48;2;255;70;70") {
		return false
	}
	shadeBg := bgOf(ansiShade)
	cursorBg := bgOf(ansiCursor)
	if shadeBg == "" || cursorBg == "" || shadeBg == cursorBg {
		return false
	}
	start := strings.Index(line, ansiShade)
	if start < 0 {
		return false
	}
	rest := line[start+len(ansiShade):]
	end := strings.Index(rest, "\033[0m")
	if end < 0 || strings.Contains(rest[:end], "\033[0m") {
		return false
	}
	mid := rest[:end]
	if strings.Contains(mid, "│") || strings.Contains(mid, "\033[96") || strings.Contains(mid, "\033[4") || strings.Contains(mid, "\033[53") {
		return false
	}
	at := strings.Index(mid, ansiCursor)
	if at < 0 {
		return true
	}
	return strings.Contains(mid[at+len(ansiCursor):], ansiShade)
}

func bgOf(code string) string {
	const mark = "48;2;"
	i := strings.Index(code, mark)
	if i < 0 {
		return ""
	}
	return strings.TrimSuffix(code[i+len(mark):], "m")
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestEmacsKeysEditTheCell(t *testing.T) {
	m := ready(sheet.Blank())
	m = press(m, runeKey('i'), runes("hello world"))
	if string(m.edit) != "hello world" || m.cur != 11 || m.mode != modeInsert {
		t.Fatalf("setup %q cur %d mode %d", string(m.edit), m.cur, m.mode)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	if m.cur != 0 || string(m.edit) != "hello world" {
		t.Fatalf("ctrl-a %q cur %d", string(m.edit), m.cur)
	}
	altF := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true}
	altB := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true}
	m = press(m, altF)
	if m.cur != 5 || string(m.edit) != "hello world" {
		t.Fatalf("alt-f %q cur %d", string(m.edit), m.cur)
	}
	m = press(m, altF)
	if m.cur != 11 || string(m.edit) != "hello world" {
		t.Fatalf("alt-f again %q cur %d", string(m.edit), m.cur)
	}
	m = press(m, altB)
	if m.cur != 6 || string(m.edit) != "hello world" {
		t.Fatalf("alt-b %q cur %d", string(m.edit), m.cur)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	if m.cur != 11 {
		t.Fatalf("ctrl-e cur %d", m.cur)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlW})
	if string(m.edit) != "hello " || m.cur != 6 || m.mode != modeInsert {
		t.Fatalf("ctrl-w %q cur %d mode %d", string(m.edit), m.cur, m.mode)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlA}, altF)
	if string(m.edit) != "hello " || m.cur != 5 {
		t.Fatalf("alt-f did not stay a motion %q cur %d", string(m.edit), m.cur)
	}
}

func TestColonWWritesAndWQQuits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget.csv")
	s := sheet.Blank()
	s.Path = path
	m := ready(s)
	m = press(m, runeKey('i'), runes("=1+2"), tea.KeyMsg{Type: tea.KeyEsc}, runeKey(':'), runeKey('w'), tea.KeyMsg{Type: tea.KeyEnter})
	if filepath.Base(m.sheet.Path) != "budget.xlsx" {
		t.Fatalf("path %s", m.sheet.Path)
	}
	loaded, err := sheet.Open(m.sheet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Source(0, 0) != "=1+2" {
		t.Fatalf("saved source %q", loaded.Source(0, 0))
	}
	if m.sheet.Dirty {
		t.Fatal("still dirty")
	}
	if !strings.Contains(m.View(), "saved budget.xlsx") {
		t.Fatalf("view:\n%s", m.View())
	}
	m.sheet.Set(0, 1, "Food")
	next, cmd := pressCmd(m, runeKey(':'), runes("wq"), tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal(":wq did not quit")
	}
	loaded, err = sheet.Open(next.sheet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Source(0, 1) != "Food" || next.sheet.Dirty {
		t.Fatalf("wq source %q dirty %v", loaded.Source(0, 1), next.sheet.Dirty)
	}
}

func TestColonWWithoutPathAsksThenQuits(t *testing.T) {
	m := ready(sheet.Blank())
	m = press(m, runeKey('i'), runes("Rent"), tea.KeyMsg{Type: tea.KeyEsc}, runeKey(':'), runes("wq"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeSave || !m.quitAfterSave || m.sheet.Path != "" {
		t.Fatalf("mode %d quit %v path %q", m.mode, m.quitAfterSave, m.sheet.Path)
	}
	if !strings.Contains(m.View(), "save as:") {
		t.Fatalf("view:\n%s", m.View())
	}
	path := filepath.Join(t.TempDir(), "budget.xlsx")
	next, cmd := pressCmd(m, runes(path), tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("save as did not quit")
	}
	loaded, err := sheet.Open(next.sheet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Source(0, 0) != "Rent" || filepath.Base(next.sheet.Path) != "budget.xlsx" {
		t.Fatalf("path %s source %q", next.sheet.Path, loaded.Source(0, 0))
	}
}

func TestCommandBarCancelUnknownAndQuit(t *testing.T) {
	m := ready(sheet.Blank())
	bar := press(ready(sheet.Blank()), runeKey(':'))
	if bar.mode != modeCommand || !strings.Contains(bar.View(), ":|") || !strings.Contains(bar.View(), "cmd") {
		t.Fatalf("command bar mode %d\n%s", bar.mode, bar.View())
	}
	m = press(m, runeKey('i'), runes("Rent"), tea.KeyMsg{Type: tea.KeyEsc}, runeKey(':'), runeKey('w'), tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal || !m.sheet.Dirty || m.sheet.Path != "" {
		t.Fatalf("esc mode %d dirty %v path %q", m.mode, m.sheet.Dirty, m.sheet.Path)
	}
	m = press(m, runeKey(':'), runes("zz"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeNormal || m.flash != "unknown command" || !m.sheet.Dirty {
		t.Fatalf("unknown mode %d flash %q dirty %v", m.mode, m.flash, m.sheet.Dirty)
	}
	m = press(m, runeKey(':'), runeKey('q'), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeQuit {
		t.Fatalf(":q mode %d", m.mode)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc}, runeKey('i'), runes("x"), tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.mode != modeInsert || m.sheet.Source(0, 0) != "Rent" || string(m.edit) != "Rentx" {
		t.Fatalf("ctrl-s mode %d source %q edit %q", m.mode, m.sheet.Source(0, 0), string(m.edit))
	}
	help := press(ready(sheet.Blank()), runeKey('?'))
	view := help.View()
	if strings.Contains(view, "ctrl-s") || !strings.Contains(view, ":w :wq :q") || !strings.Contains(view, "ctrl-n/p") || !strings.Contains(view, "=Income!B1") {
		t.Fatalf("help:\n%s", view)
	}
}

func TestSearchStepsWithCtrlNAndCtrlP(t *testing.T) {
	s := sheet.Blank()
	s.Set(0, 0, "Rent")
	s.Set(0, 1, "Food")
	s.Set(0, 2, "rent extra")
	m := ready(s)
	bar := press(m, runeKey('/'))
	if bar.mode != modeSearch || !strings.Contains(bar.View(), "/|") || !strings.Contains(bar.View(), "search") {
		t.Fatalf("search bar mode %d\n%s", bar.mode, bar.View())
	}
	none := press(m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if none.col != 0 || none.row != 0 || none.flash != "no search" {
		t.Fatalf("bare ctrl-n at %d,%d flash %q", none.col, none.row, none.flash)
	}
	m = press(m, runeKey('/'), tea.KeyMsg{Type: tea.KeyEnter})
	if m.flash != "no search" || m.row != 0 || m.mode != modeNormal {
		t.Fatalf("empty search mode %d row %d flash %q", m.mode, m.row, m.flash)
	}
	m = press(m, runeKey('/'), runes("rent"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeNormal || m.col != 0 || m.row != 2 || m.flash != "A3  2/2" {
		t.Fatalf("first match at %d,%d mode %d flash %q", m.col, m.row, m.mode, m.flash)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if m.row != 0 || m.col != 0 || m.flash != "A1  1/2" {
		t.Fatalf("wrap next at %d,%d flash %q", m.col, m.row, m.flash)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlP})
	if m.row != 2 || m.col != 0 || m.flash != "A3  2/2" {
		t.Fatalf("prev at %d,%d flash %q", m.col, m.row, m.flash)
	}
	m = press(m, runeKey('/'), runes("food"), tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal || m.row != 2 || m.query != "rent" {
		t.Fatalf("esc search mode %d row %d query %q", m.mode, m.row, m.query)
	}
	m = press(m, runeKey('/'), tea.KeyMsg{Type: tea.KeyCtrlN})
	if m.row != 0 || m.col != 0 || m.query != "rent" {
		t.Fatalf("repeat next row %d query %q", m.row, m.query)
	}
	miss := press(m, runeKey('/'), runes("nope"), tea.KeyMsg{Type: tea.KeyEnter})
	if miss.row != 0 || miss.col != 0 || miss.flash != "no match" || miss.mode != modeNormal {
		t.Fatalf("miss at %d,%d mode %d flash %q", miss.col, miss.row, miss.mode, miss.flash)
	}
}

func TestSearchFindsFormulaTextAndValue(t *testing.T) {
	s := sheet.Blank()
	s.Set(1, 0, "10")
	s.Set(1, 1, "5")
	s.Set(1, 2, "=SUM(B1:B2)")
	m := ready(s)
	m = press(m, runeKey('/'), runes("15"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.col != 1 || m.row != 2 || m.sheet.Display(1, 2) != "15" {
		t.Fatalf("value at %d,%d display %q", m.col, m.row, m.sheet.Display(1, 2))
	}
	m = press(m, runeKey('/'), runes("sum"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.col != 1 || m.row != 2 || m.flash != "B3  1/1" {
		t.Fatalf("formula at %d,%d flash %q", m.col, m.row, m.flash)
	}
	m = press(m, runeKey(':'), runes("w "+filepath.Join(t.TempDir(), "named.xlsx")), tea.KeyMsg{Type: tea.KeyEnter})
	if m.sheet.Dirty || filepath.Base(m.sheet.Path) != "named.xlsx" {
		t.Fatalf("path %s dirty %v", m.sheet.Path, m.sheet.Dirty)
	}
	loaded, err := sheet.Open(m.sheet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Source(1, 2) != "=SUM(B1:B2)" {
		t.Fatalf("saved %q", loaded.Source(1, 2))
	}
}

func TestOAndOInsertRows(t *testing.T) {
	s := sheet.Blank()
	s.Set(0, 0, "Rent")
	s.Set(1, 0, "10")
	s.Set(0, 1, "Food")
	s.Set(1, 1, "20")
	s.Set(1, 2, "=SUM(B1:B2)")
	m := ready(s)
	m = press(m, runeKey('o'))
	if m.mode != modeNormal || m.row != 1 || m.col != 0 {
		t.Fatalf("o mode %d at %d,%d", m.mode, m.col, m.row)
	}
	if m.sheet.Source(0, 0) != "Rent" || m.sheet.Source(0, 1) != "" || m.sheet.Source(0, 2) != "Food" {
		t.Fatalf("o rows %q %q %q", m.sheet.Source(0, 0), m.sheet.Source(0, 1), m.sheet.Source(0, 2))
	}
	if m.sheet.Source(1, 3) != "=SUM(B1:B3)" || m.sheet.Display(1, 3) != "30" {
		t.Fatalf("o sum %q display %s", m.sheet.Source(1, 3), m.sheet.Display(1, 3))
	}
	m = press(m, runeKey('O'))
	if m.mode != modeNormal || m.row != 1 || m.sheet.Source(0, 3) != "Food" || m.sheet.Source(1, 4) != "=SUM(B1:B4)" {
		t.Fatalf("O row %d food %q sum %q", m.row, m.sheet.Source(0, 3), m.sheet.Source(1, 4))
	}
	m = press(m, runeKey('i'), runeKey('o'), tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal || m.sheet.Source(m.col, 1) != "o" {
		t.Fatalf("typed o mode %d %q", m.mode, m.sheet.Source(m.col, 1))
	}

	total := sheet.Blank()
	total.Set(1, 0, "10")
	total.Set(1, 1, "20")
	total.Set(1, 2, "=SUM(B1:B2)")
	m = ready(total)
	m.row = 2
	m.anchorR = 2
	m = press(m, runeKey('O'))
	if m.mode != modeNormal || m.row != 2 || m.sheet.Source(1, 2) != "" || m.sheet.Source(1, 3) != "=SUM(B1:B3)" || m.sheet.Display(1, 3) != "30" {
		t.Fatalf("above total mode %d row %d blank %q sum %q display %s", m.mode, m.row, m.sheet.Source(1, 2), m.sheet.Source(1, 3), m.sheet.Display(1, 3))
	}

	held := sheet.Blank()
	held.Set(0, 0, "Rent")
	m = ready(held)
	m = press(m, runeKey('v'), runeKey('o'))
	if m.mode != modeVisual || m.sheet.Source(0, 0) != "Rent" || m.sheet.Source(0, 1) != "" {
		t.Fatalf("visual o mode %d %q %q", m.mode, m.sheet.Source(0, 0), m.sheet.Source(0, 1))
	}

	edge := sheet.Blank()
	if !edge.GrowTo(0, sheet.MaxRow-1) {
		t.Fatal("grow")
	}
	edge.Set(0, sheet.MaxRow-1, "end")
	m = ready(edge)
	m.row = sheet.MaxRow - 1
	m.anchorR = m.row
	m = press(m, runeKey('o'))
	if m.flash != "edge" || m.row != sheet.MaxRow-1 || m.sheet.Source(0, sheet.MaxRow-1) != "end" {
		t.Fatalf("edge flash %q row %d %q", m.flash, m.row, m.sheet.Source(0, sheet.MaxRow-1))
	}
	help := press(ready(sheet.Blank()), runeKey('?'))
	if !strings.Contains(help.View(), "insert row below") || !strings.Contains(help.View(), "=Income!B1") {
		t.Fatalf("help:\n%s", help.View())
	}
}

func ready(s *sheet.Sheet) *model {
	m := newModel(s)
	m.width = 80
	m.height = 24
	m.color = false
	return m
}

func press(m *model, keys ...tea.KeyMsg) *model {
	next, _ := pressCmd(m, keys...)
	return next
}

func pressCmd(m *model, keys ...tea.KeyMsg) (*model, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		next, c := m.Update(k)
		m = next.(*model)
		cmd = c
	}
	return m, cmd
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func runes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
