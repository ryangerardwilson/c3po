package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/files"
)

func TestHelpVersionStatus(t *testing.T) {
	ws := t.TempDir()
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	cfg := config.Default()
	cfg.ThresholdLOC = 100
	cfg.Enforcement.Mode = "auto"
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(ws, "a.go"), strings.Repeat("// line\n", 50))

	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.Cwd = ws
	r.ConfigPath = cfgPath

	if code := r.Run(nil); code != 0 {
		t.Fatal(code)
	}
	out.Reset()
	if code := r.Run([]string{"version"}); code != 0 || !strings.Contains(out.String(), "0.3.0") {
		t.Fatalf("version %q code %d", out.String(), code)
	}
	out.Reset()
	if code := r.Run([]string{"status", "--json"}); code != 0 {
		t.Fatalf("status %d %s", code, errB.String())
	}
	var st statusOut
	if err := json.Unmarshal(out.Bytes(), &st); err != nil {
		t.Fatal(err, out.String())
	}
	if st.Enforced {
		t.Fatal("should not enforce below threshold")
	}
	if st.LOC != 50 {
		t.Fatalf("loc %d", st.LOC)
	}

	// push over threshold
	mustWrite(t, filepath.Join(ws, "b.go"), strings.Repeat("// line\n", 60))
	out.Reset()
	errB.Reset()
	if code := r.Run([]string{"status", "--json"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	if err := json.Unmarshal(out.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if !st.Enforced {
		t.Fatal("expected enforced")
	}

	// whole-file read denied for source
	out.Reset()
	errB.Reset()
	code := r.Run([]string{"read", "a.go"})
	if code != ExitFail {
		t.Fatalf("read code %d out=%q err=%q", code, out.String(), errB.String())
	}
	// whole-file read denied for json too when enforced
	mustWrite(t, filepath.Join(ws, "package.json"), `{"name":"x"}`+"\n")
	out.Reset()
	errB.Reset()
	code = r.Run([]string{"read", "package.json"})
	if code != ExitFail {
		t.Fatalf("json read code %d out=%q err=%q", code, out.String(), errB.String())
	}

	// bounded read ok (includes header line)
	out.Reset()
	errB.Reset()
	if code := r.Run([]string{"read", "a.go", "--lines", "1-3"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	if !strings.Contains(out.String(), "lines 1-3") {
		t.Fatalf("missing header %q", out.String())
	}
}

func TestPatchFlow(t *testing.T) {
	ws := t.TempDir()
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	cfg := config.Default()
	cfg.Enforcement.Mode = "never"
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(ws, "f.go"), "package p\nconst X = 1\n")

	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.Cwd = ws
	r.ConfigPath = cfgPath

	if code := r.Run([]string{"hash", "f.go"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	parts := strings.Fields(out.String())
	if len(parts) < 1 {
		t.Fatal(out.String())
	}
	hash := parts[0]
	out.Reset()
	if code := r.Run([]string{"patch", "f.go", "--expect-hash", hash, "--old", "X = 1", "--new", "X = 2"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	data, _ := os.ReadFile(filepath.Join(ws, "f.go"))
	if !strings.Contains(string(data), "X = 2") {
		t.Fatalf("%s", data)
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	if code := r.Run([]string{"nope"}); code != ExitUsage {
		t.Fatal(code)
	}
}

func TestRootFlagAndHook(t *testing.T) {
	ws := t.TempDir()
	mustWrite(t, filepath.Join(ws, "x.go"), "package x\n")
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	cfg := config.Default()
	cfg.ThresholdLOC = 1
	cfg.Enforcement.Mode = "auto"
	cfg.Cache.Dir = t.TempDir()
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.ConfigPath = cfgPath
	r.Cwd = t.TempDir() // different cwd; --root points at ws
	if code := r.Run([]string{"--root", ws, "status", "--json"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	if !strings.Contains(out.String(), `"enforced":true`) {
		t.Fatal(out.String())
	}
	out.Reset()
	errB.Reset()
	r.In = strings.NewReader(`{"tool_name":"Edit","tool_input":{}}`)
	if code := r.Run([]string{"--root", ws, "hook"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	if !strings.Contains(out.String(), `"permissionDecision":"deny"`) {
		t.Fatal(out.String())
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFlow(t *testing.T) {
	ws := t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	cfg := config.Default()
	cfg.Enforcement.Mode = "never"
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ws, "old", "nested", "f.txt")
	mustWrite(t, path, "remove me\n")
	h, err := files.HashFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.Cwd = t.TempDir()
	r.ConfigPath = cfgPath
	if code := r.Run([]string{"--root", ws, "delete", "old/nested/f.txt", "--expect-hash", h, "--prune-empty"}); code != ExitOK {
		t.Fatalf("delete code %d out=%q err=%q", code, out.String(), errB.String())
	}
	if !strings.Contains(out.String(), "ok  deleted  old/nested/f.txt") || !strings.Contains(out.String(), "ok  pruned  old") {
		t.Fatalf("unexpected output %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(ws, "old")); !os.IsNotExist(err) {
		t.Fatalf("pruning failed: %v", err)
	}
	if _, err := os.Stat(ws); err != nil {
		t.Fatalf("workspace root removed: %v", err)
	}

	path = filepath.Join(ws, "keep.txt")
	mustWrite(t, path, "keep me\n")
	out.Reset()
	errB.Reset()
	if code := r.Run([]string{"--root", ws, "delete", "keep.txt"}); code != ExitUsage {
		t.Fatalf("missing-hash code %d err=%q", code, errB.String())
	}
	out.Reset()
	errB.Reset()
	if code := r.Run([]string{"--root", ws, "delete", "keep.txt", "--expect-hash", "deadbeef"}); code != ExitFail {
		t.Fatalf("stale-hash code %d err=%q", code, errB.String())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stale hash removed file: %v", err)
	}
}
