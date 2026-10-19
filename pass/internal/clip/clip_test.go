package clip

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWLCopySendsTextOnStdin(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "clipboard")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s' \"$#\" > \"$CLIP_ARGC\"\ncat > \"$CLIP_OUT\"\n"
	if err := os.WriteFile(filepath.Join(dir, "wl-copy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	t.Setenv("CLIP_OUT", out)
	t.Setenv("CLIP_ARGC", filepath.Join(dir, "argc"))
	if err := (WLCopy{}).Copy("copied-text", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	argc, _ := os.ReadFile(filepath.Join(dir, "argc"))
	if string(got) != "copied-text" || string(argc) != "0" {
		t.Fatalf("clipboard = %q argc = %q", got, argc)
	}
}

func TestWLCopyIsRequired(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := (WLCopy{}).Copy("x", &bytes.Buffer{})
	if err == nil || err.Error() != "wl-copy is required." {
		t.Fatalf("err = %v", err)
	}
}
