package screen

import (
	"bytes"
	"testing"
)

func TestUnknownScreen(t *testing.T) {
	for _, name := range []string{"nope", "nlp", "idealake", "clippy", "tui"} {
		out, errb, code := run(name)
		if code != 2 || !bytes.Contains(errb, []byte("unknown screen")) {
			t.Fatalf("%s %d\n%s%s", name, code, out, errb)
		}
	}
}

func TestCyclotreeHelp(t *testing.T) {
	out, _, code := run("cyclotree", "help")
	if code != 0 || !bytes.Contains(out, []byte("j and k")) || !bytes.Contains(out, []byte("h moves")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestTunnelHelp(t *testing.T) {
	out, _, code := run("tunnel", "help")
	if code != 0 || !bytes.Contains(out, []byte("r2d2 tunnel")) || bytes.Contains(out, []byte("c3po tunnel open")) {
		t.Fatalf("help %d\n%s", code, out)
	}
	_, errb, code := run("tunnel", "extra")
	if code != 2 || !bytes.Contains(errb, []byte("r2d2 tunnel")) {
		t.Fatalf("extra %d\n%s", code, errb)
	}
}

func TestGDriveHelp(t *testing.T) {
	out, _, code := run("gdrive", "help")
	for _, want := range []string{"preset", "browse", "browser", "uploads", "dd deletes"} {
		if code != 0 || !bytes.Contains(out, []byte(want)) {
			t.Fatalf("help %d missing %q\n%s", code, want, out)
		}
	}
}

func TestPassHelp(t *testing.T) {
	out, _, code := run("pass", "help")
	for _, want := range []string{"vault", "item", "enter copies the selected item", "secret is not printed", "dd deletes the selected item"} {
		if code != 0 || !bytes.Contains(out, []byte(want)) {
			t.Fatalf("help %d missing %q\n%s", code, want, out)
		}
	}
	_, errb, code := run("pass", "extra")
	if code != 2 || !bytes.Contains(errb, []byte("r2d2 pass")) {
		t.Fatalf("extra %d\n%s", code, errb)
	}
}

func TestDeportHelp(t *testing.T) {
	out, _, code := run("deport", "help")
	for _, want := range []string{"8000 through 8999", "j and k move", "dd frees the selected port"} {
		if code != 0 || !bytes.Contains(out, []byte(want)) {
			t.Fatalf("help %d missing %q\n%s", code, want, out)
		}
	}
	_, errb, code := run("deport", "extra")
	if code != 2 || !bytes.Contains(errb, []byte("r2d2 deport")) {
		t.Fatalf("extra %d\n%s", code, errb)
	}
}

func TestXLHelp(t *testing.T) {
	out, _, code := run("xl", "help")
	if code != 0 || !bytes.Contains(out, []byte("xlsx")) || !bytes.Contains(out, []byte("r2d2 xl")) {
		t.Fatalf("help %d\n%s", code, out)
	}
	out, _, code = run("xl", "version")
	if code != 0 || string(out) != "0.1.0\n" {
		t.Fatalf("version %d\n%s", code, out)
	}
	_, _, code = run("xl", "one.xlsx", "two.xlsx")
	if code != 1 {
		t.Fatalf("two paths %d", code)
	}
}

func TestSlackHelp(t *testing.T) {
	out, _, code := run("slack", "help")
	for _, want := range []string{"r2d2 slack", "h and l switch workspace", "Opening a conversation does not"} {
		if code != 0 || !bytes.Contains(out, []byte(want)) {
			t.Fatalf("help %d missing %q\n%s", code, want, out)
		}
	}
	_, errb, code := run("slack", "extra")
	if code != 2 || !bytes.Contains(errb, []byte("r2d2 slack")) {
		t.Fatalf("extra %d\n%s", code, errb)
	}
}

func TestLooptabHelp(t *testing.T) {
	out, _, code := run("looptab", "help")
	if code != 0 || !bytes.Contains(out, []byte("enter runs the selected job now")) || !bytes.Contains(out, []byte("r2d2 looptab")) {
		t.Fatalf("help %d\n%s", code, out)
	}
	out, _, code = run("looptab", "version")
	if code != 0 || !bytes.Contains(out, []byte("c3po 0.5.0")) {
		t.Fatalf("version %d\n%s", code, out)
	}
	_, errb, code := run("looptab", "extra")
	if code != 2 || !bytes.Contains(errb, []byte("r2d2 looptab")) {
		t.Fatalf("extra %d\n%s", code, errb)
	}
}

func run(name string, args ...string) ([]byte, []byte, int) {
	var out, errb bytes.Buffer
	code := Run(name, args, &out, &errb)
	return out.Bytes(), errb.Bytes(), code
}
