package usage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// addLocal adds one run to the on-disk counter. The same feature, actor, and
// UTC day share one row. Callers ignore the error.
func addLocal(feature, actor, day string, okAdd int, last int64) error {
	if feature == "" || actor == "" || day == "" {
		return nil
	}
	if okAdd != 0 {
		okAdd = 1
	}
	return updateLocal(func(rows []sqlxdb.UsageRow) ([]sqlxdb.UsageRow, bool) {
		for i := range rows {
			row := &rows[i]
			if row.Feature != feature || row.Actor != actor || row.Day != day {
				continue
			}
			row.N++
			row.OK += okAdd
			if last > row.Last {
				row.Last = last
			}
			return rows, true
		}
		return append(rows, sqlxdb.UsageRow{
			Feature: feature,
			Actor:   actor,
			Day:     day,
			N:       1,
			OK:      okAdd,
			Last:    last,
		}), true
	})
}

// takeBatch moves the journal aside as one batch with a fresh id, under the
// journal lock, so increments that arrive meanwhile start a new journal. A
// batch a dead worker left is returned again with its own id.
func takeBatch() (string, []sqlxdb.UsageRow, error) {
	path, err := journalPath()
	if err != nil {
		return "", nil, err
	}
	var id string
	var rows []sqlxdb.UsageRow
	err = withJournalLock(path, func() error {
		left, _ := filepath.Glob(path + ".batch.*")
		if len(left) > 0 {
			id = strings.TrimPrefix(left[0], path+".batch.")
			rows, err = readJournal(left[0])
			return err
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		}
		var raw [8]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return err
		}
		id = hex.EncodeToString(raw[:])
		if err := os.Rename(path, path+".batch."+id); err != nil {
			return err
		}
		rows, err = readJournal(path + ".batch." + id)
		return err
	})
	return id, rows, err
}

// finishBatch drops a batch the runtime sqlite holds. Tests replace it to
// stop a worker between the commit and this step.
var finishBatch = func(id string) error {
	path, err := journalPath()
	if err != nil {
		return err
	}
	return withJournalLock(path, func() error {
		err := os.Remove(path + ".batch." + id)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	})
}

func localRows() ([]sqlxdb.UsageRow, error) {
	path, err := journalPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rows []sqlxdb.UsageRow
	err = updateLocal(func(have []sqlxdb.UsageRow) ([]sqlxdb.UsageRow, bool) {
		rows = append([]sqlxdb.UsageRow(nil), have...)
		return have, false
	})
	return rows, err
}

func updateLocal(fn func([]sqlxdb.UsageRow) ([]sqlxdb.UsageRow, bool)) error {
	path, err := journalPath()
	if err != nil {
		return err
	}
	return withJournalLock(path, func() error {
		rows, err := readJournal(path)
		if err != nil {
			return err
		}
		next, write := fn(rows)
		if !write {
			return nil
		}
		return writeJournal(path, next)
	})
}

// withJournalLock runs fn holding the journal's lock file.
func withJournalLock(path string, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn()
}

func journalPath() (string, error) {
	state, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "usage.json"), nil
}

func stateDir() (string, error) {
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		state = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(state, "c3po"), nil
}

func readJournal(path string) ([]sqlxdb.UsageRow, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var rows []sqlxdb.UsageRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func writeJournal(path string, rows []sqlxdb.UsageRow) error {
	data, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), "usage-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o600); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
