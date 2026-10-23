package sheet

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxSheets = 32

// Book is one xlsx workbook. Every sheet in it shares the path and the dirty flag.
type Book struct {
	Path   string
	Dirty  bool
	Sheets []*Sheet
	epoch  uint64
}

type visit struct {
	sheet string
	a     Addr
}

func emptyGrid() *Sheet {
	return &Sheet{
		Cols:  blankCols,
		Rows:  blankRows,
		cells: map[Addr]string{},
	}
}

func (s *Sheet) Book() *Book {
	if s == nil || s.book == nil {
		return &Book{Sheets: []*Sheet{s}}
	}
	return s.book
}

func (s *Sheet) SheetCount() int {
	if s == nil || s.book == nil {
		return 1
	}
	return len(s.book.Sheets)
}

func (s *Sheet) SheetAt(i int) *Sheet {
	if s == nil || s.book == nil {
		if i == 0 {
			return s
		}
		return nil
	}
	if i < 0 || i >= len(s.book.Sheets) {
		return nil
	}
	return s.book.Sheets[i]
}

func (s *Sheet) resolve(name string) *Sheet {
	if s == nil {
		return nil
	}
	if name == "" || strings.EqualFold(name, s.Name) {
		return s
	}
	if s.book == nil {
		return nil
	}
	for _, other := range s.book.Sheets {
		if strings.EqualFold(other.Name, name) {
			return other
		}
	}
	return nil
}

func (s *Sheet) AddSheet(name string) (*Sheet, error) {
	if s == nil || s.book == nil {
		return nil, fmt.Errorf("no workbook")
	}
	name = strings.TrimSpace(name)
	if err := validateSheetName(name, s.book, nil); err != nil {
		return nil, err
	}
	if len(s.book.Sheets) >= maxSheets {
		return nil, fmt.Errorf("too many sheets")
	}
	n := emptyGrid()
	n.Name = name
	n.book = s.book
	n.Path = s.Path
	n.epoch = s.book.epoch
	s.book.Sheets = append(s.book.Sheets, n)
	s.touch()
	return n, nil
}

func (s *Sheet) AddSheetAuto() (*Sheet, error) {
	for i := 1; i < maxSheets+2; i++ {
		name := fmt.Sprintf("Sheet%d", i)
		if !bookHasName(s.book, name) {
			return s.AddSheet(name)
		}
	}
	return nil, fmt.Errorf("too many sheets")
}

func bookHasName(b *Book, name string) bool {
	if b == nil {
		return false
	}
	for _, sh := range b.Sheets {
		if strings.EqualFold(sh.Name, name) {
			return true
		}
	}
	return false
}

func (s *Sheet) Rename(name string) error {
	if s == nil || s.book == nil {
		return fmt.Errorf("no workbook")
	}
	name = strings.TrimSpace(name)
	if err := validateSheetName(name, s.book, s); err != nil {
		return err
	}
	old := s.Name
	if old != name {
		for _, sh := range s.book.Sheets {
			for addr, src := range sh.cells {
				next := rewriteSheetRef(src, old, name)
				if next != src {
					sh.cells[addr] = next
				}
			}
		}
		s.Name = name
	}
	s.touch()
	return nil
}

func (s *Sheet) touch() {
	if s.book == nil {
		s.Dirty = true
		s.epoch++
		return
	}
	s.book.epoch++
	s.book.Dirty = true
	for _, sh := range s.book.Sheets {
		sh.Dirty = true
		sh.epoch = s.book.epoch
	}
}

func (s *Sheet) setPath(path string) {
	s.Path = path
	if s.book == nil {
		return
	}
	s.book.Path = path
	for _, sh := range s.book.Sheets {
		sh.Path = path
	}
}

func (s *Sheet) setClean() {
	s.Dirty = false
	if s.book == nil {
		return
	}
	s.book.Dirty = false
	for _, sh := range s.book.Sheets {
		sh.Dirty = false
	}
}

func (s *Sheet) adopt(b *Book) {
	s.book = b
	b.Sheets = append(b.Sheets, s)
}

func validateSheetName(name string, b *Book, self *Sheet) error {
	if name == "" {
		return fmt.Errorf("sheet name is empty")
	}
	if utf8.RuneCountInString(name) > 31 {
		return fmt.Errorf("sheet name is too long")
	}
	if strings.ContainsAny(name, `:\/?*[]`) {
		return fmt.Errorf("sheet name has an invalid character")
	}
	for _, sh := range b.Sheets {
		if sh != self && strings.EqualFold(sh.Name, name) {
			return fmt.Errorf("sheet name is already used")
		}
	}
	return nil
}

func rewriteSheetRef(src, oldName, newName string) string {
	if oldName == "" || !strings.Contains(strings.ToLower(src), strings.ToLower(oldName)) {
		return src
	}
	quoted := quoteSheetName(newName)
	var b strings.Builder
	i := 0
	for i < len(src) {
		if src[i] == '"' {
			j := skipQuoted(src, i)
			b.WriteString(src[i:j])
			i = j
			continue
		}
		if src[i] == '\'' {
			name, j, ok := quotedSheetAt(src, i)
			if ok && j < len(src) && src[j] == '!' && strings.EqualFold(name, oldName) {
				b.WriteString(quoted)
				b.WriteByte('!')
				i = j + 1
				continue
			}
		}
		if isLetter(src[i]) {
			j := i + 1
			for j < len(src) && bareSheetChar(src[j]) {
				j++
			}
			if j < len(src) && src[j] == '!' && strings.EqualFold(src[i:j], oldName) {
				b.WriteString(quoted)
				b.WriteByte('!')
				i = j + 1
				continue
			}
		}
		b.WriteByte(src[i])
		i++
	}
	return b.String()
}

func quoteSheetName(name string) string {
	if bareSheetName(name) {
		return name
	}
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

func bareSheetName(name string) bool {
	if name == "" || !isLetter(name[0]) {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !bareSheetChar(name[i]) {
			return false
		}
	}
	return true
}

func bareSheetChar(b byte) bool {
	return isLetter(b) || isDigit(b) || b == '_' || b == '.'
}

func quotedSheetAt(s string, i int) (string, int, bool) {
	if i >= len(s) || s[i] != '\'' {
		return "", i, false
	}
	var b strings.Builder
	j := i + 1
	for j < len(s) {
		if s[j] != '\'' {
			b.WriteByte(s[j])
			j++
			continue
		}
		if j+1 < len(s) && s[j+1] == '\'' {
			b.WriteByte('\'')
			j += 2
			continue
		}
		return b.String(), j + 1, true
	}
	return "", i, false
}

func skipQuoted(s string, i int) int {
	j := i + 1
	for j < len(s) {
		if s[j] != '"' {
			j++
			continue
		}
		j++
		if j < len(s) && s[j] == '"' {
			j++
			continue
		}
		break
	}
	return j
}

func uniqueSheetName(b *Book, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Sheet1"
	}
	if !bookHasName(b, name) {
		return name
	}
	for i := 2; i < 100; i++ {
		next := fmt.Sprintf("%s %d", name, i)
		if utf8.RuneCountInString(next) > 31 {
			next = fmt.Sprintf("Sheet%d", len(b.Sheets)+1)
		}
		if !bookHasName(b, next) {
			return next
		}
	}
	return fmt.Sprintf("Sheet%d", len(b.Sheets)+1)
}

// shiftFormulaRows rewrites references to sheetName. local is true when src
// lives on that sheet, so an unqualified reference names it too.
func shiftFormulaRows(src, sheetName string, local bool, at int) (string, bool) {
	if !strings.HasPrefix(strings.TrimSpace(src), "=") {
		return src, true
	}
	var b strings.Builder
	i := 0
	for i < len(src) {
		if src[i] == '"' {
			j := skipQuoted(src, i)
			b.WriteString(src[i:j])
			i = j
			continue
		}
		if src[i] != '\'' && src[i] != '$' && !isLetter(src[i]) {
			b.WriteByte(src[i])
			i++
			continue
		}
		hit, ok := parseRefHit(src, i)
		if !ok {
			b.WriteByte(src[i])
			i++
			continue
		}
		k := skipSpaceBytes(src, hit.end)
		if k < len(src) && src[k] == ':' {
			if hit2, ok2 := parseRefHit(src, skipSpaceBytes(src, k+1)); ok2 {
				span, fit := adjustRange(src, hit, hit2, local, sheetName, at)
				if !fit {
					return "", false
				}
				b.WriteString(span)
				i = hit2.end
				continue
			}
		}
		span, fit := adjustOne(src, hit, hitApplies(hit, "", local, sheetName), at)
		if !fit {
			return "", false
		}
		b.WriteString(span)
		i = hit.end
	}
	return b.String(), true
}

type refHit struct {
	sheet            string
	start, end       int
	rowStart, rowEnd int
	row              int
}

func parseRefHit(src string, i int) (refHit, bool) {
	if i >= len(src) {
		return refHit{}, false
	}
	h := refHit{start: i}
	j := i
	if src[j] == '\'' {
		name, after, ok := quotedSheetAt(src, j)
		if !ok || after >= len(src) || src[after] != '!' {
			return refHit{}, false
		}
		if _, _, _, _, ok := cellRow(src, after+1); !ok {
			return refHit{}, false
		}
		h.sheet = name
		j = after + 1
	} else if isLetter(src[j]) {
		k := j + 1
		for k < len(src) && bareSheetChar(src[k]) {
			k++
		}
		if k < len(src) && src[k] == '!' {
			if _, _, _, _, ok := cellRow(src, k+1); ok {
				h.sheet = src[j:k]
				j = k + 1
			}
		}
	}
	rowStart, rowEnd, row, end, ok := cellRow(src, j)
	if !ok {
		return refHit{}, false
	}
	h.rowStart, h.rowEnd = rowStart, rowEnd
	h.row = row
	h.end = end
	return h, true
}

func cellRow(src string, i int) (rowStart, rowEnd, row, end int, ok bool) {
	if i < len(src) && src[i] == '$' {
		i++
	}
	nameStart := i
	for i < len(src) && isLetter(src[i]) {
		i++
	}
	if i == nameStart {
		return 0, 0, 0, 0, false
	}
	if _, good := ColIndex(src[nameStart:i]); !good {
		return 0, 0, 0, 0, false
	}
	if i < len(src) && src[i] == '$' {
		i++
	}
	rowStart = i
	for i < len(src) && isDigit(src[i]) {
		i++
	}
	if i == rowStart {
		return 0, 0, 0, 0, false
	}
	n, err := strconv.Atoi(src[rowStart:i])
	if err != nil || n < 1 {
		return 0, 0, 0, 0, false
	}
	return rowStart, i, n - 1, i, true
}

func hitApplies(h refHit, inherit string, local bool, sheetName string) bool {
	name := h.sheet
	if name == "" {
		name = inherit
	}
	if name == "" {
		return local
	}
	return strings.EqualFold(name, sheetName)
}

func adjustOne(src string, h refHit, applies bool, at int) (string, bool) {
	if !applies || h.row < at {
		return src[h.start:h.end], true
	}
	if h.row+1 >= MaxRow {
		return "", false
	}
	return src[h.start:h.rowStart] + strconv.Itoa(h.row+2) + src[h.rowEnd:h.end], true
}

func adjustRange(src string, a, b refHit, local bool, sheetName string, at int) (string, bool) {
	left := hitApplies(a, "", local, sheetName)
	inherit := a.sheet
	if inherit == "" && left {
		inherit = sheetName
	}
	right := hitApplies(b, inherit, local, sheetName)
	if !left && !right {
		return src[a.start:b.end], true
	}
	if left != right {
		leftText, ok := adjustOne(src, a, left, at)
		if !ok {
			return "", false
		}
		rightText, ok := adjustOne(src, b, right, at)
		if !ok {
			return "", false
		}
		return leftText + src[a.end:b.start] + rightText, true
	}
	lo, hi := a.row, b.row
	if lo > hi {
		lo, hi = hi, lo
	}
	shiftA, shiftB := false, false
	switch {
	case at <= lo:
		shiftA, shiftB = true, true
	case at <= hi:
		shiftA = a.row >= at
		shiftB = b.row >= at
	case at == hi+1:
		if a.row != b.row {
			shiftA = a.row == hi
			shiftB = b.row == hi
		} else {
			shiftB = true
		}
	}
	if shiftA && a.row+1 >= MaxRow || shiftB && b.row+1 >= MaxRow {
		return "", false
	}
	leftText := src[a.start:a.end]
	if shiftA {
		leftText = src[a.start:a.rowStart] + strconv.Itoa(a.row+2) + src[a.rowEnd:a.end]
	}
	rightText := src[b.start:b.end]
	if shiftB {
		rightText = src[b.start:b.rowStart] + strconv.Itoa(b.row+2) + src[b.rowEnd:b.end]
	}
	return leftText + src[a.end:b.start] + rightText, true
}

func skipSpaceBytes(src string, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return i
}
