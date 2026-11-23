package sqlxdb

// Query runs one read against the c3po database and decodes its rows, which
// sqlite returns as a JSON array of objects.
func Query(sql string, dest any) error {
	db, err := open()
	if err != nil {
		return err
	}
	return queryJSON(db, sql, dest)
}

// Exec runs a write in one transaction and returns once it has committed.
// The backup is not part of the write: nebula keep is told the file changed
// and uploads it later. A failed mark leaves the commit in place; keep's
// sweep and status show the copy lagging.
func Exec(sql string) error {
	db, err := open()
	if err != nil {
		return err
	}
	if err := execSQL(db, transaction(sql)); err != nil {
		return err
	}
	_ = keepMark(db, DatabaseName)
	return nil
}

// UsesDB reports whether path is a production record path, which lives in
// the c3po database. Other paths stay files.
func UsesDB(path string) bool {
	return useDB(path)
}

// Quote returns value as a SQL string literal.
func Quote(value string) string {
	return quote(value)
}

// transaction stops at the first failing statement, so a write commits whole
// or not at all.
func transaction(sql string) string {
	return ".bail on\nBEGIN IMMEDIATE;\n" + sql + ";\nCOMMIT;\n"
}
