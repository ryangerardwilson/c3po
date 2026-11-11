package gocyclo

import "testing"

const rules = `package p

type T[K any] struct{}

func Plain() {}

func (t *T[K]) Ptr() {}

func (t T[K]) Val() {}

func IfElse(x int) {
	if x > 0 {
	} else if x < 0 {
	} else {
	}
}

func Loops(xs []int) {
	for i := 0; i < 3; i++ {
	}
	for range xs {
	}
}

func Switches(x any) {
	switch x {
	case 1, 2:
	case 3:
	default:
	}
	switch {
	}
	switch x.(type) {
	case int:
	}
}

func Selects(a, b chan int) {
	select {
	case <-a:
	case <-b:
	}
}

func Bools(a, b, c bool) bool {
	if a && (b || c) {
	}
	return a && b || c
}

func Lit() {
	f := func(x int) {
		if x > 0 && x < 9 {
		}
	}
	f(1)
}
`

func TestFileRules(t *testing.T) {
	got, ok := File([]byte(rules))
	if !ok {
		t.Fatal("rules did not parse")
	}
	want := map[string]int{
		"Plain":    1,
		"T.Ptr":    1,
		"T.Val":    1,
		"IfElse":   3,
		"Loops":    3,
		"Switches": 6,
		"Selects":  4,
		"Bools":    6,
		"Lit":      1,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for name, cc := range want {
		if got[name] != cc {
			t.Errorf("%s: cc %d, want %d", name, got[name], cc)
		}
	}
}

func TestFileUnclean(t *testing.T) {
	if _, ok := File([]byte("package p\nfunc F( {\n")); ok {
		t.Fatal("syntax error parsed clean")
	}
	if _, ok := File([]byte("package p\nfunc init() {}\nfunc init() {}\n")); ok {
		t.Fatal("duplicate names parsed clean")
	}
	if _, ok := File([]byte("package p\ntype A struct{}\ntype B struct{}\nfunc (A) M() {}\nfunc (*B) M() {}\n")); !ok {
		t.Fatal("A.M and B.M are distinct")
	}
}
