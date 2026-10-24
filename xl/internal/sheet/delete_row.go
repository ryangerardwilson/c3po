package sheet

import (
	"strconv"
	"strings"
)

// DeleteRow removes row at and shifts later rows up. References to this
// sheet move with those rows. A reference to the removed row becomes
// #REF!. A range that contained the row shrinks.
func (s *Sheet) DeleteRow(at int) bool {
	if s == nil || at < 0 || at >= s.Rows {
		return false
	}
	sheets := []*Sheet{s}
	if s.book != nil && len(s.book.Sheets) > 0 {
		sheets = s.book.Sheets
	}
	type rewrite struct {
		sh   *Sheet
		addr Addr
		src  string
	}
	var edits []rewrite
	for _, sh := range sheets {
		if sh == nil {
			continue
		}
		for addr, src := range sh.cells {
			next := dropFormulaRows(src, s.Name, sh == s, at)
			if next != src {
				edits = append(edits, rewrite{sh, addr, next})
			}
		}
	}
	for _, ed := range edits {
		ed.sh.cells[ed.addr] = ed.src
	}
	if len(s.cells) > 0 {
		next := make(map[Addr]string, len(s.cells))
		for addr, src := range s.cells {
			if addr.R == at {
				continue
			}
			if addr.R > at {
				addr.R--
			}
			next[addr] = src
		}
		s.cells = next
	}
	if s.Rows > 1 {
		s.Rows--
	}
	s.touch()
	return true
}

func dropFormulaRows(src, sheetName string, local bool, at int) string {
	if !strings.HasPrefix(strings.TrimSpace(src), "=") {
		return src
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
				b.WriteString(dropRange(src, hit, hit2, local, sheetName, at))
				i = hit2.end
				continue
			}
		}
		b.WriteString(dropOne(src, hit, hitApplies(hit, "", local, sheetName), at))
		i = hit.end
	}
	return b.String()
}

func dropOne(src string, h refHit, applies bool, at int) string {
	if !applies || h.row < at {
		return src[h.start:h.end]
	}
	if h.row == at {
		return "#REF!"
	}
	return src[h.start:h.rowStart] + strconv.Itoa(h.row) + src[h.rowEnd:h.end]
}

func dropRange(src string, a, b refHit, local bool, sheetName string, at int) string {
	left := hitApplies(a, "", local, sheetName)
	inherit := a.sheet
	if inherit == "" && left {
		inherit = sheetName
	}
	right := hitApplies(b, inherit, local, sheetName)
	if !left && !right {
		return src[a.start:b.end]
	}
	if left != right {
		return dropOne(src, a, left, at) + src[a.end:b.start] + dropOne(src, b, right, at)
	}
	lo, hi := a.row, b.row
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo == hi && lo == at {
		return "#REF!"
	}
	return dropEndpoint(src, a, at, lo) + src[a.end:b.start] + dropEndpoint(src, b, at, lo)
}

func dropEndpoint(src string, h refHit, at, lo int) string {
	if h.row > at || (h.row == at && h.row != lo) {
		return src[h.start:h.rowStart] + strconv.Itoa(h.row) + src[h.rowEnd:h.end]
	}
	return src[h.start:h.end]
}
