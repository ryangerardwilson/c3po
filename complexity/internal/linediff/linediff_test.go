package linediff

import (
	"strings"
	"testing"
)

func TestInsertAboveKeepsLines(t *testing.T) {
	old := "a\nb\nc\n"
	cur := "x\na\nb\nc\n"
	m := Lines(old, cur)
	if !m.Fresh(1) {
		t.Fatal("line 1 should be new")
	}
	for n, want := range map[int]int{2: 1, 3: 2, 4: 3} {
		if got, ok := m.Old(n); !ok || got != want {
			t.Fatalf("line %d -> %d %v, want %d", n, got, ok, want)
		}
	}
}

func TestReindentIsNotNew(t *testing.T) {
	m := Lines("f()\n", "if x {\n\tf()\n}\n")
	if !m.Fresh(1) || m.Fresh(2) || !m.Fresh(3) {
		t.Fatalf("%v", m.Back)
	}
}

func TestLargeRegionUsesAnchors(t *testing.T) {
	var a, b []string
	for i := 0; i < 1500; i++ {
		a = append(a, "same", "a"+itoa(i))
		b = append(b, "same", "a"+itoa(i))
	}
	b = append([]string{"head"}, b...)
	b[1500] = "changed"
	m := Lines(strings.Join(a, "\n"), strings.Join(b, "\n"))
	if !m.Fresh(1) || !m.Fresh(1501) {
		t.Fatal("inserted and changed lines should be new")
	}
	if got, ok := m.Old(3001); !ok || got != 3000 {
		t.Fatalf("last line -> %d %v", got, ok)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
