package scout

import (
	"bytes"
	"testing"
)

func TestScoutListsTools(t *testing.T) {
	out, code := call("c3po", "scout")
	for _, name := range []string{"\n  ast", "\n  cyclotree"} {
		if code != 0 || !bytes.Contains(out, []byte(name)) {
			t.Fatalf("code %d missing %q\n%s", code, name, out)
		}
	}
	for _, gone := range []string{"\n  complexity", "\n  hide", "\n  visualize", "\n  vibetree"} {
		if bytes.Contains(out, []byte(gone)) {
			t.Fatalf("old name still listed %q\n%s", gone, out)
		}
	}
}

func TestScoutCyclotreeHelp(t *testing.T) {
	out, code := call("c3po", "scout", "cyclotree", "help")
	if code != 0 || !bytes.Contains(out, []byte("j and k")) || !bytes.Contains(out, []byte("h moves")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestRetiredScoutsAreRejected(t *testing.T) {
	for _, name := range []string{"complexity", "hide", "visualize", "vibetree"} {
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
	if code != 2 || !bytes.Contains(out, []byte("unknown scout")) || !bytes.Contains(out, []byte("\n  cyclotree")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestHelpFlagSelectsAFeature(t *testing.T) {
	out, code := call("c3po", "help")
	for _, want := range []string{"c3po help -f", "\n  ast", "\n  cyclotree"} {
		if code != 0 || !bytes.Contains(out, []byte(want)) {
			t.Fatalf("index %d missing %q\n%s", code, want, out)
		}
	}
	if bytes.Contains(out, []byte("\n  vibetree")) {
		t.Fatalf("vibetree still listed\n%s", out)
	}
	out, code = call("c3po", "help", "-f", "cyclotree")
	if code != 0 || !bytes.Contains(out, []byte("j and k")) || !bytes.Contains(out, []byte("h moves")) {
		t.Fatalf("cyclotree %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f", "vibetree")
	if code != 2 || !bytes.Contains(out, []byte("unknown category")) {
		t.Fatalf("vibetree %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f", "ast", "patch")
	if code != 0 || !bytes.Contains(out, []byte("expect-hash")) {
		t.Fatalf("patch %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f", "ast", "search")
	if code != 0 || !bytes.Contains(out, []byte("--regex")) {
		t.Fatalf("search %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f", "nope")
	if code != 2 || !bytes.Contains(out, []byte("unknown category")) {
		t.Fatalf("category %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f", "ast", "nope")
	if code != 2 || !bytes.Contains(out, []byte("unknown feature")) {
		t.Fatalf("feature %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f", "cyclotree", "extra")
	if code != 2 || !bytes.Contains(out, []byte("unknown feature")) {
		t.Fatalf("extra %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f")
	if code != 2 || !bytes.Contains(out, []byte("c3po help -f")) {
		t.Fatalf("bare %d\n%s", code, out)
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
