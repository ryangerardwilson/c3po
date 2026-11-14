package usage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// wakeBackup and wakeRestore start detached work. Tests replace them.
var wakeBackup = func() { spawnWorker("__usage-backup", backupLock) }

var wakeRestore = func() { spawnWorker("__usage-restore", restoreLock) }

// The usage worker waits workerIdle for another increment before it exits
// and looks every workerPoll. Tests replace them.
var (
	workerIdle = 3 * time.Second
	workerPoll = 100 * time.Millisecond
)

// A worker holds its lock file while it runs. A held lock means a worker is
// alive, so a command does not start another one.
const (
	backupLock  = "usage-backup.lock"
	restoreLock = "usage-restore.lock"
)

// rememberRuntime appends one run to the local journal and wakes the
// detached worker. The command opens neither the runtime sqlite nor the
// s3 backup. A missing runtime copy wakes a restore instead.
func rememberRuntime(feature, actor, day string, okAdd int, last int64) error {
	if err := addLocal(feature, actor, day, okAdd, last); err != nil {
		return err
	}
	if sqlxdb.RuntimeReady() {
		wakeBackup()
	} else {
		wakeRestore()
	}
	return nil
}

// FlushBackup is the detached usage worker. It writes the journal to the
// runtime sqlite, then publishes that file with s3, and stays
// until no increment has arrived for workerIdle, so a burst of commands
// shares one worker.
func FlushBackup() {
	for {
		lock, ok := tryLock(backupLock)
		if !ok {
			return // another worker is draining
		}
		w := &worker{}
		seen := w.drain()
		_ = lock.Close()
		// A command that appended after the last look found the lock held
		// and started no worker. Take its increment rather than strand it.
		if journalStamp() == seen {
			return
		}
	}
}

// RestoreRuntime opens the runtime database so sqlx can restore it from the
// backup, then drains the journal that waited for it.
func RestoreRuntime() {
	lock, ok := tryLock(restoreLock)
	if !ok {
		return
	}
	_ = sqlxdb.TouchRuntime()
	_ = lock.Close()
	if sqlxdb.RuntimeReady() {
		FlushBackup()
	}
}

// subtractJournal removes rows the runtime sqlite already has. Tests replace
// it so a failed removal cannot apply the same increment again.
var subtractJournal = subtractLocal

type worker struct {
	backupDown bool   // the last send failed; wait for new rows before retrying
	applied    string // runtime script already executed while its journal removal failed
}

// drain moves batches until none has arrived for workerIdle and returns the
// journal stamp it saw last.
func (w *worker) drain() stamp {
	quiet := time.Now()
	for {
		seen := journalStamp()
		if w.step() {
			quiet = time.Now()
			continue
		}
		if time.Since(quiet) >= workerIdle {
			return seen
		}
		time.Sleep(workerPoll)
	}
}

// step moves one batch and reports whether anything moved.
func (w *worker) step() bool {
	moved := w.toRuntime()
	if moved {
		w.backupDown = false
	}
	if w.backupDown {
		return moved
	}
	sent, err := toBackup()
	w.backupDown = err != nil
	return moved || sent
}

// toRuntime writes the journal to the runtime sqlite, then removes those rows
// and queues the same statements for the backup. A failed sqlite write leaves
// the journal alone. applied stops a failed removal from writing those rows
// again on the next look.
func (w *worker) toRuntime() bool {
	if !sqlxdb.RuntimeReady() {
		return false
	}
	rows, err := localRows()
	if err != nil || len(rows) == 0 {
		w.applied = ""
		return false
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(sqlxdb.UsageBody(row.Feature, row.Actor, row.Day, row.N, row.OK, row.Last))
	}
	script := sqlxdb.WrapUsage(b.String())
	if script != w.applied {
		if err := sqlxdb.ExecRuntime(script); err != nil {
			return false
		}
		w.applied = script
	}
	if err := subtractJournal(rows); err != nil {
		return false
	}
	w.applied = ""
	_ = enqueue(script)
	return true
}

// toBackup publishes the runtime sqlite. A failed publish puts the
// queued statements back for the next worker.
func toBackup() (bool, error) {
	script, err := takeQueue()
	if err != nil || strings.TrimSpace(script) == "" {
		return false, err
	}
	if err := sqlxdb.ExecBackup(); err != nil {
		_ = enqueue(script)
		return false, err
	}
	return true, nil
}

// stamp identifies one version of the journal file.
type stamp struct {
	size  int64
	mtime int64
}

func journalStamp() stamp {
	path, err := journalPath()
	if err != nil {
		return stamp{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return stamp{}
	}
	return stamp{size: info.Size(), mtime: info.ModTime().UnixNano()}
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

// tryLock takes a worker lock in the state directory without waiting.
// Closing the file releases it.
func tryLock(name string) (*os.File, bool) {
	dir, err := stateDir()
	if err != nil {
		return nil, false
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, false
	}
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, false
	}
	return file, true
}

// spawnWorker starts arg as a detached process unless a worker already holds
// lockName.
func spawnWorker(arg, lockName string) {
	lock, ok := tryLock(lockName)
	if !ok {
		return
	}
	_ = lock.Close()
	exe, err := os.Executable()
	if err != nil {
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
		return
	}
	_ = cmd.Process.Release()
}
