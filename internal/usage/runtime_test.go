package usage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeSQLX installs a nebula that logs each keep mark and sleeps delay seconds.
// The runtime file is a real sqlite database, because the app writes it.
func fakeSQLX(t *testing.T, delay string) string {
	t.Helper()
	runtime := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "sqlx.log")
	db := filepath.Join(runtime, "c3po.sqlite")
	schema := `CREATE TABLE usage (
  id INTEGER PRIMARY KEY,
  feature TEXT NOT NULL,
  actor TEXT NOT NULL,
  day TEXT NOT NULL,
  n INTEGER NOT NULL,
  ok INTEGER NOT NULL,
  last INTEGER NOT NULL
);
CREATE UNIQUE INDEX usage_feature_actor_day ON usage(feature, actor, day);`
	if out, err := exec.Command("/usr/bin/sqlite3", db, schema).CombinedOutput(); err != nil {
		t.Fatalf("schema: %v %s", err, out)
	}
	nebula := filepath.Join(t.TempDir(), "nebula")
	put := "#!/bin/sh\nprintf '%s\n' \"$*\" >> \"" + logPath + "\"\nsleep " + delay + "\nexit 0\n"
	if err := os.WriteFile(nebula, []byte(put), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEBULA_BIN", nebula)
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("SQLX_ENGINE", "postgres")
	t.Setenv("SQLX_RUNTIME", runtime)
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
	if strings.Count(text, "keep mark c3po ") != 1 || strings.Contains(text, "remote") || strings.Contains(text, "s3 put") {
		t.Fatalf("worker backup %q", log)
	}
	if rows, err := localRows(); err != nil || len(rows) != 0 {
		t.Fatalf("journal after the worker %#v %v", rows, err)
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
	time.Sleep(100 * time.Millisecond) // the worker is inside its first backup
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
	if n := strings.Count(string(log), "keep mark c3po "); n != 2 {
		t.Fatalf("backups %d: %q", n, log)
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

func TestABatchCommittedByADeadWorkerIsNotAppliedAgain(t *testing.T) {
	fakeSQLX(t, "0")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("C3PO_ACTOR", "agent")
	oldBackup := wakeBackup
	wakeBackup = func() {}
	oldFinish := finishBatch
	finishBatch = func(string) error { return os.ErrPermission }
	t.Cleanup(func() {
		wakeBackup = oldBackup
		finishBatch = oldFinish
	})

	Note(0, []string{"c3po", "version"})
	FlushBackup() // commits the batch, then dies before it drops it
	finishBatch = oldFinish
	Note(0, []string{"c3po", "help"})
	FlushBackup()

	out, err := exec.Command("/usr/bin/sqlite3", filepath.Join(os.Getenv("SQLX_RUNTIME"), "c3po.sqlite"), "select feature || '=' || n from usage order by feature").CombinedOutput()
	if err != nil {
		t.Fatalf("usage rows: %v %s", err, out)
	}
	if got := strings.Fields(string(out)); len(got) != 2 || got[0] != "help=1" || got[1] != "version=1" {
		t.Fatalf("usage rows %q", got)
	}
	path, _ := journalPath()
	if left, _ := filepath.Glob(path + ".batch.*"); len(left) != 0 {
		t.Fatalf("batch left: %q", left)
	}
	if rows, err := localRows(); err != nil || len(rows) != 0 {
		t.Fatalf("journal %#v %v", rows, err)
	}
}
