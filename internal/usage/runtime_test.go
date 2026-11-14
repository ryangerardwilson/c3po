package usage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeIncrementDoesNotWaitOnTheBackup(t *testing.T) {
	state := t.TempDir()
	runtime := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "sqlx.log")
	bin := filepath.Join(t.TempDir(), "sqlx")
	script := "#!/bin/sh\nprintf '%s\\n' \"$SQLX_COMMIT\" >> \"" + logPath + "\"\nexit 0\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtime, "c3po.sqlite"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("SQLX_ENGINE", "postgres")
	t.Setenv("SQLX_RUNTIME", runtime)
	t.Setenv("SQLX_BIN", bin)
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("C3PO_ACTOR", "agent")
	oldNow := Now
	Now = func() time.Time { return time.Date(2026, 11, 14, 1, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { Now = oldNow })
	backed := 0
	oldBackup := wakeBackup
	oldRestore := wakeRestore
	wakeBackup = func() { backed++ }
	wakeRestore = func() { t.Fatal("restore started while the runtime copy exists") }
	t.Cleanup(func() {
		wakeBackup = oldBackup
		wakeRestore = oldRestore
	})

	Note(0, []string{"c3po", "version"})

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(log)) != "local" {
		t.Fatalf("sqlx commit %q", log)
	}
	if backed != 1 {
		t.Fatalf("backup wakes %d", backed)
	}
	if _, err := os.Stat(filepath.Join(state, "c3po", "usage.json")); !os.IsNotExist(err) {
		t.Fatalf("journal %v", err)
	}
}

func TestMissingRuntimeCopyStaysLocal(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("SQLX_ENGINE", "postgres")
	t.Setenv("SQLX_RUNTIME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("C3PO_ACTOR", "human")
	oldNow := Now
	Now = func() time.Time { return time.Date(2026, 11, 14, 1, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { Now = oldNow })
	restored := 0
	oldBackup := wakeBackup
	oldRestore := wakeRestore
	wakeBackup = func() { t.Fatal("backup started without a runtime copy") }
	wakeRestore = func() { restored++ }
	t.Cleanup(func() {
		wakeBackup = oldBackup
		wakeRestore = oldRestore
	})

	Note(0, []string{"c3po", "ast", "hash", "secret-path"})

	if restored != 1 {
		t.Fatalf("restore wakes %d", restored)
	}
	body, err := os.ReadFile(filepath.Join(state, "c3po", "usage.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "ast hash") || strings.Contains(string(body), "secret-path") {
		t.Fatalf("journal %s", body)
	}
	text, err := Report("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "ast hash") || !strings.Contains(text, "2026-11-14") {
		t.Fatalf("report\n%s", text)
	}
}
