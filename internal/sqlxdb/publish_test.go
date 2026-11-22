package sqlxdb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
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
	err := prepareAppDB(path, "c3po")
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
	if err := prepareAppDB(path, "c3po"); err != nil {
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
	if err := prepareAppDB(path, "c3po"); err != nil {
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
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("snapshot left behind: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.snapshot")); len(left) != 0 {
		t.Fatalf("snapshots left behind: %v", left)
	}
}

func TestPublishKeepsEachSnapshotUntilItsPutReturns(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	if err := prepareAppDB(path, "c3po"); err != nil {
		t.Fatal(err)
	}
	insert := "INSERT INTO record(app, kind, item_key, body) VALUES ('tui','session','1','{}');\n"
	if err := sqliteExec(path, insert); err != nil {
		t.Fatal(err)
	}
	old := runS3
	var mu sync.Mutex
	var srcs []string
	entered := make(chan string, 1)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(func() { unblock(); runS3 = old })
	runS3 = func(_ string, args []string) (string, error) {
		src := putSource(args)
		if src == "" {
			return "", nil
		}
		if _, err := os.Stat(src); err != nil {
			return "", err
		}
		mu.Lock()
		srcs = append(srcs, src)
		first := len(srcs) == 1
		mu.Unlock()
		if !first {
			return "", nil
		}
		entered <- src
		<-release
		if _, err := os.Stat(src); err != nil {
			return "", fmt.Errorf("source gone during put: %w", err)
		}
		return "", nil
	}
	errs := make(chan error, 2)
	go func() { errs <- publishFile(path, "c3po") }()
	var first string
	select {
	case first = <-entered:
	case err := <-errs:
		t.Fatalf("first publish returned before its put: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("first put never started")
	}
	go func() { errs <- publishFile(path, "c3po") }()
	time.Sleep(200 * time.Millisecond)
	if publishLockFree(t, path) {
		t.Error("publish lock is free during a put")
	}
	if _, err := os.Stat(first); err != nil {
		t.Errorf("first source gone while its put is blocked: %v", err)
	}
	unblock()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("publish did not finish")
		}
	}
	if len(srcs) != 2 || srcs[0] == srcs[1] {
		t.Fatalf("sources %v", srcs)
	}
	for _, src := range srcs {
		if src == path || !strings.HasSuffix(src, ".snapshot") {
			t.Fatalf("source %s", src)
		}
		if _, err := os.Stat(src); !os.IsNotExist(err) {
			t.Fatalf("snapshot %s left behind: %v", src, err)
		}
	}
}

// publishLockFree reports whether another process could take the publish lock now.
func publishLockFree(t *testing.T, path string) bool {
	t.Helper()
	file, err := os.OpenFile(path+".publish.lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return false
	}
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return true
}

func putSource(args []string) string {
	for i, arg := range args {
		if arg == "put" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
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

func TestExecLocalDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	n := 0
	old := publishAfterWrite
	publishAfterWrite = func(string, string) { n++ }
	t.Cleanup(func() { publishAfterWrite = old })
	session := "INSERT INTO tui_session(id, screen) VALUES ('01a17622-5fd9-7bc2-81ff-3e200a8c16b7', 'slack')"
	if err := ExecLocal(session); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("local %d", n)
	}
	if err := Exec("DELETE FROM tui_session"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "c3po", "gmail", "tokens", "user.json")
	if err := WriteToken("gmail", path, []byte(`{"refresh_token":"x"}`)); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
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
