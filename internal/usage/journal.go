package usage

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	rows, err := readJournal(path)
	if err != nil {
		return err
	}
	next, write := fn(rows)
	if !write {
		return nil
	}
	return writeJournal(path, next)
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
