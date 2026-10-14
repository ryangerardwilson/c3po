package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNoteReportsNewOverLimit(t *testing.T) {
	dir := productRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeHot(t, filepath.Join(dir, "hot.go"))
	out, code := runHere(t, dir, "note", "--path", "hot.go", "--max-cognitive", "1")
	if code != 0 {
		t.Fatalf("code %d %s", code, out)
	}
	if !bytes.Contains(out, []byte("(new)")) || !bytes.Contains(out, []byte("Get it under")) {
		t.Fatalf("%s", out)
	}
	if bytes.Contains(out, []byte("Summary")) {
		t.Fatalf("note included the rollup\n%s", out)
	}
}

func TestBudgetNamesTheHelper(t *testing.T) {
	dir := productRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeHot(t, filepath.Join(dir, "hot.go"))
	out, code := runHere(t, dir, "budget", "--path", "hot.go", "--max-cognitive", "1")
	if code != 0 {
		t.Fatalf("code %d %s", code, out)
	}
	if !bytes.Contains(out, []byte("a new branch goes in a helper")) {
		t.Fatalf("%s", out)
	}
}

func TestAcceptWritesTheBook(t *testing.T) {
	dir := productRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeHot(t, filepath.Join(dir, "hot.go"))
	out, code := runHere(t, dir, "accept", "hot.go", "hot", "--reason", "kept", "--max-cognitive", "1")
	if code != 0 {
		t.Fatalf("code %d %s", code, out)
	}
	buf, err := os.ReadFile(filepath.Join(dir, ".complexity-accept.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf, []byte("kept")) || !bytes.Contains(buf, []byte("hot")) {
		t.Fatalf("%s", buf)
	}
}

func TestRatchetJSONIncludesTheChange(t *testing.T) {
	dir := productRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeHot(t, filepath.Join(dir, "hot.go"))
	out, code := runHere(t, dir, "check", "--diff", "HEAD", "--ratchet", "--format", "json", "--path", "hot.go", "--max-cognitive", "1")
	if code != 1 {
		t.Fatalf("code %d %s", code, out)
	}
	var doc struct {
		Hotspots []struct {
			Name string `json:"name"`
		} `json:"hotspots"`
		Ratchet struct {
			Changes []struct {
				Status string `json:"status"`
				PassAt int    `json:"pass_at"`
				Action string `json:"action"`
			} `json:"changes"`
		} `json:"ratchet"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Hotspots) == 0 || len(doc.Ratchet.Changes) == 0 {
		t.Fatalf("%s", out)
	}
	got := doc.Ratchet.Changes[0]
	if got.Status != "new" || got.PassAt != 1 || got.Action == "" {
		t.Fatalf("%s", out)
	}
}

func TestInsertedLineDoesNotRenewAnArrow(t *testing.T) {
	dir := productRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	body := "const handle = (a) => {\n  if (a) {\n  } else if (a) {\n  }\n}\n"
	path := filepath.Join(dir, "handle.ts")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, code := runHere(t, dir, "check", "--path", "handle.ts", "--max-cognitive", "1", "--format", "json"); code != 1 {
		t.Fatalf("arrow was not over the limit, code %d", code)
	}
	gitAt(t, dir, "add", "handle.ts")
	gitAt(t, dir, "commit", "-m", "arrow")
	if err := os.WriteFile(path, []byte("\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runHere(t, dir, "check", "--diff", "HEAD", "--ratchet", "--path", "handle.ts", "--max-cognitive", "1", "--format", "json")
	if code != 0 {
		t.Fatalf("line insert looked new\ncode %d %s", code, out)
	}
}

func TestRatchetIgnoresAnUntouchedModel(t *testing.T) {
	dir := productRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sql := "CREATE TABLE a_node (id int primary key, b_id int references b_node(id));\n" +
		"CREATE TABLE b_node (id int primary key, a_id int references a_node(id));\n"
	if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(sql), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "schema.sql")
	gitAt(t, dir, "commit", "-m", "schema")
	if _, code := runHere(t, dir, "check", "--path", "schema.sql", "--format", "json"); code != 1 {
		t.Fatalf("cycle should fail an absolute check, code %d", code)
	}
	ok := filepath.Join(dir, "ok.go")
	buf, err := os.ReadFile(ok)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ok, append(buf, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runHere(t, dir, "check", "--ratchet", "--path", ".", "--format", "json")
	if code != 0 {
		t.Fatalf("untouched model failed ratchet\ncode %d %s", code, out)
	}
}

func writeHot(t *testing.T, path string) {
	t.Helper()
	body := "package p\nfunc hot(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runHere(t *testing.T, dir string, args ...string) ([]byte, int) {
	t.Helper()
	t.Chdir(dir)
	var out, errb bytes.Buffer
	code := Execute(args, &out, &errb)
	if errb.Len() > 0 {
		_, _ = out.Write(errb.Bytes())
	}
	return out.Bytes(), code
}

func productRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitAt(t, dir, "init")
	gitAt(t, dir, "config", "user.email", "t@example.com")
	gitAt(t, dir, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\nfunc ok() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "ok.go")
	gitAt(t, dir, "commit", "-m", "init")
	return dir
}

func gitAt(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
}
