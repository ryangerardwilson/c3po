package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ryangerardwilson/c3po/xl/internal/sheet"
)

const (
	ansiReset = "\033[0m"
	// Truecolor. The terminal theme maps the 16 ANSI colors onto white and
	// gray, so indexed cyan and yellow never show, and the cursor disappears
	// into the shade.
	ansiCyan   = "\033[38;2;0;229;255m"
	ansiRed    = "\033[38;2;255;70;70m"
	ansiYellow = "\033[38;2;255;214;0m"
	ansiWhite  = "\033[38;2;255;255;255m"
	// White on red. The red is not the cyan shade.
	ansiCursor = "\033[38;2;255;255;255;48;2;255;70;70m"
	ansiSelect = "\033[38;2;255;255;255;48;2;214;40;40m"
	// Black on cyan.
	ansiShade = "\033[38;2;0;0;0;48;2;0;229;255m"
)

func (m *model) paint(s, code string) string {
	if !m.color || s == "" || code == "" {
		return s
	}
	return code + s + ansiReset
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "xl\n"
	}
	if m.width < 24 || m.height < 8 {
		return "xl needs a larger terminal\n"
	}
	lines := []string{m.titleLine(), m.headerLine()}
	if m.mode == modeHelp {
		lines = append(lines, m.helpModal()...)
	} else {
		lines = append(lines, m.gridLines()...)
	}
	lines = append(lines, m.formulaLine())
	if bottom := m.bottomLine(); bottom != "" {
		lines = append(lines, bottom)
	}
	return strings.Join(lines, "\n")
}

func (m *model) titleLine() string {
	name := m.sheet.Path
	if name == "" {
		name = "blank"
	}
	state := ""
	switch {
	case m.flash != "":
		state = m.flash
	case m.sheet.Dirty:
		state = "modified"
	}
	prefix := "xl  "
	suffix := ""
	if state != "" {
		suffix = "  " + state
	}
	room := m.width - utf8.RuneCountInString(prefix) - utf8.RuneCountInString(suffix)
	if room < 4 {
		return fit(prefix+suffix, m.width)
	}
	if utf8.RuneCountInString(name) > room {
		name = fit(name, room)
	}
	if !m.color {
		return prefix + name + suffix
	}
	line := ansiWhite + "xl" + ansiReset + "  " + name
	if state != "" {
		tone := ansiYellow
		if state == "modified" {
			tone = ansiRed
		}
		line += "  " + tone + state + ansiReset
	}
	return line
}

func (m *model) headerLine() string {
	var b strings.Builder
	b.WriteString(padLeft("", m.gutter()))
	for i, c := range m.visibleCols() {
		if i > 0 {
			b.WriteByte(' ')
		}
		label := padCenter(sheet.ColName(c), m.colWidth(c))
		if c == m.col {
			label = m.paint(label, ansiYellow)
		} else {
			label = m.paint(label, "\033[2m")
		}
		b.WriteString(label)
	}
	return b.String()
}

func (m *model) gridLines() []string {
	cols := m.visibleCols()
	c1, r1, c2, r2 := m.selection()
	lines := make([]string, 0, m.gridRows())
	for i := 0; i < m.gridRows(); i++ {
		r := m.top + i
		if r >= m.sheet.Rows {
			break
		}
		var b strings.Builder
		gutter := padLeft(strconv.Itoa(r+1), m.gutter()-1) + " "
		if r == m.row {
			gutter = m.paint(gutter, ansiYellow)
		}
		b.WriteString(gutter)
		for n, c := range cols {
			if n > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(m.paintCell(c, r, c >= c1 && c <= c2 && r >= r1 && r <= r2))
		}
		lines = append(lines, b.String())
	}
	return lines
}

func (m *model) paintCell(c, r int, selected bool) string {
	width := m.colWidth(c)
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	active := c == m.col && r == m.row
	editing := active && m.mode == modeInsert
	var field string
	if editing {
		field = editField(m.edit, m.cur, inner, m.color)
	} else {
		content := fit(m.sheet.Display(c, r), inner)
		if m.sheet.Value(c, r).Kind == sheet.Number {
			field = padLeft(content, inner)
		} else {
			field = padRight(content, inner)
		}
		if m.color && !selected && strings.HasPrefix(strings.TrimSpace(content), "#") {
			field = ansiRed + field + ansiReset
		}
	}
	left, right := " ", " "
	if active && !m.color {
		left, right = "|", "|"
	}
	body := left + field + right
	if active && m.color {
		return ansiShade + body + ansiReset
	}
	if m.color && selected && m.mode == modeVisual && !active {
		return ansiSelect + body + ansiReset
	}
	return body
}

func editField(edit []rune, cur, inner int, color bool) string {
	if inner < 1 {
		return ""
	}
	if cur < 0 {
		cur = 0
	}
	if cur > len(edit) {
		cur = len(edit)
	}
	start := 0
	if cur >= inner {
		start = cur - inner + 1
	}
	chars := make([]rune, inner)
	for i := range chars {
		chars[i] = ' '
	}
	for i := 0; start+i < len(edit) && i < inner; i++ {
		chars[i] = edit[start+i]
	}
	at := cur - start
	if at < 0 {
		at = 0
	}
	if at >= inner {
		at = inner - 1
	}
	var b strings.Builder
	for i, ch := range chars {
		s := string(ch)
		switch {
		case i == at && color:
			s = ansiCursor + s + ansiShade
		case i == at && ch == ' ':
			s = "_"
		}
		b.WriteString(s)
	}
	return b.String()
}

func (m *model) formulaLine() string {
	name := "normal"
	body := m.sheet.Source(m.col, m.row)
	switch m.mode {
	case modeInsert:
		name = "insert"
		body = cursorText(m.edit, m.cur)
	case modeVisual:
		name = "visual"
	case modeHelp:
		name = "help"
	case modeSave:
		name = "save"
	case modeRename:
		name = "rename"
	case modeCommand:
		name = "cmd"
	case modeSearch:
		name = "search"
	}
	addr := m.sheet.Name + "!" + sheet.AddrLabel(m.col, m.row)
	line := fit(fmt.Sprintf("%-6s  %s  %s", name, addr, body), m.width)
	if m.color && strings.Contains(line, addr) {
		line = strings.Replace(line, addr, ansiCyan+addr+ansiReset, 1)
	}
	return line
}

func (m *model) bottomLine() string {
	if prompt := m.promptLine(); prompt != "" {
		return prompt
	}
	return m.statsLine()
}

func (m *model) statsLine() string {
	if m.mode != modeVisual {
		return ""
	}
	c1, r1, c2, r2 := m.selection()
	if c1 == c2 && r1 == r2 || !m.sheet.PureNumbers(c1, r1, c2, r2) {
		return ""
	}
	label := sheet.RangeLabel(c1, r1, c2, r2) + "  "
	return m.paint(fit(label+m.sheet.Stats(c1, r1, c2, r2).String(), m.width), ansiYellow)
}

func (m *model) promptLine() string {
	switch m.mode {
	case modeSave:
		return fit("save as: "+cursorText(m.saveBuf, m.saveCur)+"   enter save   esc cancel", m.width)
	case modeRename:
		return fit("rename sheet: "+cursorText(m.saveBuf, m.saveCur)+"   enter rename   esc cancel", m.width)
	case modeCommand:
		return m.paint(fit(":"+cursorText(m.saveBuf, m.saveCur), m.width), ansiYellow)
	case modeSearch:
		return m.paint(fit("/"+cursorText(m.saveBuf, m.saveCur), m.width), ansiYellow)
	default:
		return ""
	}
}

func helpBody() []string {
	return []string{
		"shortcuts",
		"? or esc          close",
		"hjkl or arrows    move",
		"i or =            edit the cell",
		"esc               normal mode, keep the cell",
		"enter             edit; down when editing",
		"ctrl-a/e/w       start, end, delete word",
		"alt-f/b          next word, previous word",
		"v then hjkl       select",
		"x                 clear",
		"[ ]               previous and next sheet",
		"n                 new sheet",
		"R                 rename sheet",
		":w :wq :q!       save, save and quit, discard",
		"/ ctrl-n/p       search, next, previous",
		"o O               insert row below, above",
		"dd                delete the current row",
		"=Income!B1",
		"=SUM(Income!B1:B2)",
		"=E4-B5",
		"=SUM(B2:B4)",
		"=AVERAGE(B2:B4)",
		"=MEDIAN(B2:B4)",
		"The xlsx stores formulas as formulas.",
	}
}

func (m *model) helpModal() []string {
	room := m.gridRows()
	body := helpBody()
	maxBody := room - 2
	if maxBody < 1 {
		maxBody = room
	}
	if len(body) > maxBody {
		body = body[:maxBody]
	}
	innerW := 0
	for _, line := range body {
		if n := utf8.RuneCountInString(line); n > innerW {
			innerW = n
		}
	}
	boxW := innerW + 4
	if boxW > m.width {
		boxW = m.width
	}
	if boxW < 4 {
		boxW = 4
	}
	inner := boxW - 4
	boxed := make([]string, 0, len(body)+2)
	rule := m.paint("+"+strings.Repeat("-", boxW-2)+"+", ansiCyan)
	boxed = append(boxed, rule)
	for _, line := range body {
		boxed = append(boxed, m.helpRow(padRight(fit(line, inner), inner)))
	}
	boxed = append(boxed, rule)
	if len(boxed) > room {
		boxed = boxed[:room]
	}
	top := (room - len(boxed)) / 2
	left := (m.width - boxW) / 2
	if left < 0 {
		left = 0
	}
	pad := strings.Repeat(" ", left)
	out := make([]string, room)
	for i, row := range boxed {
		out[top+i] = pad + row
	}
	return out
}

func (m *model) helpRow(inner string) string {
	if !m.color {
		return "| " + inner + " |"
	}
	return ansiCyan + "| " + ansiReset + inner + ansiCyan + " |" + ansiReset
}

func (m *model) visibleCols() []int {
	used := m.gutter()
	var cols []int
	for c := m.left; c < m.sheet.Cols; c++ {
		w := m.colWidth(c)
		gap := 0
		if len(cols) > 0 {
			gap = 1
		}
		if len(cols) > 0 && used+gap+w > m.width {
			break
		}
		used += gap + w
		cols = append(cols, c)
	}
	if len(cols) == 0 && m.sheet.Cols > 0 {
		cols = []int{m.left}
	}
	return cols
}

func (m *model) colWidth(c int) int {
	w := utf8.RuneCountInString(sheet.ColName(c))
	for r := 0; r < m.sheet.Rows; r++ {
		n := utf8.RuneCountInString(m.sheet.Display(c, r))
		if n > w {
			w = n
		}
	}
	if m.mode == modeInsert && c == m.col {
		n := len(m.edit) + 1
		if n > w {
			w = n
		}
	}
	w += 2
	if w < 8 {
		w = 8
	}
	if w > 20 {
		w = 20
	}
	return w
}

func (m *model) gutter() int {
	w := len(strconv.Itoa(m.sheet.Rows))
	if w < 2 {
		w = 2
	}
	return w + 1
}

func cursorText(edit []rune, cur int) string {
	if cur < 0 {
		cur = 0
	}
	if cur > len(edit) {
		cur = len(edit)
	}
	return string(edit[:cur]) + "|" + string(edit[cur:])
}

func fit(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return string(r[:1])
	}
	return string(r[:n-1]) + "…"
}

func padRight(s string, n int) string {
	w := utf8.RuneCountInString(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func padLeft(s string, n int) string {
	w := utf8.RuneCountInString(s)
	if w >= n {
		return s
	}
	return strings.Repeat(" ", n-w) + s
}

func padCenter(s string, n int) string {
	w := utf8.RuneCountInString(s)
	if w >= n {
		return fit(s, n)
	}
	left := (n - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", n-w-left)
}
