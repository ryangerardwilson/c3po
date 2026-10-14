package gate

import "github.com/ryangerardwilson/c3po/complexity/internal/metric"

// Back maps a current line of file to the line it had in the baseline.
// A nil Back keeps line numbers.
type Back func(file string, line int) (int, bool)

// Pair is a current function and its baseline. Base is nil for new code.
type Pair struct {
	Cur  metric.Function
	Base *metric.Function
}

// Match pairs each current function with its baseline in the same file.
// A named function matches by name. An anonymous function matches by body
// signature, then by a header line the edit kept, so a line inserted above
// it does not make it new. Ties go to the nearest line.
func Match(cur, base []metric.Function, back Back) []Pair {
	ix := newIndex(base, back)
	out := make([]Pair, len(cur))
	for i, fn := range cur {
		out[i].Cur = fn
		if metric.Anonymous(fn.Name) {
			out[i].Base = ix.take(fn, ix.sigs[fileKey(fn.File, fn.Sig)])
		} else {
			out[i].Base = ix.nearest(fn, ix.names[fileKey(fn.File, fn.Name)], false)
		}
	}
	for i := range out {
		if out[i].Base == nil && metric.Anonymous(out[i].Cur.Name) {
			out[i].Base = ix.header(out[i].Cur)
		}
	}
	return out
}

type index struct {
	base  []metric.Function
	used  []bool
	names map[string][]int
	sigs  map[string][]int
	heads map[string]int
	back  Back
}

func newIndex(base []metric.Function, back Back) *index {
	ix := &index{
		base: base, used: make([]bool, len(base)), back: back,
		names: map[string][]int{}, sigs: map[string][]int{}, heads: map[string]int{},
	}
	for i, fn := range base {
		if !metric.Anonymous(fn.Name) {
			k := fileKey(fn.File, fn.Name)
			ix.names[k] = append(ix.names[k], i)
			continue
		}
		if fn.Sig != "" {
			k := fileKey(fn.File, fn.Sig)
			ix.sigs[k] = append(ix.sigs[k], i)
		}
		ix.heads[headKey(fn.File, fn.Name, fn.Line)] = i
	}
	return ix
}

func (ix *index) old(fn metric.Function) (int, bool) {
	if ix.back == nil {
		return fn.Line, true
	}
	return ix.back(fn.File, fn.Line)
}

// take claims the nearest unclaimed anonymous baseline with fn's signature.
func (ix *index) take(fn metric.Function, group []int) *metric.Function {
	if fn.Sig == "" {
		return nil
	}
	return ix.nearest(fn, group, true)
}

func (ix *index) nearest(fn metric.Function, group []int, claim bool) *metric.Function {
	target, ok := ix.old(fn)
	if !ok {
		target = fn.Line
	}
	best := -1
	for _, i := range group {
		if claim && ix.used[i] {
			continue
		}
		if best < 0 || abs(ix.base[i].Line-target) < abs(ix.base[best].Line-target) {
			best = i
		}
	}
	return ix.claim(best, claim)
}

// header matches an edited anonymous function whose first line the edit kept.
func (ix *index) header(fn metric.Function) *metric.Function {
	line, ok := ix.old(fn)
	if !ok {
		return nil
	}
	i, found := ix.heads[headKey(fn.File, fn.Name, line)]
	if !found || ix.used[i] {
		return nil
	}
	return ix.claim(i, true)
}

func (ix *index) claim(i int, mark bool) *metric.Function {
	if i < 0 {
		return nil
	}
	if mark {
		ix.used[i] = true
	}
	fn := ix.base[i]
	return &fn
}

func fileKey(file, name string) string {
	return file + "\x00" + name
}

func headKey(file, name string, line int) string {
	return file + "\x00" + metric.AnonKind(name) + "\x00" + itoa(line)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// PassAt is the score that passes for one metric: the limit for new code
// and for code that was within the limit, and the previous score for code
// that was already over it.
func PassAt(limit int, base *metric.Function, name string) int {
	if base == nil {
		return limit
	}
	if old := metric.Value(*base, name); old > limit {
		return old
	}
	return limit
}
