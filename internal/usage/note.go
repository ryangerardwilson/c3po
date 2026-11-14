package usage

import (
	"os"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// Now is the clock for the UTC day and the last-used time. Tests replace it.
var Now = time.Now

// Note records one finished invocation. A failure is ignored so the command's
// exit code and stderr stay what the command itself produced.
// On the production path the increment is only appended to the local
// journal; a detached worker writes the runtime sqlite and then the Postgres
// backup, so the command neither runs sqlx nor dials Postgres.
// A test that does not set SQLX_ROOT records nothing, so go test does not
// write the login-home database. C3PO_SQLX=off records nothing.
func Note(code int, argv []string) {
	if skip() {
		return
	}
	feature := Feature(argv)
	if feature == "" {
		return
	}
	okAdd := 0
	if code == 0 {
		okAdd = 1
	}
	when := Now().UTC()
	if sqlxdb.RuntimeMode() {
		_ = rememberRuntime(feature, Actor(), when.Format("2006-01-02"), okAdd, when.Unix())
		return
	}
	_ = sqlxdb.RecordUsage(feature, Actor(), when.Format("2006-01-02"), okAdd, when.Unix())
}

func skip() bool {
	if os.Getenv("C3PO_SQLX") == "off" {
		return true
	}
	if testing.Testing() && os.Getenv("SQLX_ROOT") == "" {
		return true
	}
	return false
}
