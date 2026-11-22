package sqlxdb

import (
	"errors"
	"os"
	"path/filepath"
)

// ReadToken returns the OAuth token document of one app and the account
// whose email names path (<dir>/<email>.json). A production record path reads
// google_token; another path is the file.
func ReadToken(app, path string) ([]byte, error) {
	if !useDB(path) {
		return os.ReadFile(path)
	}
	db, err := open()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Document string `json:"document"`
	}
	if err := queryJSON(db, "SELECT document FROM google_token WHERE "+tokenWhere(app, path), &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, os.ErrNotExist
	}
	return []byte(rows[0].Document), nil
}

// WriteToken stores the token document and publishes the database. Another
// path is written as a file readable only by the user.
func WriteToken(app, path string, document []byte) error {
	if !useDB(path) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		return os.WriteFile(path, document, 0o600)
	}
	db, err := open()
	if err != nil {
		return err
	}
	if err := execSQL(db, tokenUpsert(app, Key(path), document)); err != nil {
		return err
	}
	publishAfterWrite(db, DatabaseName)
	return nil
}

// TokenExists reports whether the account named by path has a token.
func TokenExists(app, path string) (bool, error) {
	_, err := ReadToken(app, path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func tokenWhere(app, path string) string {
	return "app = " + quote(app) + " AND email = " + quote(Key(path))
}

func tokenUpsert(app, email string, document []byte) string {
	return "INSERT INTO google_token(app, email, document) VALUES (" + quote(app) + ", " + quote(email) + ", " + quote(string(document)) +
		") ON CONFLICT(app, email) DO UPDATE SET document = excluded.document"
}
