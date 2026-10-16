package scout

import (
	"bytes"
	"testing"
)

func TestScoutListsTools(t *testing.T) {
	out, code := call("c3po", "scout")
	for _, name := range []string{"\n  ast", "\n  vibetree"} {
		if code != 0 || !bytes.Contains(out, []byte(name)) {
			t.Fatalf("code %d missing %q\n%s", code, name, out)
		}
	}
	for _, gone := range []string{"\n  complexity", "\n  hide", "\n  visualize"} {
		if bytes.Contains(out, []byte(gone)) {
			t.Fatalf("old name still listed %q\n%s", gone, out)
		}
	}
}

func TestScoutVibetreeHelp(t *testing.T) {
	out, code := call("c3po", "scout", "vibetree", "help")
	if code != 0 || !bytes.Contains(out, []byte("-p N")) || !bytes.Contains(out, []byte("-m loc|cyclo")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestRetiredScoutsAreRejected(t *testing.T) {
	for _, name := range []string{"complexity", "hide", "visualize"} {
		out, code := call("c3po", "scout", name)
		if code != 2 || !bytes.Contains(out, []byte("unknown scout")) {
			t.Fatalf("%s code %d\n%s", name, code, out)
		}
	}
}

func TestScoutAstHelp(t *testing.T) {
	out, code := call("c3po", "scout", "ast", "help")
	if code != 0 || !bytes.Contains(out, []byte("patch")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestNamesSelectTheScout(t *testing.T) {
	astOut, astCode := call("rgw-ast", "help")
	if astCode != 0 || !bytes.Contains(astOut, []byte("patch")) {
		t.Fatalf("ast %d\n%s", astCode, astOut)
	}
}

func TestUnknownScout(t *testing.T) {
	out, code := call("c3po", "scout", "nope")
	if code != 2 || !bytes.Contains(out, []byte("unknown scout")) || !bytes.Contains(out, []byte("\n  vibetree")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestVersion(t *testing.T) {
	out, code := call("c3po", "version")
	if code != 0 || !bytes.Contains(out, []byte("c3po ")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func call(args ...string) ([]byte, int) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	if errb.Len() > 0 {
		_, _ = out.Write(errb.Bytes())
	}
	return out.Bytes(), code
}
