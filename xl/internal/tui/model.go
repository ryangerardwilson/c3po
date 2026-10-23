package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryangerardwilson/c3po/xl/internal/sheet"
)

type mode int

const (
	modeNormal mode = iota
	modeVisual
	modeInsert
	modeHelp
	modeSave
	modeQuit
	modeRename
	modeCommand
	modeSearch
)

type place struct {
	col, row  int
	top, left int
}

type model struct {
	sheet  *sheet.Sheet
	places map[*sheet.Sheet]place
	mode   mode

	col, row       int
	anchorC        int
	anchorR        int
	top, left      int
	width, height  int
	color          bool
	flash          string
	edit           []rune
	cur            int
	saveBuf        []rune
	saveCur        int
	quitAfterSave  bool
	modeBeforeSave mode
	query          string
}

func Run(s *sheet.Sheet) error {
	p := tea.NewProgram(newModel(s), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel(s *sheet.Sheet) *model {
	if s == nil {
		s = sheet.Blank()
	}
	return &model{
		sheet: s,
		color: os.Getenv("NO_COLOR") == "",
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.reveal()
		return m, nil
	case tea.KeyMsg:
		m.flash = ""
		var next tea.Model
		var cmd tea.Cmd
		switch m.mode {
		case modeHelp:
			next, cmd = m.onHelp(msg)
		case modeSave:
			next, cmd = m.onSave(msg)
		case modeRename:
			next, cmd = m.onRename(msg)
		case modeQuit:
			next, cmd = m.onQuit(msg)
		case modeCommand:
			next, cmd = m.onCommand(msg)
		case modeSearch:
			next, cmd = m.onSearch(msg)
		case modeInsert:
			next, cmd = m.onInsert(msg)
		default:
			next, cmd = m.onNormal(msg)
		}
		if updated, ok := next.(*model); ok {
			updated.reveal()
			return updated, cmd
		}
		return next, cmd
	default:
		return m, nil
	}
}

func (m *model) onNormal(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "h", "left":
		m.move(-1, 0)
	case "l", "right":
		m.move(1, 0)
	case "j", "down":
		m.move(0, 1)
	case "k", "up":
		m.move(0, -1)
	case "ctrl+h", "ctrl+left":
		m.move(-m.pageCols(), 0)
	case "ctrl+l", "ctrl+right":
		m.move(m.pageCols(), 0)
	case "ctrl+j", "ctrl+down", "ctrl+d", "pgdown":
		m.move(0, m.pageRows())
	case "ctrl+k", "ctrl+up", "ctrl+u", "pgup":
		m.move(0, -m.pageRows())
	case "shift+left":
		m.extend(-1, 0)
	case "shift+right":
		m.extend(1, 0)
	case "shift+down":
		m.extend(0, 1)
	case "shift+up":
		m.extend(0, -1)
	case "home":
		m.moveTo(0, m.row)
	case "end":
		m.moveTo(m.sheet.LastCol(m.row), m.row)
	case "ctrl+home":
		m.moveTo(0, 0)
	case "ctrl+end":
		c, r, ok := m.sheet.LastUsed()
		if !ok {
			c, r = 0, 0
		}
		m.moveTo(c, r)
	case "v":
		if m.mode == modeVisual {
			m.mode = modeNormal
			m.anchorC, m.anchorR = m.col, m.row
		} else {
			m.mode = modeVisual
			m.anchorC, m.anchorR = m.col, m.row
		}
	case "i":
		m.beginInsert()
	case "enter":
		if m.mode == modeNormal {
			m.beginInsert()
			break
		}
		m.mode = modeNormal
		m.anchorC, m.anchorR = m.col, m.row
		m.step(0, 1, false)
	case "=":
		m.beginInsert()
		m.edit = []rune{'='}
		m.cur = 1
	case "[":
		m.prevSheet()
	case "]":
		m.nextSheet()
	case "n":
		m.newSheet()
	case "R":
		m.beginRename()
	case "x":
		m.clearSelection()
	case "o":
		if m.mode == modeNormal {
			m.insertRow(true)
		}
	case "O", "shift+o":
		if m.mode == modeNormal {
			m.insertRow(false)
		}
	case "q", "ctrl+c":
		return m.askQuit()
	case ":":
		m.beginCommand()
	case "/":
		m.beginSearch()
	case "ctrl+n":
		m.searchStep(1)
	case "ctrl+p":
		m.searchStep(-1)
	case "?":
		m.mode = modeHelp
	case "esc":
		m.mode = modeNormal
		m.anchorC, m.anchorR = m.col, m.row
	}
	return m, nil
}

func (m *model) onInsert(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyRunes && len(k.Runes) > 0 && !k.Alt {
		for _, r := range k.Runes {
			m.insertRune(r)
		}
		return m, nil
	}
	switch k.String() {
	case "esc":
		// Esc keeps the typed text and returns to normal on this cell.
		m.commit()
		m.mode = modeNormal
		m.anchorC, m.anchorR = m.col, m.row
	case "enter":
		m.commit()
		m.mode = modeNormal
		m.anchorC, m.anchorR = m.col, m.row
		m.step(0, 1, false)
	case "tab":
		m.commit()
		m.mode = modeNormal
		m.step(1, 0, false)
		m.beginInsert()
	case "shift+tab":
		m.commit()
		m.mode = modeNormal
		m.step(-1, 0, false)
		m.beginInsert()
	case "up":
		m.commit()
		m.mode = modeNormal
		m.step(0, -1, false)
		m.beginInsert()
	case "down":
		m.commit()
		m.mode = modeNormal
		m.step(0, 1, false)
		m.beginInsert()
	case "left":
		if m.cur > 0 {
			m.cur--
		}
	case "right":
		if m.cur < len(m.edit) {
			m.cur++
		}
	case "home":
		m.cur = 0
	case "end":
		m.cur = len(m.edit)
	case "backspace", "ctrl+h":
		m.backspace()
	case "ctrl+a":
		m.cur = 0
	case "ctrl+e":
		m.cur = len(m.edit)
	case "ctrl+w":
		m.ruboutWord()
	case "alt+f", "alt+F":
		m.forwardWord()
	case "alt+b", "alt+B":
		m.backwardWord()
	case "delete":
		if m.cur < len(m.edit) {
			m.edit = append(m.edit[:m.cur], m.edit[m.cur+1:]...)
		}
	case " ":
		m.insertRune(' ')
	case "ctrl+c":
		m.commit()
		return m.askQuit()
	}
	return m, nil
}

func (m *model) onHelp(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q", "?":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *model) onSave(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyRunes && len(k.Runes) > 0 {
		for _, r := range k.Runes {
			m.insertSave(r)
		}
		return m, nil
	}
	switch k.String() {
	case "esc":
		m.quitAfterSave = false
		m.mode = m.modeBeforeSave
	case "enter":
		path := strings.TrimSpace(string(m.saveBuf))
		if path == "" {
			m.flash = "save as needs a path"
			return m, nil
		}
		if err := m.sheet.SaveAs(path); err != nil {
			m.flash = err.Error()
			return m, nil
		}
		m.flash = "saved " + filepath.Base(m.sheet.Path)
		if m.quitAfterSave {
			return m, tea.Quit
		}
		m.mode = modeNormal
	case "backspace", "ctrl+h":
		if m.saveCur > 0 {
			m.saveBuf = append(m.saveBuf[:m.saveCur-1], m.saveBuf[m.saveCur:]...)
			m.saveCur--
		}
	case "delete":
		if m.saveCur < len(m.saveBuf) {
			m.saveBuf = append(m.saveBuf[:m.saveCur], m.saveBuf[m.saveCur+1:]...)
		}
	case "left":
		if m.saveCur > 0 {
			m.saveCur--
		}
	case "right":
		if m.saveCur < len(m.saveBuf) {
			m.saveCur++
		}
	case " ":
		m.insertSave(' ')
	case "ctrl+c":
		m.quitAfterSave = false
		m.mode = modeNormal
	}
	return m, nil
}

func (m *model) onQuit(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "y":
		if m.sheet.Path == "" {
			m.quitAfterSave = true
			m.modeBeforeSave = modeNormal
			m.mode = modeSave
			m.saveBuf = nil
			m.saveCur = 0
			return m, nil
		}
		if err := m.sheet.Save(); err != nil {
			m.flash = err.Error()
			m.mode = modeNormal
			return m, nil
		}
		return m, tea.Quit
	case "n":
		return m, tea.Quit
	case "esc":
		m.quitAfterSave = false
		m.mode = modeNormal
	}
	return m, nil
}

func (m *model) onRename(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyRunes && len(k.Runes) > 0 {
		for _, r := range k.Runes {
			m.insertSave(r)
		}
		return m, nil
	}
	switch k.String() {
	case "esc", "ctrl+c":
		m.mode = modeNormal
	case "enter":
		name := strings.TrimSpace(string(m.saveBuf))
		if name == m.sheet.Name {
			m.mode = modeNormal
			return m, nil
		}
		if err := m.sheet.Rename(name); err != nil {
			m.flash = err.Error()
			return m, nil
		}
		m.flash = "renamed " + m.sheet.Name
		m.mode = modeNormal
	case "backspace", "ctrl+h":
		if m.saveCur > 0 {
			m.saveBuf = append(m.saveBuf[:m.saveCur-1], m.saveBuf[m.saveCur:]...)
			m.saveCur--
		}
	case "delete":
		if m.saveCur < len(m.saveBuf) {
			m.saveBuf = append(m.saveBuf[:m.saveCur], m.saveBuf[m.saveCur+1:]...)
		}
	case "left":
		if m.saveCur > 0 {
			m.saveCur--
		}
	case "right":
		if m.saveCur < len(m.saveBuf) {
			m.saveCur++
		}
	case "home":
		m.saveCur = 0
	case "end":
		m.saveCur = len(m.saveBuf)
	case " ":
		m.insertSave(' ')
	}
	return m, nil
}

func (m *model) beginCommand() {
	m.mode = modeCommand
	m.saveBuf = nil
	m.saveCur = 0
}

func (m *model) beginSearch() {
	m.mode = modeSearch
	m.saveBuf = nil
	m.saveCur = 0
}

func (m *model) onCommand(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editPrompt(k) {
		return m, nil
	}
	switch k.String() {
	case "esc", "ctrl+c":
		m.mode = modeNormal
	case "enter":
		return m.runCommand()
	}
	return m, nil
}

func (m *model) onSearch(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+n":
		return m.finishSearch(1)
	case "ctrl+p":
		return m.finishSearch(-1)
	}
	if m.editPrompt(k) {
		return m, nil
	}
	switch k.String() {
	case "esc", "ctrl+c":
		m.mode = modeNormal
	case "enter":
		return m.finishSearch(1)
	}
	return m, nil
}

// editPrompt edits the command or search line.
func (m *model) editPrompt(k tea.KeyMsg) bool {
	if k.Type == tea.KeyRunes && len(k.Runes) > 0 && !k.Alt {
		for _, r := range k.Runes {
			m.insertSave(r)
		}
		return true
	}
	switch k.String() {
	case "backspace", "ctrl+h":
		if m.saveCur > 0 {
			m.saveBuf = append(m.saveBuf[:m.saveCur-1], m.saveBuf[m.saveCur:]...)
			m.saveCur--
		}
		return true
	case "delete":
		if m.saveCur < len(m.saveBuf) {
			m.saveBuf = append(m.saveBuf[:m.saveCur], m.saveBuf[m.saveCur+1:]...)
		}
		return true
	case "left":
		if m.saveCur > 0 {
			m.saveCur--
		}
		return true
	case "right":
		if m.saveCur < len(m.saveBuf) {
			m.saveCur++
		}
		return true
	case "home", "ctrl+a":
		m.saveCur = 0
		return true
	case "end", "ctrl+e":
		m.saveCur = len(m.saveBuf)
		return true
	case "ctrl+w":
		m.ruboutSave()
		return true
	case " ":
		m.insertSave(' ')
		return true
	default:
		return false
	}
}

func (m *model) ruboutSave() {
	if m.saveCur == 0 {
		return
	}
	i := m.saveCur
	for i > 0 && unicode.IsSpace(m.saveBuf[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(m.saveBuf[i-1]) {
		i--
	}
	m.saveBuf = append(m.saveBuf[:i], m.saveBuf[m.saveCur:]...)
	m.saveCur = i
}

func (m *model) runCommand() (tea.Model, tea.Cmd) {
	line := strings.TrimSpace(string(m.saveBuf))
	m.mode = modeNormal
	cmd, arg := splitCmd(line)
	switch strings.ToLower(cmd) {
	case "":
		return m, nil
	case "w":
		return m.writeAnd(arg, false)
	case "wq":
		return m.writeAnd(arg, true)
	case "q":
		return m.askQuit()
	default:
		m.flash = "unknown command"
		return m, nil
	}
}

func splitCmd(line string) (string, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}
	i := strings.IndexFunc(line, unicode.IsSpace)
	if i < 0 {
		return line, ""
	}
	return line[:i], strings.TrimSpace(line[i:])
}

// writeAnd saves the workbook. An empty path with no file yet asks for one.
// quit leaves after a successful save.
func (m *model) writeAnd(path string, quit bool) (tea.Model, tea.Cmd) {
	if path != "" {
		if err := m.sheet.SaveAs(path); err != nil {
			m.flash = err.Error()
			return m, nil
		}
	} else if m.sheet.Path == "" {
		m.quitAfterSave = quit
		m.modeBeforeSave = modeNormal
		m.mode = modeSave
		m.saveBuf = nil
		m.saveCur = 0
		return m, nil
	} else if err := m.sheet.Save(); err != nil {
		m.flash = err.Error()
		return m, nil
	}
	m.flash = "saved " + filepath.Base(m.sheet.Path)
	if quit {
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) finishSearch(dir int) (tea.Model, tea.Cmd) {
	q := strings.TrimSpace(string(m.saveBuf))
	m.mode = modeNormal
	if q != "" {
		m.query = q
	}
	m.searchStep(dir)
	return m, nil
}

// searchStep moves to the next or previous match on this sheet.
// A positive dir walks forward from the cell after the cursor and wraps.
func (m *model) searchStep(dir int) {
	if strings.TrimSpace(m.query) == "" {
		m.flash = "no search"
		return
	}
	hits := m.searchHits()
	if len(hits) == 0 {
		m.flash = "no match"
		return
	}
	idx := pickHit(hits, m.col, m.row, dir)
	h := hits[idx]
	m.mode = modeNormal
	m.moveTo(h.c, h.r)
	m.flash = fmt.Sprintf("%s  %d/%d", sheet.AddrLabel(h.c, h.r), idx+1, len(hits))
}

type cellHit struct {
	c, r int
}

func (m *model) searchHits() []cellHit {
	q := strings.ToLower(m.query)
	var hits []cellHit
	for r := 0; r < m.sheet.Rows; r++ {
		for c := 0; c < m.sheet.Cols; c++ {
			src := m.sheet.Source(c, r)
			if src == "" {
				continue
			}
			shown := strings.ToLower(src)
			if strings.Contains(shown, q) || strings.Contains(strings.ToLower(m.sheet.Display(c, r)), q) {
				hits = append(hits, cellHit{c, r})
			}
		}
	}
	return hits
}

func pickHit(hits []cellHit, col, row, dir int) int {
	if dir < 0 {
		for i := len(hits) - 1; i >= 0; i-- {
			if beforeCell(hits[i].c, hits[i].r, col, row) {
				return i
			}
		}
		return len(hits) - 1
	}
	for i, h := range hits {
		if beforeCell(col, row, h.c, h.r) {
			return i
		}
	}
	return 0
}

func beforeCell(c1, r1, c2, r2 int) bool {
	return r1 < r2 || (r1 == r2 && c1 < c2)
}

func (m *model) remember() {
	if m.places == nil {
		m.places = map[*sheet.Sheet]place{}
	}
	m.places[m.sheet] = place{m.col, m.row, m.top, m.left}
}

func (m *model) showSheet(s *sheet.Sheet) {
	if s == nil || s == m.sheet {
		return
	}
	m.remember()
	m.sheet = s
	p := m.places[s]
	m.col, m.row = p.col, p.row
	m.anchorC, m.anchorR = m.col, m.row
	m.top, m.left = p.top, p.left
	m.mode = modeNormal
	m.reveal()
}

func (m *model) sheetIndex() int {
	n := m.sheet.SheetCount()
	for i := 0; i < n; i++ {
		if m.sheet.SheetAt(i) == m.sheet {
			return i
		}
	}
	return 0
}

func (m *model) prevSheet() {
	i := m.sheetIndex()
	if i > 0 {
		m.showSheet(m.sheet.SheetAt(i - 1))
	}
}

func (m *model) nextSheet() {
	m.showSheet(m.sheet.SheetAt(m.sheetIndex() + 1))
}

func (m *model) newSheet() {
	s, err := m.sheet.AddSheetAuto()
	if err != nil {
		m.flash = err.Error()
		return
	}
	m.showSheet(s)
}

func (m *model) beginRename() {
	m.mode = modeRename
	m.saveBuf = []rune(m.sheet.Name)
	m.saveCur = len(m.saveBuf)
}

func (m *model) insertRow(below bool) {
	at := m.row
	if below {
		at++
	}
	if !m.sheet.InsertRow(at) {
		m.flash = "edge"
		return
	}
	m.row = at
	m.anchorC, m.anchorR = m.col, m.row
}

func (m *model) beginInsert() {
	m.mode = modeInsert
	m.anchorC, m.anchorR = m.col, m.row
	m.edit = []rune(m.sheet.Source(m.col, m.row))
	m.cur = len(m.edit)
}

func (m *model) commit() {
	src := string(m.edit)
	if strings.TrimSpace(src) == "" {
		src = ""
	}
	m.sheet.Set(m.col, m.row, src)
}

func (m *model) insertRune(r rune) {
	if r < 32 {
		return
	}
	next := make([]rune, 0, len(m.edit)+1)
	next = append(next, m.edit[:m.cur]...)
	next = append(next, r)
	next = append(next, m.edit[m.cur:]...)
	m.edit = next
	m.cur++
}

func (m *model) insertSave(r rune) {
	if r < 32 {
		return
	}
	next := make([]rune, 0, len(m.saveBuf)+1)
	next = append(next, m.saveBuf[:m.saveCur]...)
	next = append(next, r)
	next = append(next, m.saveBuf[m.saveCur:]...)
	m.saveBuf = next
	m.saveCur++
}

func (m *model) backspace() {
	if m.cur == 0 {
		return
	}
	m.edit = append(m.edit[:m.cur-1], m.edit[m.cur:]...)
	m.cur--
}

func (m *model) forwardWord() {
	i := m.cur
	n := len(m.edit)
	for i < n && !isEditWord(m.edit[i]) {
		i++
	}
	for i < n && isEditWord(m.edit[i]) {
		i++
	}
	m.cur = i
}

func (m *model) backwardWord() {
	i := m.cur
	for i > 0 && !isEditWord(m.edit[i-1]) {
		i--
	}
	for i > 0 && isEditWord(m.edit[i-1]) {
		i--
	}
	m.cur = i
}

// ruboutWord deletes back through the previous word. A space is the boundary.
func (m *model) ruboutWord() {
	if m.cur == 0 {
		return
	}
	i := m.cur
	for i > 0 && unicode.IsSpace(m.edit[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(m.edit[i-1]) {
		i--
	}
	m.edit = append(m.edit[:i], m.edit[m.cur:]...)
	m.cur = i
}

func isEditWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (m *model) clearSelection() {
	c1, r1, c2, r2 := m.selection()
	for r := r1; r <= r2; r++ {
		for c := c1; c <= c2; c++ {
			m.sheet.Set(c, r, "")
		}
	}
	m.mode = modeNormal
	m.anchorC, m.anchorR = m.col, m.row
}

func (m *model) selection() (c1, r1, c2, r2 int) {
	c1, c2 = m.anchorC, m.col
	r1, r2 = m.anchorR, m.row
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	return c1, r1, c2, r2
}

func (m *model) move(dc, dr int) {
	m.step(dc, dr, m.mode == modeVisual)
}

func (m *model) extend(dc, dr int) {
	m.mode = modeVisual
	m.step(dc, dr, true)
}

func (m *model) moveTo(c, r int) {
	m.move(c-m.col, r-m.row)
}

func (m *model) step(dc, dr int, keepAnchor bool) bool {
	wantC := m.col + dc
	wantR := m.row + dr
	nc, nr := wantC, wantR
	if nc < 0 {
		nc = 0
	}
	if nr < 0 {
		nr = 0
	}
	if nc >= sheet.MaxCol {
		nc = sheet.MaxCol - 1
	}
	if nr >= sheet.MaxRow {
		nr = sheet.MaxRow - 1
	}
	if nc == m.col && nr == m.row {
		if wantC >= sheet.MaxCol || wantR >= sheet.MaxRow {
			m.flash = "edge"
		}
		return false
	}
	if !m.sheet.GrowTo(nc, nr) {
		m.flash = "edge"
		return false
	}
	m.col, m.row = nc, nr
	if !keepAnchor {
		m.anchorC, m.anchorR = nc, nr
	}
	m.reveal()
	return true
}

func (m *model) askQuit() (tea.Model, tea.Cmd) {
	if !m.sheet.Dirty {
		return m, tea.Quit
	}
	m.mode = modeQuit
	return m, nil
}

func (m *model) pageRows() int {
	n := m.gridRows() / 2
	if n < 1 {
		return 1
	}
	return n
}

func (m *model) pageCols() int { return 4 }

func (m *model) gridRows() int {
	// Title, column header, and the formula line.
	chrome := 3
	if m.bottomLine() != "" {
		chrome++
	}
	n := m.height - chrome
	if n < 1 {
		return 1
	}
	return n
}

func (m *model) reveal() {
	if m.width == 0 || m.height == 0 {
		return
	}
	rows := m.gridRows()
	if m.row < m.top {
		m.top = m.row
	}
	if m.row >= m.top+rows {
		m.top = m.row - rows + 1
	}
	if m.top < 0 {
		m.top = 0
	}
	if m.col < m.left {
		m.left = m.col
	}
	for !m.colInView() && m.left < m.col {
		m.left++
	}
	if m.left < 0 {
		m.left = 0
	}
}

func (m *model) colInView() bool {
	used := m.gutter()
	for c := m.left; c <= m.col && c < m.sheet.Cols; c++ {
		if c > m.left {
			used++
		}
		used += m.colWidth(c)
	}
	return used <= m.width || m.left == m.col
}
