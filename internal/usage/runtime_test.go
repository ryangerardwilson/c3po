package usage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// fakeSQLX installs a sqlx that logs each SQLX_COMMIT and sleeps delay
// seconds, plus a runtime copy, and returns the log path.
func fakeSQLX(t *testing.T, delay string) string {
	t.Helper()
	runtime := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "sqlx.log")
	bin := filepath.Join(t.TempDir(), "sqlx")
	script := "#!/bin/sh\nprintf '%s\\n' \"$SQLX_COMMIT\" >> \"" + logPath + "\"\nsleep " + delay + "\nexit 0\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	nebula := filepath.Join(t.TempDir(), "nebula")
	put := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + logPath + "\"\nexit 0\n"
	if err := os.WriteFile(nebula, []byte(put), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEBULA_BIN", nebula)
	if err := os.WriteFile(filepath.Join(runtime, "c3po.sqlite"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("SQLX_ENGINE", "postgres")
	t.Setenv("SQLX_RUNTIME", runtime)
	t.Setenv("SQLX_BIN", bin)
	oldIdle, oldPoll := workerIdle, workerPoll
	workerIdle, workerPoll = 300*time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() { workerIdle, workerPoll = oldIdle, oldPoll })
	return logPath
}

func TestRuntimeIncrementOnlyAppendsTheJournal(t *testing.T) {
	state := t.TempDir()
	logPath := fakeSQLX(t, "0")
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

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("the command ran sqlx: %v", err)
	}
	if backed != 1 {
		t.Fatalf("backup wakes %d", backed)
	}
	rows, err := localRows()
	if err != nil || len(rows) != 1 || rows[0].Feature != "version" || rows[0].N != 1 {
		t.Fatalf("journal %#v %v", rows, err)
	}

	FlushBackup()

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(log)
	if strings.Count(text, "local") != 1 || strings.Contains(text, "remote") || !strings.Contains(text, "s3 put") || !strings.Contains(text, "s3://sqlx/c3po.sqlite") {
		t.Fatalf("worker backup %q", log)
	}
	if rows, err := localRows(); err != nil || len(rows) != 0 {
		t.Fatalf("journal after the worker %#v %v", rows, err)
	}
	if script, err := takeQueue(); err != nil || script != "" {
		t.Fatalf("backup queue after the worker %q %v", script, err)
	}
}

func TestOneWorkerDrainsIncrementsThatArriveWhileItWorks(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	logPath := fakeSQLX(t, "0.2")
	t.Setenv("C3PO_ACTOR", "agent")
	oldBackup := wakeBackup
	wakeBackup = func() {}
	t.Cleanup(func() { wakeBackup = oldBackup })

	Note(0, []string{"c3po", "version"})
	done := make(chan struct{})
	go func() {
		FlushBackup()
		close(done)
	}()
	time.Sleep(100 * time.Millisecond) // the worker is inside its first sqlx call
	if lock, ok := tryLock(backupLock); ok {
		_ = lock.Close()
		t.Fatal("the worker does not hold its lock")
	}
	Note(0, []string{"c3po", "help"})
	<-done

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(log), "local"); n != 2 {
		t.Fatalf("runtime writes %d: %q", n, log)
	}
	if rows, err := localRows(); err != nil || len(rows) != 0 {
		t.Fatalf("journal after the worker %#v %v", rows, err)
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

func TestFailedJournalRemovalDoesNotReapply(t *testing.T) {
	logPath := fakeSQLX(t, "0")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("C3PO_ACTOR", "agent")
	oldBackup := wakeBackup
	wakeBackup = func() {}
	oldSub := subtractJournal
	subtractJournal = func([]sqlxdb.UsageRow) error { return os.ErrPermission }
	t.Cleanup(func() {
		wakeBackup = oldBackup
		subtractJournal = oldSub
	})

	Note(0, []string{"c3po", "version"})
	FlushBackup()

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(log), "local"); n != 1 {
		t.Fatalf("runtime writes %d: %q", n, log)
	}
	if rows, err := localRows(); err != nil || len(rows) != 1 || rows[0].N != 1 {
		t.Fatalf("journal %#v %v", rows, err)
	}
}
