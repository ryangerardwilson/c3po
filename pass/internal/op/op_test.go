package op

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const fakeToken = "sa-token-value"

func clearToken(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN_FILE", "")
	return home
}

func TestEnvUsesTheEnvironmentToken(t *testing.T) {
	clearToken(t)
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", fakeToken)
	env, err := Env()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(env, "OP_SERVICE_ACCOUNT_TOKEN="+fakeToken) {
		t.Fatal("token missing from env")
	}
}

func TestEnvReadsTheDefaultTokenFile(t *testing.T) {
	home := clearToken(t)
	path := filepath.Join(home, ".config", "c3po", "pass", "service-account-token")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("  "+fakeToken+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := Env()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range env {
		if strings.HasPrefix(entry, "OP_SERVICE_ACCOUNT_TOKEN=") {
			count++
		}
	}
	if count != 1 || !slices.Contains(env, "OP_SERVICE_ACCOUNT_TOKEN="+fakeToken) {
		t.Fatal("token file was not used once")
	}
}

func TestEnvReadsTheNamedTokenFile(t *testing.T) {
	clearToken(t)
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(fakeToken), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN_FILE", path)
	env, err := Env()
	if err != nil || !slices.Contains(env, "OP_SERVICE_ACCOUNT_TOKEN="+fakeToken) {
		t.Fatal("named token file was not used")
	}
}

func TestEnvReportsAMissingOrEmptyToken(t *testing.T) {
	clearToken(t)
	if _, err := Env(); err == nil || err.Error() != "1Password service account token is missing." {
		t.Fatalf("err = %v", err)
	}
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN_FILE", path)
	if _, err := Env(); err == nil {
		t.Fatal("empty token file succeeded")
	}
}

// fakeBinary puts a fake op script first on PATH. It records its args
// and stdin, and prints stdout only when the token matches.
func fakeBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${OP_SERVICE_ACCOUNT_TOKEN:-}" != "sa-token-value" ]]; then
  printf 'token rejected\n' >&2
  exit 1
fi
printf '%s\n' "$*" > "$FAKE_OP_DIR/args"
cat > "$FAKE_OP_DIR/stdin"
printf '[{"id":"one","title":"T","category":"LOGIN","vault":{"id":"v"}}]'
`
	if err := os.WriteFile(filepath.Join(dir, "op"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	t.Setenv("FAKE_OP_DIR", dir)
	return dir
}

func TestExecRunsOpWithEnvAndStdin(t *testing.T) {
	dir := fakeBinary(t)
	if err := (Exec{}).LookPath(); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	client := Client{Runner: Exec{}, Env: append(os.Environ(), "OP_SERVICE_ACCOUNT_TOKEN="+fakeToken), Err: &stderr}
	if err := client.Do([]byte("body"), "item", "create", "--vault", "Root", "-"); err != nil {
		t.Fatalf("err = %v stderr = %s", err, stderr.String())
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args"))
	stdin, _ := os.ReadFile(filepath.Join(dir, "stdin"))
	if string(args) != "item create --vault Root -\n" || string(stdin) != "body" {
		t.Fatalf("args = %q stdin = %q", args, stdin)
	}
	items, err := client.List("Root")
	if err != nil || len(items) != 1 || items[0].ID != "one" || items[0].Vault.ID != "v" {
		t.Fatalf("items = %#v err = %v", items, err)
	}
}

func TestExecReportsARejectedToken(t *testing.T) {
	fakeBinary(t)
	var stderr bytes.Buffer
	client := Client{Runner: Exec{}, Env: []string{"PATH=" + os.Getenv("PATH")}, Err: &stderr}
	_, err := client.List("Root")
	if err == nil || err.Error() != "Unable to list 1Password items in vault Root." {
		t.Fatalf("err = %v", err)
	}
	if stderr.String() != "token rejected\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecLookPathFailsWithoutOp(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := (Exec{}).LookPath(); err == nil {
		t.Fatal("op was found on an empty PATH")
	}
}
