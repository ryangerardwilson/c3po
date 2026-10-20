package scout

import (
	"bytes"
	"testing"
)

var portedScouts = []string{"deport", "gcal", "gdrive", "gmail", "pass", "tunnel", "tui"}

var retiredScouts = []string{"complexity", "hide", "visualize", "vibetree", "iha", "cx"}

func TestScoutListsTools(t *testing.T) {
	out, code := call("c3po")
	names := []string{"\n  ast", "\n  tui"}
	for _, name := range portedScouts {
		names = append(names, "\n  "+name+" ")
	}
	for _, name := range names {
		if code != 0 || !bytes.Contains(out, []byte(name)) {
			t.Fatalf("code %d missing %q\n%s", code, name, out)
		}
	}
	for _, name := range retiredScouts {
		if gone := "\n  " + name + " "; bytes.Contains(out, []byte(gone)) {
			t.Fatalf("old name still listed %q\n%s", gone, out)
		}
	}
	for _, gone := range []string{"\n  cyclotree", "\n  l ", "\n  o "} {
		if bytes.Contains(out, []byte(gone)) {
			t.Fatalf("screen still listed as an app %q\n%s", gone, out)
		}
	}
}

func TestScoutCyclotreeHelp(t *testing.T) {
	out, code := call("c3po", "tui", "cyclotree", "help")
	if code != 0 || !bytes.Contains(out, []byte("j and k")) || !bytes.Contains(out, []byte("h moves")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestRetiredScoutsAreRejected(t *testing.T) {
	for _, name := range retiredScouts {
		out, code := call("c3po", name)
		if code != 2 || !bytes.Contains(out, []byte("unknown command")) {
			t.Fatalf("%s code %d\n%s", name, code, out)
		}
	}
	out, code := call("c3po", "scout")
	if code != 2 || !bytes.Contains(out, []byte("unknown command")) {
		t.Fatalf("scout code %d\n%s", code, out)
	}
}

func TestScoutAstHelp(t *testing.T) {
	out, code := call("c3po", "ast", "help")
	if code != 0 || !bytes.Contains(out, []byte("patch")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestNamesSelectTheScout(t *testing.T) {
	astOut, astCode := call("rgw-ast", "help")
	if astCode != 0 || !bytes.Contains(astOut, []byte("patch")) {
		t.Fatalf("ast %d\n%s", astCode, astOut)
	}
	for _, name := range portedScouts {
		out, code := call(name, "list")
		if code != 2 || !bytes.Contains(out, []byte("unknown command")) {
			t.Fatalf("argv0 %s selected a scout: %d\n%s", name, code, out)
		}
	}
}

func TestPortedScoutHelpMatchesHelpFlag(t *testing.T) {
	for _, name := range portedScouts {
		scoutOut, scoutCode := call("c3po", name, "help")
		helpOut, helpCode := call("c3po", "help", "-f", name)
		if scoutCode != 0 || helpCode != 0 {
			t.Fatalf("%s codes scout=%d help=%d\n%s\n%s", name, scoutCode, helpCode, scoutOut, helpOut)
		}
		if !bytes.Equal(scoutOut, helpOut) {
			t.Fatalf("%s help differs\n%s\n---\n%s", name, scoutOut, helpOut)
		}
		if !bytes.Contains(scoutOut, []byte("c3po "+name)) {
			t.Fatalf("%s help does not name its command\n%s", name, scoutOut)
		}
		out, code := call("c3po", "help", "-f", name, "extra")
		if code != 2 || !bytes.Contains(out, []byte("unknown feature")) {
			t.Fatalf("%s extra %d\n%s", name, code, out)
		}
	}
}

func TestHelpListsPortedScouts(t *testing.T) {
	out, code := call("c3po", "help")
	for _, name := range portedScouts {
		if code != 0 || !bytes.Contains(out, []byte("c3po "+name)) {
			t.Fatalf("help %d missing %s\n%s", code, name, out)
		}
	}
}

func TestUnknownScout(t *testing.T) {
	out, code := call("c3po", "nope")
	if code != 2 || !bytes.Contains(out, []byte("unknown command")) || !bytes.Contains(out, []byte("\n  tui")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestHelpFlagSelectsAFeature(t *testing.T) {
	out, code := call("c3po", "help")
	for _, want := range []string{"c3po help -f", "\n  ast", "\n  tui"} {
		if code != 0 || !bytes.Contains(out, []byte(want)) {
			t.Fatalf("index %d missing %q\n%s", code, want, out)
		}
	}
	if bytes.Contains(out, []byte("\n  vibetree")) {
		t.Fatalf("vibetree still listed\n%s", out)
	}
	out, code = call("c3po", "help", "-f", "tui", "cyclotree")
	if code != 0 || !bytes.Contains(out, []byte("j and k")) || !bytes.Contains(out, []byte("h moves")) {
		t.Fatalf("cyclotree %d\n%s", code, out)
	}
	for _, name := range retiredScouts {
		out, code = call("c3po", "help", "-f", name)
		if code != 2 || !bytes.Contains(out, []byte("unknown category")) {
			t.Fatalf("%s %d\n%s", name, code, out)
		}
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
	out, code = call("c3po", "help", "-f", "tui", "nope")
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
	if code != 0 || !bytes.Contains(out, []byte("c3po 0.5.0")) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestScreensAreOnlyUnderTUI(t *testing.T) {
	for _, name := range []string{"cyclotree", "l", "o"} {
		out, code := call("c3po", name, "help")
		if code != 2 || !bytes.Contains(out, []byte("unknown command")) {
			t.Fatalf("%s code %d\n%s", name, code, out)
		}
		out, code = call("c3po", "help", "-f", name)
		if code != 2 || !bytes.Contains(out, []byte("unknown category")) {
			t.Fatalf("help %s code %d\n%s", name, code, out)
		}
	}
	out, code := call("c3po", "tunnel", "open")
	if code != 2 || bytes.Contains(out, []byte("c3po tunnel open")) || !bytes.Contains(out, []byte("c3po tui tunnel")) {
		t.Fatalf("tunnel open %d\n%s", code, out)
	}
	out, code = call("c3po", "tui")
	for _, name := range []string{"\n  cyclotree", "\n  l ", "\n  o ", "\n  tunnel ", "\n  gdrive "} {
		if code != 0 || !bytes.Contains(out, []byte(name)) {
			t.Fatalf("tui %d missing %q\n%s", code, name, out)
		}
	}
	out, code = call("c3po", "tui", "nope")
	if code != 2 || !bytes.Contains(out, []byte("unknown tui")) {
		t.Fatalf("nope %d\n%s", code, out)
	}
	out, code = call("c3po", "tui", "tunnel", "help")
	if code != 0 || !bytes.Contains(out, []byte("c3po tui tunnel")) || bytes.Contains(out, []byte("c3po tunnel open")) {
		t.Fatalf("screen help %d\n%s", code, out)
	}
	out, code = call("c3po", "tui", "tunnel", "extra")
	if code != 2 || !bytes.Contains(out, []byte("c3po tui tunnel")) {
		t.Fatalf("extra %d\n%s", code, out)
	}
}

func TestGDriveScreenHelp(t *testing.T) {
	out, code := call("c3po", "tui", "gdrive", "help")
	if code != 0 || !bytes.Contains(out, []byte("preset")) || !bytes.Contains(out, []byte("c3po tui gdrive")) {
		t.Fatalf("help %d\n%s", code, out)
	}
	out, code = call("c3po", "help", "-f", "tui", "gdrive")
	direct, directCode := call("c3po", "tui", "gdrive", "help")
	if code != 0 || directCode != 0 || !bytes.Equal(out, direct) {
		t.Fatalf("flag %d %d\n%s", code, directCode, out)
	}
	out, code = call("c3po", "gdrive", "1", "browse")
	if code != 2 || !bytes.Contains(out, []byte("unknown command")) {
		t.Fatalf("browse %d\n%s", code, out)
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
