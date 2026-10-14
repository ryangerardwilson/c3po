package parse

import (
	"testing"

	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

func TestPythonPaper(t *testing.T) {
	src := []byte(`
def my_method(condition):
    try:
        if condition:
            for i in range(10):
                while condition:
                    pass
    except Exception:
        if condition:
            pass

def chain(a, b):
    if a:
        pass
    elif b:
        pass
    else:
        pass

def flags(a, b, c, d, e, f):
    if a and b and c or d or e and f:
        pass

def outer():
    if ready:
        def inner():
            if ready:
                pass
        return inner
`)
	fns, diags := ParsePython("m.py", src)
	if len(diags) != 0 {
		t.Fatal(diags)
	}
	by := indexFns(fns)
	checkFn(t, by, "my_method", 9, 3, 1)
	checkFn(t, by, "chain", 3, 1, 2)
	checkFn(t, by, "flags", 4, 1, 6)
	if by["outer"].Cognitive < 2 || by["inner"].Cognitive != 1 {
		t.Fatalf("nested outer=%d inner=%d", by["outer"].Cognitive, by["inner"].Cognitive)
	}
}

func TestBracePaper(t *testing.T) {
	src := []byte(`
function myMethod(condition) {
  try {
    if (condition) {
      for (let i = 0; i < 10; i++) {
        while (condition) {
        }
      }
    }
  } catch (e) {
    if (condition) {
    }
  }
}
function chain(a, b) {
  if (a) {
  } else if (b) {
  } else {
  }
}
function flags(a, b, c, d, e, f) {
  if (a && b && c || d || e && f) {
  }
}
`)
	fns, _ := File("m.ts", "typescript", src)
	by := indexFns(fns)
	checkFn(t, by, "myMethod", 9, 3, 1)
	checkFn(t, by, "chain", 3, 1, 2)
	checkFn(t, by, "flags", 4, 1, 6)
}

func TestRustIf(t *testing.T) {
	src := []byte(`
fn flags(a: bool, b: bool, c: bool, d: bool, e: bool, f: bool) {
    if a && b && c || d || e && f {
    }
}
`)
	fns, _ := File("m.rs", "rust", src)
	if len(fns) != 1 || fns[0].Cognitive != 4 || fns[0].Params != 6 || fns[0].Nesting != 1 {
		t.Fatalf("%#v", fns)
	}
}

func indexFns(fns []metric.Function) map[string]metric.Function {
	out := map[string]metric.Function{}
	for _, fn := range fns {
		out[fn.Name] = fn
	}
	return out
}

func checkFn(t *testing.T, by map[string]metric.Function, name string, cog, nest, params int) {
	t.Helper()
	fn, ok := by[name]
	if !ok {
		t.Fatalf("missing %s in %v", name, by)
	}
	if fn.Cognitive != cog || fn.Nesting != nest || fn.Params != params {
		t.Errorf("%s cog=%d nest=%d params=%d, want %d %d %d", name, fn.Cognitive, fn.Nesting, fn.Params, cog, nest, params)
	}
}
