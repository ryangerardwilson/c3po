package vibetree

import "testing"

func TestCyclomaticFileSumsFunctions(t *testing.T) {
	goSrc := []byte("package p\n\nfunc A(x int) int {\n\tif x > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n\nfunc B() int {\n\treturn 1\n}\n")
	n, ok := cyclomaticFile("p.go", goSrc)
	if !ok || n != 3 {
		t.Fatalf("go %d %v", n, ok)
	}
	if _, ok := cyclomaticFile("notes.txt", []byte("plain\n")); ok {
		t.Fatal("text scored")
	}
	for _, c := range []struct{ path, src string }{
		{"a.py", "def a(x):\n    if x > 0:\n        return 1\n    return 0\n"},
		{"a.rs", "fn a(x: i32) -> i32 {\n    if x > 0 {\n        return 1;\n    }\n    0\n}\n"},
		{"a.ts", "function a(x: number): number {\n  if (x > 0) {\n    return 1;\n  }\n  return 0;\n}\n"},
		{"a.js", "function a(x) {\n  if (x > 0) {\n    return 1;\n  }\n  return 0;\n}\n"},
	} {
		n, ok := cyclomaticFile(c.path, []byte(c.src))
		if !ok || n < 2 {
			t.Fatalf("%s %d %v", c.path, n, ok)
		}
	}
}
