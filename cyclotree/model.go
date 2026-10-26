package cyclotree

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/ast"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

type node struct {
	parent                                *node
	id, name, path                        string
	line, cyclo, levels                   int
	scored, linked, open, loaded, canOpen bool
	cycle                                 bool
	kids                                  []*node
}

type placed struct {
	n      *node
	depth  int
	parent int
}

type callees interface {
	Callees(id string) []ast.Fn
}

type screen struct {
	root      *node
	calls     callees
	scores    *scoreIndex
	cursor    int
	rows      []placed
	nav       map[string]navMemo
	steps     int
	shortcuts bool
	session   *tuisession.Session
}

func (s *screen) noteSession(dir, symbol string) {
	if s == nil || s.session == nil {
		return
	}
	args := []string{}
	if symbol != "" {
		args = []string{symbol}
	}
	mark := cursorMarkOf(s)
	_ = s.session.Place(dir, mark.key)
	_ = s.session.Note(tuisession.Body{Args: args})
}

// Key applies one motion. j and k move among visible rows. h moves to
// the caller. l opens a closed row, or moves into the first call of an
// open row. c collapses the row and every open row under it. A cycle
// stays closed. ? toggles the shortcut list. While that list is open,
// motions do nothing.
func (s *screen) Key(r rune) {
	if s == nil {
		return
	}
	if r == '?' {
		s.shortcuts = !s.shortcuts
		return
	}
	if s.shortcuts {
		return
	}
	switch r {
	case 'j':
		s.move(1)
	case 'k':
		s.move(-1)
	case 'h':
		s.toCaller()
	case 'l':
		s.openCalls()
	case 'c':
		s.collapse()
	}
}

func (s *screen) move(d int) {
	s.layout()
	next := s.cursor + d
	if next < 0 || next >= len(s.rows) {
		return
	}
	s.cursor = next
}

func (s *screen) toCaller() {
	s.layout()
	if s.cursor < 0 || s.cursor >= len(s.rows) {
		return
	}
	if p := s.rows[s.cursor].parent; p >= 0 {
		s.cursor = p
	}
}

func (s *screen) openCalls() {
	s.layout()
	if len(s.rows) == 0 {
		return
	}
	n := s.rows[s.cursor].n
	if n.cycle {
		return
	}
	wasOpen := n.open
	s.expand(n)
	if wasOpen {
		s.enterFirst()
	}
}

func (s *screen) enterFirst() {
	s.layout()
	if s.cursor+1 < len(s.rows) && s.rows[s.cursor+1].depth > s.rows[s.cursor].depth {
		s.cursor++
	}
}

func (s *screen) collapse() {
	s.layout()
	if len(s.rows) == 0 {
		return
	}
	closeUnder(s.rows[s.cursor].n)
}

func closeUnder(n *node) {
	stack := []*node{n}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == nil {
			continue
		}
		cur.open = false
		stack = append(stack, cur.kids...)
	}
}

func (n *node) expandable() bool {
	if n == nil || n.cycle {
		return false
	}
	if n.loaded {
		return len(n.kids) > 0
	}
	return n.canOpen
}

func (s *screen) expand(n *node) {
	if n.cycle {
		return
	}
	if !n.loaded {
		n.loaded = true
		s.fill(n)
	}
	n.open = true
}

// expandReachable opens every call reached from the root. A cycle stays
// closed. A function already opened on another branch stays closed.
func (s *screen) expandReachable() {
	if s == nil || s.root == nil {
		return
	}
	seen := map[string]bool{}
	stack := []*node{s.root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == nil || n.cycle {
			continue
		}
		if n.id != "" {
			if seen[n.id] {
				continue
			}
			seen[n.id] = true
		}
		s.expand(n)
		for i := len(n.kids) - 1; i >= 0; i-- {
			stack = append(stack, n.kids[i])
		}
	}
}

func (s *screen) fill(n *node) {
	if !n.linked || s.calls == nil {
		return
	}
	anc := ancestorIDs(n)
	for _, c := range s.calls.Callees(n.id) {
		if c.ID == "" {
			continue
		}
		n.kids = append(n.kids, s.child(n, c, anc))
	}
}

func (s *screen) child(parent *node, c ast.Fn, anc map[string]bool) *node {
	n := &node{
		parent: parent, id: c.ID, name: c.Name, path: c.Path, line: c.Line,
		linked: true, cycle: anc[c.ID],
	}
	if s.scores != nil {
		n.cyclo, n.scored = s.scores.lookup(c.Path, c.Line, c.Name)
	}
	if !n.cycle && s.calls != nil && len(s.calls.Callees(c.ID)) > 0 {
		n.canOpen = true
	}
	s.bindDepth(n)
	return n
}

func ancestorIDs(n *node) map[string]bool {
	out := map[string]bool{}
	for cur := n; cur != nil; cur = cur.parent {
		if cur.id != "" {
			out[cur.id] = true
		}
	}
	return out
}

func (s *screen) layout() {
	s.rows = nil
	s.place(s.root, 0, -1)
	if len(s.rows) == 0 {
		s.cursor = 0
		return
	}
	if s.cursor >= len(s.rows) {
		s.cursor = len(s.rows) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
}

func (s *screen) place(n *node, depth, parent int) {
	if n == nil {
		return
	}
	idx := len(s.rows)
	s.rows = append(s.rows, placed{n: n, depth: depth, parent: parent})
	if !n.open {
		return
	}
	for _, k := range n.kids {
		s.place(k, depth+1, idx)
	}
}

// Text renders the open rows. When cursor is set, a fixed column holds the marker so names do not shift.
func (s *screen) Text(cursor bool) string {
	s.layout()
	var b strings.Builder
	for i, row := range s.rows {
		fmt.Fprintln(&b, formatRow(row.depth, row.n, cursor, cursor && i == s.cursor))
	}
	return b.String()
}

func formatRow(depth int, n *node, gutter, cursor bool) string {
	label := rowLabel(n)
	pad := strings.Repeat("  ", depth)
	if !gutter {
		return pad + label
	}
	label = paintLabel(label, n.expandable())
	if cursor {
		return "> " + pad + label
	}
	return "  " + pad + label
}

func rowLabel(n *node) string {
	label := n.name
	if n.path != "" {
		cyclo := "-"
		if n.scored {
			cyclo = strconv.Itoa(n.cyclo)
		}
		label = fmt.Sprintf("%s  %s  %d  %s:%d", n.name, cyclo, n.levels, rowPath(n), n.line)
	}
	if n.cycle {
		label += " cycle"
	}
	return label
}

// rowPath is the caller's directory plus the rest of the file.
// The root keeps the path from the directory being read. A step up that
// would be longer than that path stays as it is.
func rowPath(n *node) string {
	if n == nil || n.path == "" {
		return ""
	}
	if n.parent == nil || n.parent.path == "" {
		return n.path
	}
	return pathFromParent(n.parent.path, n.path)
}

func pathFromParent(parent, child string) string {
	rel, err := filepath.Rel(filepath.Dir(parent), child)
	if err != nil {
		return child
	}
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." {
		return child
	}
	if strings.HasPrefix(rel, "..") && len(rel) >= len(child) {
		return child
	}
	return rel
}

func paintLabel(label string, bright bool) string {
	if bright {
		return "\x1b[1m" + label + "\x1b[22m"
	}
	return "\x1b[2m" + label + "\x1b[22m"
}

// Frame is the screen, scrolled so the cursor stays visible.
// The last two lines are the call chain and the full path of the row
// under the cursor. The shortcut list replaces the tree while it is open.
func (s *screen) Frame(height int) string {
	if height < 1 {
		height = 1
	}
	lines := make([]string, height)
	lines[0] = "cyclotree"
	foot := writeFooter(lines, s)
	if s != nil && s.shortcuts {
		placeLines(lines, 1, foot, shortcutLines())
		return strings.Join(lines, "\n") + "\n"
	}
	if height > 3 {
		lines[1] = "  name  cyclo  depth"
	}
	if height > 4 && s != nil {
		body := strings.Split(strings.TrimRight(s.Text(true), "\n"), "\n")
		start, end := window(len(body), s.cursor, height)
		placeLines(lines, 2, foot, body[start:end])
	}
	return strings.Join(lines, "\n") + "\n"
}

// writeFooter puts the call chain and the full path on the last two lines.
// It returns the index where the tree or the shortcut list must stop.
func writeFooter(lines []string, s *screen) int {
	n := len(lines)
	if n > 1 {
		lines[n-1] = focusPath(s)
	}
	if n > 2 {
		lines[n-2] = focusChain(s)
	}
	foot := n - 2
	if foot < 1 {
		foot = 1
	}
	return foot
}

// focusPath is the path from the directory being read, with its line.
func focusPath(s *screen) string {
	rel, line, ok := cursorFile(s)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", rel, line)
}

// focusChain is the calls from the root down to the row under the cursor.
func focusChain(s *screen) string {
	var names []string
	for n := cursorNode(s); n != nil; n = n.parent {
		names = append(names, n.name)
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return strings.Join(names, " -> ")
}

func cursorNode(s *screen) *node {
	if s == nil {
		return nil
	}
	s.layout()
	if s.cursor < 0 || s.cursor >= len(s.rows) {
		return nil
	}
	return s.rows[s.cursor].n
}

// placeLines writes src into dst from start up to end.
func placeLines(dst []string, start, end int, src []string) {
	for i, line := range src {
		at := start + i
		if at >= end || at >= len(dst) {
			return
		}
		dst[at] = line
	}
}

func shortcutLines() []string {
	body := []string{
		"j/k    move",
		"h      caller",
		"l      calls",
		"c      collapse",
		"enter  open in vim",
		"?      shortcuts",
		"q      quit",
	}
	width := 0
	for _, line := range body {
		if len(line) > width {
			width = len(line)
		}
	}
	bar := "  +" + strings.Repeat("-", width+2) + "+"
	out := []string{bar}
	for _, line := range body {
		pad := strings.Repeat(" ", width-len(line))
		out = append(out, "  | "+line+pad+" |")
	}
	return append(out, bar)
}

func window(n, cursor, height int) (int, int) {
	room := height - 4 // title, column header, call chain, and the path line
	if room < 1 {
		room = 1
	}
	if n == 0 {
		return 0, 0
	}
	start := 0
	if cursor >= room {
		start = cursor - room + 1
	}
	end := start + room
	if end > n {
		end = n
		if start > end-room && end-room > 0 {
			start = end - room
		}
	}
	if start < 0 {
		start = 0
	}
	return start, end
}
