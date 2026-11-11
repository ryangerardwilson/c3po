package sqlxdb

import (
	"os"
	"strconv"
)

// UsageRow is one daily counter: one feature, one actor, one UTC day.
// N is how many times it ran. OK is how many of those exited 0.
// Last is the unix time of the latest run. Day is YYYY-MM-DD in UTC.
type UsageRow struct {
	Feature string `json:"feature"`
	Actor   string `json:"actor"`
	Day     string `json:"day"`
	N       int    `json:"n"`
	OK      int    `json:"ok"`
	Last    int64  `json:"last"`
}

// RecordUsage adds one run to the daily counter. okAdd is 1 when the command
// exited 0. The update and the insert share one immediate transaction so two
// processes cannot insert the same feature, actor, and day.
// C3PO_SQLX=off writes nothing. The caller ignores the error.
func RecordUsage(feature, actor, day string, okAdd int, last int64) error {
	if os.Getenv("C3PO_SQLX") == "off" {
		return nil
	}
	if feature == "" || actor == "" || day == "" {
		return nil
	}
	if okAdd != 0 {
		okAdd = 1
	}
	db, err := open()
	if err != nil {
		return err
	}
	okText := strconv.Itoa(okAdd)
	lastText := strconv.FormatInt(last, 10)
	sql := "BEGIN IMMEDIATE;\n" +
		"UPDATE usage SET n = n + 1, ok = ok + " + okText +
		", last = CASE WHEN last > " + lastText + " THEN last ELSE " + lastText +
		" END WHERE feature = " + quote(feature) + " AND actor = " + quote(actor) +
		" AND day = " + quote(day) + ";\n" +
		"INSERT INTO usage(feature, actor, day, n, ok, last) SELECT " +
		quote(feature) + ", " + quote(actor) + ", " + quote(day) + ", 1, " + okText + ", " + lastText +
		" WHERE changes() = 0;\n" +
		"COMMIT;\n"
	return execSQL(db, sql)
}

// ListUsage returns every stored daily counter. C3PO_SQLX=off returns no rows.
func ListUsage() ([]UsageRow, error) {
	if os.Getenv("C3PO_SQLX") == "off" {
		return nil, nil
	}
	db, err := open()
	if err != nil {
		return nil, err
	}
	var rows []UsageRow
	err = queryJSON(db, `SELECT feature, actor, day, n, ok, last FROM usage`, &rows)
	return rows, err
}
