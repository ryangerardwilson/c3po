package sqlxdb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// nebula keep owns the durable copy of this file at s3://backups/c3po.sqlite.
// c3po says "the file changed" after each commit and asks for a missing file
// at startup; the upload, the snapshot, the locking, and the rule that an
// empty file never replaces the backup all live in nebula.

const (
	markWait    = 5 * time.Second  // local only: a record and a lock
	restoreWait = 90 * time.Second // one download over SSH
)

// keepMark tells nebula keep the database changed. It returns once nebula
// has recorded the change; the upload happens later in nebula's worker.
// Tests replace it.
var keepMark = func(path, name string) error {
	return runKeep(markWait, "mark", name, path)
}

// keepRestore asks nebula keep to bring a missing database back. It fails,
// and leaves no file, when the backup is unreachable. Tests replace it.
var keepRestore = func(path, name string) error {
	return runKeep(restoreWait, "restore", name, path)
}

// runKeep reaches nebula only for the runtime file. The local SQLX_ROOT
// tree that tests use has no backup.
// A test binary reaches only the nebula NEBULA_BIN names, never the
// operator's, because the login shell's SQLX_ENGINE=postgres would
// otherwise put every test in runtime mode.
func runKeep(wait time.Duration, verb, name, path string) error {
	if !RuntimeMode() || path == "" {
		return nil
	}
	if testing.Testing() && os.Getenv("NEBULA_BIN") == "" {
		return nil
	}
	bin, err := nebulaBin()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "keep", verb, name, path).CombinedOutput()
	if err == nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		text = err.Error()
	}
	return fmt.Errorf("nebula keep %s %s: %s", verb, name, text)
}

func nebulaBin() (string, error) {
	if bin := os.Getenv("NEBULA_BIN"); bin != "" {
		return bin, nil
	}
	bin, err := exec.LookPath("nebula")
	if err != nil {
		return "", errors.New("nebula is not on PATH")
	}
	return bin, nil
}

// MarkChanged tells nebula keep the runtime database changed when it exists.
// A missing file stays missing. c3po __snapshot runs it; that command stays
// one release as an alias for timers that still name it, and goes once
// c3po-snapshot.timer is retired everywhere.
func MarkChanged() error {
	path := dbFile(DatabaseName)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	return keepMark(path, DatabaseName)
}
