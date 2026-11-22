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

func TestNoArgumentsPrintTheHelp(t *testing.T) {
	d := testDesk(t)
	var out bytes.Buffer
	d.stdout = &out
	if code := d.run(nil); code != 0 || !strings.Contains(out.String(), "r2d2 looptab") || strings.Contains(out.String(), "vim") {
		t.Fatalf("no args %d %s", code, out.String())
	}
	if _, err := os.Stat(d.jobs); !os.IsNotExist(err) {
		t.Fatalf("jobs file %v", err)
	}
	if _, err := os.Stat(d.schedule); !os.IsNotExist(err) {
		t.Fatalf("schedule %v", err)
	}
}

func TestEditIsNotACommand(t *testing.T) {
	d := testDesk(t)
	var err bytes.Buffer
	d.stderr = &err
	if code := d.run([]string{"edit"}); code != 2 || !strings.Contains(err.String(), "c3po looptab list") {
		t.Fatalf("edit %d %s", code, err.String())
	}
	if _, statErr := os.Stat(d.jobs); !os.IsNotExist(statErr) {
		t.Fatalf("jobs file %v", statErr)
	}
}

func TestJobsFileRegistersTheTemplateAndInstallWritesIt(t *testing.T) {
	d := testDesk(t)
	path, err := d.first()
	if err != nil || path != d.jobs {
		t.Fatalf("first %q %v", path, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(body), "# Schedules:") {
		t.Fatalf("template %s %v", body, err)
	}
	var out bytes.Buffer
	d.stdout = &out
	if code := d.run([]string{"list"}); code != 0 || !strings.Contains(out.String(), "jobs\t"+d.jobs) {
		t.Fatalf("list %d %s", code, out.String())
	}
	if err := os.WriteFile(path, []byte("hourly c3po gmail tidy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := d.first()
	if err != nil || again != path {
		t.Fatalf("again %q %v", again, err)
	}
	n, err := d.installAll()
	if err != nil || n != 1 {
		t.Fatalf("install %d %v", n, err)
	}
	sched, err := os.ReadFile(d.schedule)
	if err != nil || !strings.Contains(string(sched), "hourly c3po gmail tidy") || !strings.Contains(string(sched), "Installed by c3po looptab") {
		t.Fatalf("schedule %s %v", sched, err)
	}
}

func TestJobsFileOpensTheFirstRegisteredFile(t *testing.T) {
	d := testDesk(t)
	src := filepath.Join(t.TempDir(), "gmail")
	if err := os.WriteFile(src, []byte("daily 5am c3po gmail tidy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := d.run([]string{"register", src}); code != 0 {
		t.Fatalf("register %d", code)
	}
	path, err := d.first()
	if err != nil || path != src {
		t.Fatalf("first %q %v", path, err)
	}
	if _, err := os.Stat(d.jobs); !os.IsNotExist(err) {
		t.Fatalf("template %v", err)
	}
}

func TestHelpAndRejectedArgs(t *testing.T) {
	d := testDesk(t)
	var out, err bytes.Buffer
	d.stdout, d.stderr = &out, &err
	if code := d.run([]string{"help"}); code != 0 || strings.Contains(out.String(), "vim") || !strings.Contains(out.String(), "edited with r2d2 looptab") {
		t.Fatalf("help %d %s", code, out.String())
	}
	err.Reset()
	if code := d.run([]string{"register"}); code != 2 || !strings.Contains(err.String(), "c3po looptab list") {
		t.Fatalf("register %d %s", code, err.String())
	}
	err.Reset()
	if code := d.run([]string{"nope"}); code != 2 || !strings.Contains(err.String(), "c3po looptab list") {
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

func testDesk(t *testing.T) *desk {
	t.Helper()
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	return &desk{
		stdout: io.Discard,
		stderr: io.Discard,
		paths: paths{
			record:   filepath.Join(root, "c3po", "looptab", "files.json"),
			schedule: filepath.Join(t.TempDir(), "looptab"),
			jobs:     filepath.Join(t.TempDir(), "jobs"),
		},
	}
}
