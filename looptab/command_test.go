package looptab

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisterListShowRemove(t *testing.T) {
	d := testDesk(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "gmail")
	other := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(src, []byte("daily 5am c3po gmail tidy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("hourly echo notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := d.run([]string{"register", src}); code != 0 {
		t.Fatalf("register %d", code)
	}
	var err bytes.Buffer
	d.stderr = &err
	if code := d.run([]string{"register", src}); code != 2 || !strings.Contains(err.String(), "already registered") {
		t.Fatalf("duplicate %d %s", code, err.String())
	}
	if code := d.run([]string{"register", other}); code != 0 {
		t.Fatalf("second %d", code)
	}
	var out bytes.Buffer
	d.stdout = &out
	d.stderr = io.Discard
	if code := d.run([]string{"list"}); code != 0 || !strings.Contains(out.String(), "gmail\t"+src) || !strings.Contains(out.String(), "notes\t"+other) {
		t.Fatalf("list %d %s", code, out.String())
	}
	out.Reset()
	if code := d.run([]string{"show", "gmail"}); code != 0 || !strings.Contains(out.String(), "c3po gmail tidy") {
		t.Fatalf("show %d %s", code, out.String())
	}
	body, readErr := os.ReadFile(d.schedule)
	if readErr != nil || !strings.Contains(string(body), "Installed by c3po looptab") || !strings.Contains(string(body), "c3po gmail tidy") || !strings.Contains(string(body), "echo notes") {
		t.Fatalf("schedule %s %v", body, readErr)
	}
	if tidy := strings.Index(string(body), "c3po gmail tidy"); tidy < 0 || tidy > strings.Index(string(body), "echo notes") {
		t.Fatalf("order %s", body)
	}
	if code := d.run([]string{"remove", "gmail"}); code != 0 {
		t.Fatalf("remove %d", code)
	}
	if code := d.run([]string{"remove", "notes"}); code != 0 {
		t.Fatalf("remove notes %d", code)
	}
	out.Reset()
	if code := d.run([]string{"list"}); code != 0 || strings.TrimSpace(out.String()) != "no jobs files" {
		t.Fatalf("empty %d %s", code, out.String())
	}
	sched, readErr := os.ReadFile(d.schedule)
	if readErr != nil || strings.Contains(string(sched), "c3po gmail tidy") || !strings.Contains(string(sched), "Installed by c3po looptab") {
		t.Fatalf("cleared %s %v", sched, readErr)
	}
	err.Reset()
	d.stderr = &err
	if code := d.run([]string{"remove", "gmail"}); code != 2 || !strings.Contains(err.String(), "unknown jobs file") {
		t.Fatalf("second remove %d %s", code, err.String())
	}
}

func TestEditInstallsTheSavedDocument(t *testing.T) {
	d := testDesk(t)
	d.edit = func(path string) error {
		return os.WriteFile(path, []byte("hourly c3po gmail tidy\n"), 0o644)
	}
	var out bytes.Buffer
	d.stdout = &out
	if code := d.run(nil); code != 0 {
		t.Fatalf("edit %d", code)
	}
	body, err := os.ReadFile(d.schedule)
	if err != nil || !strings.Contains(string(body), "hourly c3po gmail tidy") {
		t.Fatalf("schedule %s %v", body, err)
	}
	if !strings.Contains(out.String(), "installed 1 jobs files") {
		t.Fatalf("stdout %s", out.String())
	}
}

func TestHelpAndRejectedArgs(t *testing.T) {
	d := testDesk(t)
	var out, err bytes.Buffer
	d.stdout, d.stderr = &out, &err
	if code := d.run([]string{"help"}); code != 0 || !strings.Contains(out.String(), "in vim") || !strings.Contains(out.String(), "c3po tui looptab") {
		t.Fatalf("help %d %s", code, out.String())
	}
	err.Reset()
	if code := d.run([]string{"register"}); code != 2 || !strings.Contains(err.String(), "in vim") {
		t.Fatalf("register %d %s", code, err.String())
	}
	err.Reset()
	if code := d.run([]string{"nope"}); code != 2 || !strings.Contains(err.String(), "in vim") {
		t.Fatalf("nope %d %s", code, err.String())
	}
}

func TestSchedulerFileIsNotASource(t *testing.T) {
	d := testDesk(t)
	if err := os.WriteFile(d.schedule, []byte("# old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var err bytes.Buffer
	d.stderr = &err
	if code := d.run([]string{"register", d.schedule}); code != 2 || !strings.Contains(err.String(), "scheduler file") {
		t.Fatalf("register schedule %d %s", code, err.String())
	}
}

func TestMissingVim(t *testing.T) {
	d := testDesk(t)
	d.edit = nil
	var err bytes.Buffer
	d.stderr = &err
	if code := d.run(nil); code != 1 || !strings.Contains(err.String(), "vim is not on PATH") {
		t.Fatalf("vim %d %s", code, err.String())
	}
}

func testDesk(t *testing.T) *desk {
	t.Helper()
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	return &desk{
		stdout:   io.Discard,
		stderr:   io.Discard,
		record:   filepath.Join(root, "c3po", "looptab", "files.json"),
		schedule: filepath.Join(t.TempDir(), "looptab"),
		jobs:     filepath.Join(t.TempDir(), "jobs"),
		edit:     func(string) error { return nil },
	}
}
