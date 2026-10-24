package clip

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestWLCopyReturnsWhileTheServerStaysOpen(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "clipboard")
	pidfile := filepath.Join(dir, "server")
	script := "#!/bin/sh\nset -eu\ncat > \"$CLIP_OUT\"\ntrap '' HUP\nsleep 30 &\necho $! > \"$CLIP_PID\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "wl-copy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	t.Setenv("CLIP_OUT", out)
	t.Setenv("CLIP_PID", pidfile)
	t.Cleanup(func() {
		raw, err := os.ReadFile(pidfile)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil {
			return
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			return
		}
		_ = proc.Kill()
	})
	done := make(chan error, 1)
	go func() {
		done <- (WLCopy{}).Copy("copied-text", &bytes.Buffer{})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("copy stayed blocked while the clipboard server held stderr")
	}
	got, _ := os.ReadFile(out)
	if string(got) != "copied-text" {
		t.Fatalf("clipboard = %q", got)
	}
}
