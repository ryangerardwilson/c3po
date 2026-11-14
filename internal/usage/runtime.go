package usage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// wakeBackup and wakeRestore start detached work. Tests replace them.
var wakeBackup = func() { spawnWorker("__usage-backup", "usage-backup.pid") }

var wakeRestore = func() { spawnWorker("__usage-restore", "usage-restore.pid") }

// rememberRuntime stores one run in the runtime sqlite. The Postgres backup
// is started in the background. A missing runtime copy is held in the local
// journal and a restore is started without blocking the command.
func rememberRuntime(feature, actor, day string, okAdd int, last int64) error {
	if !sqlxdb.RuntimeReady() {
		if err := addLocal(feature, actor, day, okAdd, last); err != nil {
			return err
		}
		wakeRestore()
		return nil
	}
	rows, err := localRows()
	if err != nil {
		rows = nil
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(sqlxdb.UsageBody(row.Feature, row.Actor, row.Day, row.N, row.OK, row.Last))
	}
	b.WriteString(sqlxdb.UsageBody(feature, actor, day, 1, okAdd, last))
	script := sqlxdb.WrapUsage(b.String())
	if err := sqlxdb.ExecRuntime(script); err != nil {
		return addLocal(feature, actor, day, okAdd, last)
	}
	if len(rows) > 0 {
		_ = clearJournal()
	}
	if err := enqueue(script); err != nil {
		return err
	}
	wakeBackup()
	return nil
}

func clearJournal() error {
	return updateLocal(func([]sqlxdb.UsageRow) ([]sqlxdb.UsageRow, bool) {
		return []sqlxdb.UsageRow{}, true
	})
}

// FlushBackup applies queued usage statements to the Postgres backup.
// The command that queued them has already returned.
func FlushBackup() {
	defer clearPid("usage-backup.pid")
	for {
		script, err := takeQueue()
		if err != nil || strings.TrimSpace(script) == "" {
			time.Sleep(50 * time.Millisecond)
			script, err = takeQueue()
			if err != nil || strings.TrimSpace(script) == "" {
				return
			}
		}
		_ = sqlxdb.ExecBackup(script)
	}
}

// RestoreRuntime opens the runtime database so sqlx can restore it from the backup.
func RestoreRuntime() {
	defer clearPid("usage-restore.pid")
	_ = sqlxdb.TouchRuntime()
}

func enqueue(script string) error {
	path, err := queuePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	lock, err := lockFile(path + ".lock")
	if err != nil {
		return err
	}
	defer lock.Close()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}
	_, err = file.WriteString(script)
	return err
}

func takeQueue() (string, error) {
	path, err := queuePath()
	if err != nil {
		return "", err
	}
	lock, err := lockFile(path + ".lock")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer lock.Close()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	if err := os.Truncate(path, 0); err != nil {
		return "", err
	}
	return string(data), nil
}

func queuePath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "usage-backup.sql"), nil
}

func lockFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func spawnWorker(arg, pidName string) {
	pidPath, err := pidPath(pidName)
	if err != nil || procAlive(pidPath) {
		return
	}
	_ = os.Remove(pidPath)
	pidFile, err := os.OpenFile(pidPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		_ = pidFile.Close()
		_ = os.Remove(pidPath)
		return
	}
	cmd := exec.Command(exe, arg)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if null, err := os.Open(os.DevNull); err == nil {
		cmd.Stdout = null
		cmd.Stderr = null
		cmd.Stdin = null
	}
	if err := cmd.Start(); err != nil {
		_ = pidFile.Close()
		_ = os.Remove(pidPath)
		return
	}
	_, _ = pidFile.WriteString(strconv.Itoa(cmd.Process.Pid) + "\n")
	_ = pidFile.Close()
	_ = cmd.Process.Release()
}

func pidPath(name string) (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func clearPid(name string) {
	path, err := pidPath(name)
	if err != nil {
		return
	}
	_ = os.Remove(path)
}

func procAlive(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
