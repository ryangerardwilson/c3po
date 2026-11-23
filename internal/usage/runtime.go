package usage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
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
// runtime sqlite, then marks that file changed for nebula keep, and stays
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

type worker struct{}

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

// step moves one batch and reports whether anything moved. A batch that
// moved marks the runtime sqlite changed; nebula keep uploads it later and
// keeps its own record of what is not backed up yet.
func (w *worker) step() bool {
	moved := w.toRuntime()
	if moved {
		_ = sqlxdb.ExecBackup()
	}
	return moved
}

// toRuntime moves one journal batch into the runtime sqlite and drops it.
// The batch id commits with its increments, so a batch that a dead worker
// committed but did not drop is dropped, not applied again (statute 2.8).
func (w *worker) toRuntime() bool {
	if !sqlxdb.RuntimeReady() {
		return false
	}
	id, rows, err := takeBatch()
	if err != nil || id == "" {
		return false
	}
	done, err := sqlxdb.UsageBatchApplied(id)
	if err != nil {
		return false
	}
	if !done && len(rows) > 0 {
		var b strings.Builder
		for _, row := range rows {
			b.WriteString(sqlxdb.UsageBody(row.Feature, row.Actor, row.Day, row.N, row.OK, row.Last))
		}
		if err := sqlxdb.ExecRuntime(sqlxdb.WrapUsageBatch(id, b.String(), time.Now().Unix())); err != nil {
			return false
		}
	}
	return finishBatch(id) == nil
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
// lockName. A test binary starts none: os.Executable is the test binary,
// which reruns its whole suite on the worker argument and forks without end.
func spawnWorker(arg, lockName string) {
	if testing.Testing() {
		return
	}
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
