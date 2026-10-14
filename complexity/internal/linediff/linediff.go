// Package linediff maps the lines of a new text back to an old text.
package linediff

import "strings"

// maxCells bounds the quadratic table used for one unmatched region.
const maxCells = 1 << 20

// Map is a line correspondence. Back[n] is the old 1-based line that new
// line n kept, or 0 when new line n is new.
type Map struct {
	Back []int
}

// Lines compares old and new text line by line. Leading and trailing
// spaces are ignored, so a re-indented line is not new.
func Lines(old, cur string) Map {
	a, b := split(old), split(cur)
	back := make([]int, len(b)+1)
	match(a, b, span{0, len(a), 0, len(b)}, back)
	return Map{Back: back}
}

// Old returns the old line for new line n.
func (m Map) Old(n int) (int, bool) {
	if n < 1 || n >= len(m.Back) || m.Back[n] == 0 {
		return 0, false
	}
	return m.Back[n], true
}

// Fresh reports whether new line n has no old counterpart.
func (m Map) Fresh(n int) bool {
	_, ok := m.Old(n)
	return !ok
}

func split(text string) []string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return lines
}

type span struct{ alo, ahi, blo, bhi int }

func match(a, b []string, s span, back []int) {
	s = trimEnds(a, b, s, back)
	if s.alo == s.ahi || s.blo == s.bhi {
		return
	}
	if (s.ahi-s.alo)*(s.bhi-s.blo) <= maxCells {
		lcs(a, b, s, back)
		return
	}
	anchors := unique(a, b, s)
	prev := span{s.alo, s.alo, s.blo, s.blo}
	for _, p := range anchors {
		back[p[1]+1] = p[0] + 1
		match(a, b, span{prev.ahi, p[0], prev.bhi, p[1]}, back)
		prev = span{p[0] + 1, p[0] + 1, p[1] + 1, p[1] + 1}
	}
	if len(anchors) > 0 {
		match(a, b, span{prev.ahi, s.ahi, prev.bhi, s.bhi}, back)
	}
}

func trimEnds(a, b []string, s span, back []int) span {
	for s.alo < s.ahi && s.blo < s.bhi && a[s.alo] == b[s.blo] {
		back[s.blo+1] = s.alo + 1
		s.alo++
		s.blo++
	}
	for s.alo < s.ahi && s.blo < s.bhi && a[s.ahi-1] == b[s.bhi-1] {
		back[s.bhi] = s.ahi
		s.ahi--
		s.bhi--
	}
	return s
}

// lcs fills back from a longest common subsequence of one region.
func lcs(a, b []string, s span, back []int) {
	n, m := s.ahi-s.alo, s.bhi-s.blo
	t := make([]int32, (n+1)*(m+1))
	at := func(i, j int) *int32 { return &t[i*(m+1)+j] }
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			*at(i, j) = best(a[s.alo+i] == b[s.blo+j], *at(i+1, j+1), *at(i+1, j), *at(i, j+1))
		}
	}
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case a[s.alo+i] == b[s.blo+j]:
			back[s.blo+j+1] = s.alo + i + 1
			i++
			j++
		case *at(i+1, j) >= *at(i, j+1):
			i++
		default:
			j++
		}
	}
}

func best(same bool, diag, down, right int32) int32 {
	if same {
		return diag + 1
	}
	if down > right {
		return down
	}
	return right
}

// unique returns lines that occur once in each region, in an increasing
// order on both sides, as patience diff anchors.
func unique(a, b []string, s span) [][2]int {
	ca := counts(a[s.alo:s.ahi])
	cb := counts(b[s.blo:s.bhi])
	pos := map[string]int{}
	for i := s.alo; i < s.ahi; i++ {
		if ca[a[i]] == 1 && cb[a[i]] == 1 {
			pos[a[i]] = i
		}
	}
	var pairs [][2]int
	for j := s.blo; j < s.bhi; j++ {
		if i, ok := pos[b[j]]; ok {
			pairs = append(pairs, [2]int{i, j})
		}
	}
	return increasing(pairs)
}

func counts(lines []string) map[string]int {
	out := make(map[string]int, len(lines))
	for _, l := range lines {
		out[l]++
	}
	return out
}

// increasing keeps the longest run of pairs whose old index rises.
func increasing(pairs [][2]int) [][2]int {
	if len(pairs) == 0 {
		return nil
	}
	tails := []int{}
	prev := make([]int, len(pairs))
	for k, p := range pairs {
		lo := search(tails, pairs, p[0])
		prev[k] = -1
		if lo > 0 {
			prev[k] = tails[lo-1]
		}
		if lo == len(tails) {
			tails = append(tails, k)
		} else {
			tails[lo] = k
		}
	}
	out := make([][2]int, len(tails))
	for k, i := tails[len(tails)-1], len(tails)-1; i >= 0; i-- {
		out[i] = pairs[k]
		k = prev[k]
	}
	return out
}

func search(tails []int, pairs [][2]int, v int) int {
	lo, hi := 0, len(tails)
	for lo < hi {
		mid := (lo + hi) / 2
		if pairs[tails[mid]][0] < v {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}
