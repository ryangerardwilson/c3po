package scout

import (
	"bytes"
	"testing"
)

func TestScoutListsBothTools(t *testing.T) {
	out, code := call("c3po", "scout")
	if code != 0 || !bytes.Contains(out, []byte("\n  ast")) || !bytes.Contains(out, []byte("\n  complexity")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestScoutAstHelp(t *testing.T) {
	out, code := call("c3po", "scout", "ast", "help")
	if code != 0 || !bytes.Contains(out, []byte("patch")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestScoutComplexityHelp(t *testing.T) {
	out, code := call("c3po", "scout", "complexity", "help")
	if code != 0 || !bytes.Contains(out, []byte("note")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestNamesSelectTheScout(t *testing.T) {
	astOut, astCode := call("rgw-ast", "help")
	cxOut, cxCode := call("cx", "help")
	fullOut, fullCode := call("complexity", "help")
	if astCode != 0 || cxCode != 0 || fullCode != 0 {
		t.Fatalf("ast %d cx %d complexity %d", astCode, cxCode, fullCode)
	}
	if !bytes.Contains(astOut, []byte("patch")) || !bytes.Contains(cxOut, []byte("note")) || !bytes.Contains(fullOut, []byte("budget")) {
		t.Fatalf("ast %s\ncx %s\ncomplexity %s", astOut, cxOut, fullOut)
	}
}

func TestUnknownScout(t *testing.T) {
	out, code := call("c3po", "scout", "nope")
	if code != 2 || !bytes.Contains(out, []byte("unknown scout")) {
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
