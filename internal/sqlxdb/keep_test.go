package sqlxdb

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stubKeep replaces nebula keep. restore makes the empty file keep makes
// when there is no backup.
func stubKeep(t *testing.T, restoreErr error) (marks *[]string, restores *int) {
	t.Helper()
	marks = &[]string{}
	restores = new(int)
	oldMark, oldRestore := keepMark, keepRestore
	keepMark = func(path, name string) error {
		*marks = append(*marks, name+" "+path)
		return nil
	}
	keepRestore = func(path, name string) error {
		*restores++
		if restoreErr != nil {
			_ = os.WriteFile(path, []byte("partial"), 0o600)
			return restoreErr
		}
		return os.WriteFile(path, nil, 0o600)
	}
	t.Cleanup(func() { keepMark, keepRestore = oldMark, oldRestore })
	return marks, restores
}

func runtimeEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("SQLX_RUNTIME", dir)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	return filepath.Join(dir, DatabaseName+".sqlite")
}

func TestRestoreFailureLeavesTheFileMissing(t *testing.T) {
	path := runtimeEnv(t)
	_, restores := stubKeep(t, errors.New("nebula keep restore c3po: connection timed out"))
	if err := prepareAppDB(path, DatabaseName); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err %v", err)
	}
	if *restores != 1 {
		t.Fatalf("restores %d", *restores)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file remained: %v", err)
	}
}

func TestAFreshDatabaseFromKeepIsMigrated(t *testing.T) {
	path := runtimeEnv(t)
	stubKeep(t, nil)
	if err := prepareAppDB(path, DatabaseName); err != nil {
		t.Fatal(err)
	}
	if ok, err := tableExists(path, "tui_session"); err != nil || !ok {
		t.Fatalf("schema missing: %v %v", ok, err)
	}
}

func TestEveryWriteMarksTheFileAndNoneUploads(t *testing.T) {
	path := runtimeEnv(t)
	marks, _ := stubKeep(t, nil)
	session := "INSERT INTO tui_session(id, screen) VALUES ('01a17622-5fd9-7bc2-81ff-3e200a8c16b7', 'slack')"
	if err := Exec(session); err != nil {
		t.Fatal(err)
	}
	if err := ExecBackup(); err != nil {
		t.Fatal(err)
	}
	want := "c3po " + path
	if len(*marks) != 2 || (*marks)[0] != want || (*marks)[1] != want {
		t.Fatalf("marks %q", *marks)
	}
}

func TestATokenWriteMarksTheFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	marks, _ := stubKeep(t, nil)
	token := filepath.Join(t.TempDir(), "c3po", "gmail", "tokens", "user.json")
	if err := WriteToken("gmail", token, []byte(`{"refresh_token":"x"}`)); err != nil {
		t.Fatal(err)
	}
	if len(*marks) != 1 || (*marks)[0] != "c3po "+filepath.Join(root, "data", "c3po.sqlite") {
		t.Fatalf("marks %q", *marks)
	}
}

func TestAFailedMarkKeepsTheCommit(t *testing.T) {
	runtimeEnv(t)
	stubKeep(t, nil)
	old := keepMark
	keepMark = func(string, string) error { return errors.New("nebula is not on PATH") }
	t.Cleanup(func() { keepMark = old })
	if err := Exec("INSERT INTO tui_session(id, screen) VALUES ('01a17622-5fd9-7bc2-81ff-3e200a8c16b8', 'o')"); err != nil {
		t.Fatalf("the write failed after it committed: %v", err)
	}
	var rows []struct {
		N int `json:"n"`
	}
	if err := Query("SELECT COUNT(*) AS n FROM tui_session", &rows); err != nil || len(rows) != 1 || rows[0].N != 1 {
		t.Fatalf("%v %v", rows, err)
	}
}

func TestMarkChangedLeavesAMissingFileMissing(t *testing.T) {
	path := runtimeEnv(t)
	marks, restores := stubKeep(t, nil)
	if err := MarkChanged(); err != nil {
		t.Fatal(err)
	}
	if len(*marks) != 0 || *restores != 0 {
		t.Fatalf("marks %q restores %d", *marks, *restores)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("created: %v", err)
	}
}

func TestTheSchemaIsCheckedOncePerProcess(t *testing.T) {
	path := runtimeEnv(t)
	stubKeep(t, nil)
	logPath := filepath.Join(t.TempDir(), "sqlite.log")
	wrapper := filepath.Join(t.TempDir(), "sqlite3")
	script := "#!/bin/sh\necho run >> " + logPath + "\nexec /usr/bin/sqlite3 \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SQLX_SQLITE", wrapper)
	migrated.Delete(path)
	if err := Exec("DELETE FROM tui_active"); err != nil {
		t.Fatal(err)
	}
	first := strings.Count(readText(t, logPath), "run")
	if err := Exec("DELETE FROM tui_active"); err != nil {
		t.Fatal(err)
	}
	second := strings.Count(readText(t, logPath), "run") - first
	if second != 1 {
		t.Fatalf("a write after the first ran sqlite3 %d times (first %d)", second, first)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestKeepRunsNebula(t *testing.T) {
	path := runtimeEnv(t)
	logPath := filepath.Join(t.TempDir(), "nebula.log")
	fake := filepath.Join(t.TempDir(), "nebula")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + logPath + "\n"
	if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEBULA_BIN", fake)
	if err := keepMark(path, DatabaseName); err != nil {
		t.Fatal(err)
	}
	if err := keepRestore(path, DatabaseName); err != nil {
		t.Fatal(err)
	}
	if got := readText(t, logPath); got != "keep mark c3po "+path+"\nkeep restore c3po "+path+"\n" {
		t.Fatalf("nebula ran %q", got)
	}
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("SQLX_ENGINE", "")
	if err := keepMark(path, DatabaseName); err != nil {
		t.Fatal(err)
	}
	if got := readText(t, logPath); strings.Count(got, "\n") != 2 {
		t.Fatalf("SQLX_ROOT reached nebula: %q", got)
	}
}

func TestInstallerRetiresTheSnapshotTimer(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "disable --now c3po-snapshot.timer") || !strings.Contains(text, "c3po-snapshot.service") {
		t.Fatal("install does not retire c3po-snapshot.timer")
	}
	if strings.Contains(text, "OnUnitActiveSec=10min") || strings.Contains(text, "enable c3po-snapshot.timer") {
		t.Fatal("install still arms the snapshot timer")
	}
}
