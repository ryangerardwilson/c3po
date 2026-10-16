package parse

import "testing"

func TestGoCognitive(t *testing.T) {
	src := `package p
func myMethod(condition bool) {
	if condition {
		for i := 0; i < 10; i++ {
			for condition {
			}
		}
	}
}
func chain(a, b bool) {
	if a {
	} else if b {
	} else {
	}
}
func flags(a, b, c, d, e, f bool) {
	if a && b && c || d || e && f {
	}
}
func outer() {
	inner := func() {
		if true {
		}
	}
	_ = inner
}
func plain(a, b, c int) int { return a + b + c }
func (s *Server) Handle(a int) {}
`
	fns, diags := ParseGo("p.go", []byte(src))
	if len(diags) != 0 {
		t.Fatal(diags)
	}
	by := map[string]int{}
	nest := map[string]int{}
	params := map[string]int{}
	for _, fn := range fns {
		by[fn.Name] = fn.Cognitive
		nest[fn.Name] = fn.Nesting
		params[fn.Name] = fn.Params
	}
	check := func(name string, cog, n, p int) {
		t.Helper()
		if by[name] != cog || nest[name] != n || params[name] != p {
			t.Errorf("%s cog=%d nest=%d params=%d, want %d %d %d", name, by[name], nest[name], params[name], cog, n, p)
		}
	}
	check("myMethod", 6, 3, 1)
	check("chain", 3, 1, 2)
	check("flags", 4, 1, 6)
	check("outer", 1, 0, 0)
	check("plain", 0, 0, 3)
	check("Server.Handle", 0, 0, 1)
	foundInner := false
	for _, fn := range fns {
		if len(fn.Name) >= 5 && fn.Name[:5] == "func@" && fn.Cognitive == 1 && fn.Nesting == 1 {
			foundInner = true
		}
	}
	if !foundInner {
		t.Fatalf("inner func missing: %#v", fns)
	}
}

func TestGoIgnoresStringAndComment(t *testing.T) {
	src := "package p\nfunc f() {\n\t_ = \"if for while\"\n\t// if for\n}\n"
	fns, _ := ParseGo("p.go", []byte(src))
	if len(fns) != 1 || fns[0].Cognitive != 0 {
		t.Fatalf("%#v", fns)
	}
}
