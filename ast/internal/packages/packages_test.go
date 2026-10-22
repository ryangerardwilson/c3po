package packages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAndAffected(t *testing.T) {
	root := t.TempDir()
	mustMk(t, filepath.Join(root, "Apps", "tm"))
	mustWrite(t, filepath.Join(root, "Apps", "tm", "go.mod"), "module tm\n")
	mustWrite(t, filepath.Join(root, "Apps", "tm", "install.sh"), "#!/bin/sh\n")
	mustMk(t, filepath.Join(root, "Apps", "web"))
	mustWrite(t, filepath.Join(root, "Apps", "web", "package.json"), `{"scripts":{"test":"echo ok"}}`+"\n")
	mustMk(t, filepath.Join(root, "Apps", "gmail", "tests"))
	mustWrite(t, filepath.Join(root, "Apps", "gmail", "requirements.txt"), "example\n")
	mustWrite(t, filepath.Join(root, "Apps", "gmail", "install.sh"), "#!/bin/sh\n")
	mustMk(t, filepath.Join(root, "tests", "rgwos-chromium"))
	mustWrite(t, filepath.Join(root, "tests", "rgwos-chromium", "run.sh"), "#!/bin/sh\n")
	mustMk(t, filepath.Join(root, ".config", "hypr"))
	mustMk(t, filepath.Join(root, ".config", "quickshell"))
	mustMk(t, filepath.Join(root, "tests", "quickshell-desktop"))
	mustWrite(t, filepath.Join(root, "tests", "quickshell-desktop", "run.sh"), "#!/bin/sh\n")

	pkgs, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Package{}
	for _, p := range pkgs {
		byID[p.ID] = p
	}
	tm, ok := byID["Apps/tm"]
	if !ok {
		t.Fatalf("missing Apps/tm in %#v", pkgs)
	}
	if tm.Install != "Apps/tm/install.sh from Apps/tm" {
		t.Fatalf("install %q", tm.Install)
	}
	if !contains(tm.Verify, "go test -C Apps/tm ./...") {
		t.Fatalf("verify %#v", tm.Verify)
	}
	web := byID["Apps/web"]
	if !contains(web.Verify, "npm test --prefix Apps/web") {
		t.Fatalf("web verify %#v", web.Verify)
	}
	gmail := byID["Apps/gmail"]
	if !contains(gmail.Verify, "PYTHONPATH=Apps/gmail python3 -m unittest discover -s Apps/gmail/tests -q") {
		t.Fatalf("gmail verify %#v", gmail.Verify)
	}
	if _, ok := byID["tests/rgwos-chromium"]; !ok {
		t.Fatal("missing test package")
	}
	qs := byID[".config/quickshell"]
	if !contains(qs.Verify, "tests/quickshell-desktop/run.sh") {
		t.Fatalf("quickshell verify %#v", qs.Verify)
	}

	got := MapPaths(pkgs, []string{"Apps/tm/foo.go", "README.md"})
	if len(got.Packages) != 1 || got.Packages[0].ID != "Apps/tm" {
		t.Fatalf("affected packages %#v", got.Packages)
	}
	if !contains(got.Unmapped, "README.md") {
		t.Fatalf("unmapped %#v", got.Unmapped)
	}
	empty := MapPaths(pkgs, nil)
	if len(empty.Packages) != 0 {
		t.Fatalf("expected empty %#v", empty.Packages)
	}
}

func mustMk(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func TestUVWorkspaceAndGoModule(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "pyproject.toml"), `[tool.uv.workspace]
members = ["packages/*", "apps/*"]
`)
	mustMk(t, filepath.Join(root, "packages", "promise-engine"))
	mustWrite(t, filepath.Join(root, "packages", "promise-engine", "pyproject.toml"), "[project]\nname = \"promise-engine\"\ndependencies = [\"model\"]\n")
	mustMk(t, filepath.Join(root, "packages", "model"))
	mustWrite(t, filepath.Join(root, "packages", "model", "pyproject.toml"), "[project]\nname = \"model\"\n")
	mustMk(t, filepath.Join(root, "apps", "web"))
	mustWrite(t, filepath.Join(root, "apps", "web", "pyproject.toml"), "[project]\nname = \"web\"\ndependencies = [\"promise-engine\"]\n")
	mustMk(t, filepath.Join(root, "tools", "metrics"))
	mustWrite(t, filepath.Join(root, "tools", "metrics", "go.mod"), "module metrics\n")
	mustWrite(t, filepath.Join(root, "tools", "metrics", "main.go"), "package main\nfunc main() {}\n")
	pkgs, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Package{}
	for _, p := range pkgs {
		byID[p.ID] = p
	}
	if _, ok := byID["packages/promise-engine"]; !ok {
		t.Fatalf("missing python package %#v", pkgs)
	}
	if _, ok := byID["tools/metrics"]; !ok {
		t.Fatalf("missing go module %#v", pkgs)
	}
	mustMk(t, filepath.Join(root, "extra", "tool", ".git"))
	mustWrite(t, filepath.Join(root, "extra", "tool", "go.mod"), "module tool\n")
	mustWrite(t, filepath.Join(root, "go.work"), "go 1.22\n\nuse (\n\t./extra/tool\n)\n")
	pkgs, err = Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	foundWork := false
	for _, p := range pkgs {
		if p.ID == "extra/tool" && p.Kind == "module" {
			foundWork = true
		}
	}
	if !foundWork {
		t.Fatalf("go.work module missing %#v", pkgs)
	}
	got := MapPaths(pkgs, []string{"packages/promise-engine/engine.py"})
	if len(got.Unmapped) != 0 || got.Packages[0].ID != "packages/promise-engine" {
		t.Fatalf("unmapped %#v packages %#v", got.Unmapped, got.Packages)
	}
}
