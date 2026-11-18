package sqlxdb

import (
	"fmt"
	"os"
	"sync"
)

var publishMu sync.Mutex

// Publish uploads the runtime database when it exists and has a record.
// A missing file stays missing. SQLX_ROOT does not upload.
func Publish() error {
	if os.Getenv("SQLX_ROOT") != "" {
		return nil
	}
	path := dbFile(DatabaseName)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	return publishFile(path, DatabaseName)
}

func publishFile(path, name string) error {
	if path == "" {
		return nil
	}
	publishMu.Lock()
	defer publishMu.Unlock()
	if !canPublish(path) {
		return nil
	}
	dest, err := snapshotCopy(path)
	if err != nil {
		return err
	}
	defer os.Remove(dest)
	if !canPublish(dest) {
		return nil
	}
	if bin, args, err := s3Args("make", "bucket", "backups"); err == nil {
		_, _ = runS3(bin, args)
	}
	bin, args, err := s3Args("put", dest, "s3://backups/"+name+".sqlite")
	if err != nil {
		return err
	}
	msg, err := runS3(bin, args)
	if err != nil {
		return fmt.Errorf("s3 put s3://backups/%s.sqlite: %s", name, s3Text(msg, err))
	}
	return nil
}

func canPublish(path string) bool {
	exists, err := tableExists(path, "record")
	if err != nil || !exists {
		return err == nil && !exists
	}
	n, err := countRows(path, "record")
	return err == nil && n > 0
}

func countRows(path, table string) (int, error) {
	var rows []struct {
		N int `json:"n"`
	}
	query := "SELECT COUNT(*) AS n FROM " + quoteIdent(table)
	if err := sqliteQueryJSON(path, query, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("%s: sqlite returned no count", path)
	}
	return rows[0].N, nil
}

func snapshotCopy(path string) (string, error) {
	dest := path + ".snapshot"
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := sqliteExec(path, "VACUUM INTO "+quote(dest)+";\n"); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	return dest, nil
}
