package parse

import (
	"os"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/gocyclo"
)

// TestParseGoCyclomaticIsGocyclo checks that every Go declaration cyclotree
// scores has the cyclomatic number internal/gocyclo gives it.
func TestParseGoCyclomaticIsGocyclo(t *testing.T) {
	for _, file := range []string{"golang.go", "score.go", "python.go", "golang_test.go"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		want, ok := gocyclo.File(src)
		if !ok {
			t.Fatalf("%s did not parse clean", file)
		}
		fns, _ := ParseGo(file, src)
		decls := 0
		for _, fn := range fns {
			if strings.HasPrefix(fn.Name, "func@") {
				continue
			}
			decls++
			if cc, ok := want[fn.Name]; !ok || cc != fn.Cyclomatic {
				t.Errorf("%s:%s cyclotree %d, gocyclo %d (found %v)", file, fn.Name, fn.Cyclomatic, cc, ok)
			}
		}
		if decls != len(want) || decls == 0 {
			t.Errorf("%s: cyclotree has %d declarations, gocyclo %d", file, decls, len(want))
		}
	}
}

func TestParseGoCyclomaticRules(t *testing.T) {
	src := []byte("package p\n\nfunc F(a, b bool, x any, c chan int) {\n" +
		"\tif a && b || a {\n\t} else if b {\n\t} else {\n\t}\n" +
		"\tswitch x {\n\t}\n" +
		"\tselect {\n\tcase <-c:\n\t}\n" +
		"\tg := func() {\n\t\tif a {\n\t\t}\n\t}\n\tg()\n}\n")
	fns, _ := ParseGo("f.go", src)
	want, _ := gocyclo.File(src)
	for _, fn := range fns {
		if fn.Name != "F" {
			continue
		}
		// 1 + if 1 + && || 2 + else if 1 + empty switch 1 + select 1 + one comm clause 1.
		if fn.Cyclomatic != 8 || want["F"] != 8 {
			t.Fatalf("cyclotree %d gocyclo %d, want 8", fn.Cyclomatic, want["F"])
		}
		return
	}
	t.Fatal("F not scored")
}
