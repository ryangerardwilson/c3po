package dup

import "testing"

func TestFindClone(t *testing.T) {
	body := "alpha := 1\nbeta := alpha + 2\nif beta > 3 {\n\tbeta = beta - 1\n}\nreturn beta\n"
	files := []File{{Path: "a.go", Src: "func a() {\n" + body + "}\n"}, {Path: "b.go", Src: "func b() {\n" + body + "}\n"}}
	got := Find(files, 6)
	if len(got) == 0 || len(got[0].Occurrences) != 2 {
		t.Fatalf("%#v", got)
	}
}
