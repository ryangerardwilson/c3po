package xdg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppJoinsTheC3PODirectory(t *testing.T) {
	if got := App("/cfg", "ast"); got != "/cfg/c3po/ast" {
		t.Fatalf("App = %s", got)
	}
}

func TestConfigBaseKeepsAForeignHome(t *testing.T) {
	t.Setenv("HOME", "/real/home")
	t.Setenv("XDG_CONFIG_HOME", "/real/xdg")
	if got := ConfigBase("/tmp/work"); got != "/tmp/work/.config" {
		t.Fatalf("foreign home base = %s", got)
	}
	if got := ConfigBase("/real/home"); got != "/real/xdg" {
		t.Fatalf("process home base = %s", got)
	}
	if got := ConfigBase(""); got != "/real/xdg" {
		t.Fatalf("empty home base = %s", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := ConfigBase("/real/home"); got != "/real/home/.config" {
		t.Fatalf("default base = %s", got)
	}
}

func TestMoveRenamesWhenDestIsMissing(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "gmail")
	if err := os.Mkdir(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := App(root, "gmail")
	if err := move(dest, legacy); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dest, "config.json"))
	if err != nil || string(body) != "ok" {
		t.Fatalf("moved config: %v %q", err, body)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("legacy directory still present")
	}
}

func TestMoveRenamesAFile(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "op", "service-account-token")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(App(root, "pass"), "service-account-token")
	if err := move(dest, legacy); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil || string(body) != "kept" {
		t.Fatalf("moved file: %v %q", err, body)
	}
}

func TestMoveKeepsAnExistingDest(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "op", "service-account-token")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(App(root, "pass"), "service-account-token")
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := move(dest, legacy); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil || string(body) != "new" {
		t.Fatalf("dest = %q", body)
	}
	old, err := os.ReadFile(legacy)
	if err != nil || string(old) != "old" {
		t.Fatal("legacy file was removed")
	}
}

func TestMoveIgnoresAMissingLegacy(t *testing.T) {
	root := t.TempDir()
	if err := move(App(root, "ast"), filepath.Join(root, "rgw-ast")); err != nil {
		t.Fatal(err)
	}
}

func TestAdoptSkipsDuringTests(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "ast")
	if err := os.Mkdir(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	dest := App(root, "ast")
	if err := Adopt(dest, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatal("adopt moved a directory during tests")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("adopt created a destination during tests")
	}
}

func TestMigrateNoopsWhileTesting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	legacy := filepath.Join(home, "config", "rgw-ast")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.toml"), []byte("policy"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "config.toml")); err != nil {
		t.Fatal("migrate ran during tests")
	}
}
