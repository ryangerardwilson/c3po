package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPreToolAllowsNewHotspot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(dir, "complexity.toml"), []byte("[thresholds]\ncognitive = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "a.go")
	body := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	ev := map[string]any{
		"hook_event_name": "PreToolUse",
		"tool_name":       "write",
		"cwd":             dir,
		"tool_input":      map[string]any{"file_path": path, "content": body},
	}
	out := runEvent(t, ev)
	if out["decision"] != "allow" {
		t.Fatalf("%v", out)
	}
}

func TestPreToolAllowsLegacy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(dir, "a.go")
	body := "package p\nfunc hot(a, b, c, d, e, f, g bool) { _ = a; _ = b; _ = c; _ = d; _ = e; _ = f; _ = g }\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{
		"hook_event_name": "PreToolUse",
		"tool_name":       "search_replace",
		"cwd":             dir,
		"tool_input": map[string]any{
			"file_path": path, "old_string": "_ = g", "new_string": "_ = g // touched",
		},
	}
	out := runEvent(t, ev)
	if out["decision"] != "allow" {
		t.Fatalf("%v", out)
	}
}

func TestStopSkipsShutdown(t *testing.T) {
	dir := gitRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ev := map[string]any{"hook_event_name": "Stop", "cwd": dir, "reason": "channel_closed"}
	out := runEvent(t, ev)
	if out["decision"] == "block" {
		t.Fatalf("%v", out)
	}
}

func TestStopBlocksNewFile(t *testing.T) {
	dir := gitRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(dir, "complexity.toml"), []byte("[thresholds]\ncognitive = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "new.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runEvent(t, map[string]any{"hook_event_name": "Stop", "cwd": dir})
	if out["decision"] != "block" {
		t.Fatalf("%v", out)
	}
}

func runEvent(t *testing.T, ev map[string]any) map[string]string {
	t.Helper()
	buf, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run(bytes.NewReader(buf), &out); code != 0 {
		t.Fatalf("code %d", code)
	}
	var doc map[string]string
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("%v %s", err, out.String())
	}
	return doc
}

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "t@example.com")
	run(t, dir, "git", "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\nfunc ok() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "ok.go")
	run(t, dir, "git", "commit", "-m", "init")
	return dir
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %s", name, err, out)
	}
}

func TestPreToolAllowsWrite(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(pkg, "complexity.toml"), []byte("[thresholds]\ncognitive = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(pkg, "a.go")
	body := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	ev := map[string]any{
		"hook_event_name": "PreToolUse",
		"tool_name":       "write",
		"cwd":             pkg,
		"tool_input":      map[string]any{"file_path": path, "content": body},
	}
	out := runEvent(t, ev)
	if out["decision"] != "allow" {
		t.Fatalf("%v", out)
	}
}

func TestStopReasonRelativeToEventCwd(t *testing.T) {
	root := gitRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "complexity.toml"), []byte("[thresholds]\ncognitive = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(pkg, "new.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runEvent(t, map[string]any{"hook_event_name": "Stop", "cwd": pkg})
	if out["decision"] != "block" {
		t.Fatalf("%v", out)
	}
	reason := []byte(out["reason"])
	if !bytes.Contains(reason, []byte("new.go")) || bytes.Contains(reason, []byte("pkg/new.go")) || bytes.Contains(reason, []byte(pkg)) {
		t.Fatalf("reason %s", out["reason"])
	}
}

func TestStopHookActiveAllows(t *testing.T) {
	dir := gitRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(dir, "complexity.toml"), []byte("[thresholds]\ncognitive = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "new.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runEvent(t, map[string]any{"hook_event_name": "Stop", "cwd": dir, "stop_hook_active": true})
	if out["decision"] != "allow" {
		t.Fatalf("%v", out)
	}
}

func TestStopSessionSurvivesCommit(t *testing.T) {
	dir := gitRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(dir, "complexity.toml"), []byte("[thresholds]\ncognitive = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	path := filepath.Join(dir, "new.go")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	runEvent(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      "sid-1",
		"cwd":             dir,
		"tool_name":       "write",
		"tool_input":      map[string]any{"file_path": path, "content": body},
	})
	run(t, dir, "git", "add", "new.go")
	run(t, dir, "git", "commit", "-m", "add hot")
	out := runEvent(t, map[string]any{"hook_event_name": "Stop", "cwd": dir, "session_id": "sid-1"})
	if out["decision"] != "block" {
		t.Fatalf("session %v", out)
	}
	if !bytes.Contains([]byte(out["reason"]), []byte("new.go")) {
		t.Fatalf("reason %s", out["reason"])
	}
	out = runEvent(t, map[string]any{"hook_event_name": "Stop", "cwd": dir})
	if out["decision"] == "block" {
		t.Fatalf("HEAD allowed the commit to clear it: %v", out)
	}
}
