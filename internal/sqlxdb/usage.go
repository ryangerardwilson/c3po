package sqlxdb

import (
	"errors"
	"os"
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
	if RuntimeMode() {
		return errRuntimeOnly
	}
	db, err := open()
	if err != nil {
		return err
	}
	return execSQL(db, WrapUsage(UsageBody(feature, actor, day, 1, okAdd, last)))
}

// errRuntimeOnly tells the caller to write the runtime sqlite itself.
// RecordUsage stays on the sqlite engine used by tests and SQLX_ROOT.
var errRuntimeOnly = errors.New("usage belongs on the runtime copy")

// ListUsage returns every stored daily counter. C3PO_SQLX=off returns no rows.
func ListUsage() ([]UsageRow, error) {
	if os.Getenv("C3PO_SQLX") == "off" {
		return nil, nil
	}
	if RuntimeMode() && !RuntimeReady() {
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
