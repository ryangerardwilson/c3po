package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckExitCodes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	hot := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "hot.go"), []byte(hot), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Execute([]string{"check", "--path", dir, "--format", "json", "--max-cognitive", "1"}, &out, &errb)
	if code != 1 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
	var doc struct {
		ViolationsFound bool `json:"violations_found"`
		Hotspots        []struct {
			Name string `json:"name"`
		} `json:"hotspots"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if !doc.ViolationsFound || len(doc.Hotspots) == 0 || doc.Hotspots[0].Name != "hot" {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	clean := "package p\nfunc ok() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "hot.go")); err != nil {
		t.Fatal(err)
	}
	code = Execute([]string{"check", "--path", dir, "--format", "json", "--max-cognitive", "1"}, &out, &errb)
	if code != 0 {
		t.Fatalf("clean code %d %s", code, out.String())
	}
}

func TestFlagOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := "package p\nfunc f(a, b bool) {\n\tif a {\n\t} else if b {\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := "thresholds.cognitive = 15\n"
	if err := os.WriteFile(filepath.Join(dir, "complexity.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Execute([]string{"check", "--config", filepath.Join(dir, "complexity.toml"), "--path", dir, "--max-cognitive", "1", "--format", "json"}, &out, &errb)
	if code != 1 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
}

func TestMissingPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var errb bytes.Buffer
	code := Execute([]string{"check", "--path", filepath.Join(t.TempDir(), "missing")}, &bytes.Buffer{}, &errb)
	if code != 2 {
		t.Fatalf("code %d", code)
	}
	errb.Reset()
	code = Execute([]string{"check", "--path", "missing-rel"}, &bytes.Buffer{}, &errb)
	if code != 2 {
		t.Fatalf("rel code %d %s", code, errb.String())
	}
	if !bytes.Contains(errb.Bytes(), []byte("missing-rel")) || bytes.Contains(errb.Bytes(), []byte("/missing-rel")) {
		t.Fatalf("err %s", errb.String())
	}
}

func TestAbsoluteMissingPathIsRelative(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	t.Chdir(dir)
	var errb bytes.Buffer
	code := Execute([]string{"check", "--path", filepath.Join(dir, "nope")}, &bytes.Buffer{}, &errb)
	if code != 2 {
		t.Fatalf("code %d %s", code, errb.String())
	}
	if !bytes.Contains(errb.Bytes(), []byte("nope")) || bytes.Contains(errb.Bytes(), []byte(dir)) {
		t.Fatalf("err %s", errb.String())
	}
}

func TestSummaryPathRelativeToCwd(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package p\nfunc hot() {}\n"
	if err := os.WriteFile(filepath.Join(pkg, "hot.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(pkg)
	var out, errb bytes.Buffer
	code := Execute([]string{"rank", "--format", "text", "--path", "."}, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("hot.go:")) || bytes.Contains(out.Bytes(), []byte("pkg/hot.go")) {
		t.Fatalf("%s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Rank by cognitive")) || bytes.Contains(out.Bytes(), []byte("Hotspots\n")) {
		t.Fatalf("%s", out.String())
	}
}

func TestStatusRootRelativeToCwd(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(pkg)
	var out, errb bytes.Buffer
	code := Execute([]string{"status"}, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
	var doc struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Root != ".." || strings.HasPrefix(doc.Root, "/") {
		t.Fatalf("root %q %s", doc.Root, out.String())
	}
}

func TestConfigInitPrintsRelativePath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	var out, errb bytes.Buffer
	code := Execute([]string{"config", "--init"}, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
	text := strings.TrimSpace(out.String())
	if !strings.Contains(text, "settings.json") || strings.HasPrefix(text, "/") {
		t.Fatalf("out %s", out.String())
	}
}

func TestCleanCheckDoesNotListPassingFunctions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := "package p\nfunc calm() {}\nfunc hot(a, b bool) {\n\tif a {\n\t\tif b {\n\t\t}\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Execute([]string{"check", "--path", dir, "--format", "text", "--max-cognitive", "1"}, &out, &errb)
	if code != 1 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("\thot\t")) || bytes.Contains(out.Bytes(), []byte("calm")) {
		t.Fatalf("%s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Hotspots\n")) {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	code = Execute([]string{"check", "--path", dir, "--format", "text"}, &out, &errb)
	if code != 0 {
		t.Fatalf("clean code %d %s", code, out.String())
	}
	if bytes.Contains(out.Bytes(), []byte("calm")) || bytes.Contains(out.Bytes(), []byte("\thot\t")) || bytes.Contains(out.Bytes(), []byte("Hotspots\n")) {
		t.Fatalf("%s", out.String())
	}
}

func TestSummaryNamesOnlyRealHotspot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := "package p\nfunc calm() {}\nfunc hot(a, b bool) {\n\tif a {\n\t\tif b {\n\t\t}\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Execute([]string{"summary", "--path", dir, "--format", "text", "--max-cognitive", "1"}, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("\thot\t")) || bytes.Contains(out.Bytes(), []byte("calm")) {
		t.Fatalf("%s", out.String())
	}
}

func TestRankOrdersAndExitsZero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := "package p\nfunc small(a bool) {\n\tif a {\n\t}\n}\nfunc large(a, b bool) {\n\tif a {\n\t}\n\tif b {\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Execute([]string{"rank", "cyclomatic", "1", "--path", dir, "--format", "text", "--max-cognitive", "1"}, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d %s %s", code, out.String(), errb.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("\tlarge\t")) || bytes.Contains(out.Bytes(), []byte("small")) {
		t.Fatalf("%s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Rank by cyclomatic")) || bytes.Contains(out.Bytes(), []byte("Hotspots\n")) {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	code = Execute([]string{"top", "1", "--path", dir, "--format", "text"}, &out, &errb)
	if code != 0 || !bytes.Contains(out.Bytes(), []byte("Rank by cognitive")) || !bytes.Contains(out.Bytes(), []byte("\tlarge\t")) {
		t.Fatalf("top %d %s", code, out.String())
	}
	out.Reset()
	code = Execute([]string{"rank", "cognitive", "--path", dir, "--format", "json"}, &out, &errb)
	if code != 0 {
		t.Fatalf("json %d %s", code, out.String())
	}
	var doc struct {
		ViolationsFound bool `json:"violations_found"`
		Ranked          []struct {
			Name string `json:"name"`
		} `json:"ranked"`
		Hotspots []struct {
			Name string `json:"name"`
		} `json:"hotspots"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.ViolationsFound || len(doc.Hotspots) != 0 || len(doc.Ranked) != 2 || doc.Ranked[0].Name != "large" {
		t.Fatalf("%s", out.String())
	}
}
