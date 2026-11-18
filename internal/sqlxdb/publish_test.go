package sqlxdb

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRestoreFailureLeavesTheFileMissing(t *testing.T) {
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	old := runS3
	runS3 = func(string, []string) (string, error) {
		if err := os.WriteFile(path, []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
		return "connection timed out", errors.New("exit status 1")
	}
	t.Cleanup(func() { runS3 = old })
	err := prepareAppDB(path, "c3po", ormText())
	if err == nil {
		t.Fatal("expected restore error")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("file remained: %v", statErr)
	}
}

func TestRestoreNotFoundCreatesADatabase(t *testing.T) {
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	old := runS3
	runS3 = func(string, []string) (string, error) {
		return "s3://backups/c3po.sqlite not found", errors.New("exit status 1")
	}
	t.Cleanup(func() { runS3 = old })
	if err := prepareAppDB(path, "c3po", ormText()); err != nil {
		t.Fatal(err)
	}
	n, err := countRows(path, "record")
	if err != nil || n != 0 {
		t.Fatalf("records %d %v", n, err)
	}
}

func TestPublishSkipsAnEmptyDatabaseAndUploadsACopy(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	old := runS3
	var src string
	puts := 0
	runS3 = func(_ string, args []string) (string, error) {
		for i, arg := range args {
			if arg == "put" && i+1 < len(args) {
				puts++
				src = args[i+1]
			}
		}
		return "", nil
	}
	t.Cleanup(func() { runS3 = old })
	if err := prepareAppDB(path, "c3po", ormText()); err != nil {
		t.Fatal(err)
	}
	if err := publishFile(path, "c3po"); err != nil {
		t.Fatal(err)
	}
	if puts != 0 {
		t.Fatalf("empty puts %d", puts)
	}
	insert := "INSERT INTO record(app, kind, item_key, body) VALUES ('tui','session','1','{}');\n"
	if err := sqliteExec(path, insert); err != nil {
		t.Fatal(err)
	}
	if err := publishFile(path, "c3po"); err != nil {
		t.Fatal(err)
	}
	if puts != 1 || src == path || !strings.HasSuffix(src, ".snapshot") {
		t.Fatalf("puts %d src %s", puts, src)
	}
	if _, err := os.Stat(path + ".snapshot"); !os.IsNotExist(err) {
		t.Fatalf("snapshot left behind: %v", err)
	}
}

func TestPublishLeavesAMissingFileMissing(t *testing.T) {
	t.Setenv("SQLX_ROOT", "")
	dir := t.TempDir()
	t.Setenv("SQLX_RUNTIME", dir)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	old := runS3
	called := 0
	runS3 = func(string, []string) (string, error) {
		called++
		return "", nil
	}
	t.Cleanup(func() { runS3 = old })
	if err := Publish(); err != nil {
		t.Fatal(err)
	}
	if called != 0 {
		t.Fatalf("called %d", called)
	}
	if _, err := os.Stat(filepath.Join(dir, "c3po.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("created: %v", err)
	}
}

func TestPutLocalDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	n := 0
	old := publishAfterWrite
	publishAfterWrite = func(string, string) { n++ }
	t.Cleanup(func() { publishAfterWrite = old })
	if err := PutLocal("tui", "session", "1", []byte(`{"screen":"slack"}`)); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("local %d", n)
	}
	if err := Put("tui", "session", "1", []byte(`{"screen":"slack"}`)); err != nil {
		t.Fatal(err)
	}
	if err := DeleteLocal("tui", "session", "1"); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("after local delete %d", n)
	}
	if err := Delete("tui", "session", "gone"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "c3po", "gmail", "tokens", "user.json")
	if err := Write("gmail", "token", "user", path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Remove("gmail", "token", "user", path); err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("durable writes %d", n)
	}
}

func TestInstallerArmsATenMinuteSnapshot(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"OnActiveSec=10min",
		"OnUnitActiveSec=10min",
		"AccuracySec=5s",
		"c3po __snapshot",
		"c3po-snapshot.timer",
		"WantedBy=timers.target",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(text, "start c3po-snapshot.service") {
		t.Fatal("install starts a snapshot")
	}
}
