package sqlxdb

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

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

// publishFile holds the publish lock from before the copy until the put has
// returned and the copy is removed, so a publisher in another process waits.
func publishFile(path, name string) error {
	if path == "" {
		return nil
	}
	lock, err := publishLock(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer lock.Close()
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

// publishLock takes the exclusive publish flock beside the database.
// Closing the file releases it.
func publishLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path+".publish.lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// canPublish refuses a c3po database that holds no data, so an empty file
// never replaces the backup. The migration ledger is not data.
func canPublish(path string) bool {
	exists, err := tableExists(path, "record")
	if err != nil || !exists {
		return err == nil && !exists
	}
	var rows []struct {
		Name string `json:"name"`
	}
	if err := sqliteQueryJSON(path, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name <> 'schema_migration'", &rows); err != nil {
		return false
	}
	for _, row := range rows {
		if n, err := countRows(path, row.Name); err == nil && n > 0 {
			return true
		}
	}
	return false
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

// snapshotCopy writes a consistent copy to a new name beside the database.
// VACUUM INTO needs the destination absent, so the reserved file is removed.
func snapshotCopy(path string) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.snapshot")
	if err != nil {
		return "", err
	}
	dest := file.Name()
	_ = file.Close()
	if err := os.Remove(dest); err != nil {
		return "", err
	}
	if err := sqliteExec(path, "VACUUM INTO "+quote(dest)+";\n"); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	return dest, nil
}
