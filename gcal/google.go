package gcal

import (
	"io"
	"net/http"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/calendar"
	"github.com/ryangerardwilson/c3po/gcal/internal/desk"
	"github.com/ryangerardwilson/c3po/gcal/internal/oauth"
	"github.com/ryangerardwilson/c3po/gcal/internal/paths"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// google adapts the OAuth flow, the Calendar and Drive clients, and the
// sqlx token store to the desk ports Connector, Authorizer, and Tokens.
// Directories resolve on each call, as the commands resolve them.
type google struct {
	getenv       func(string) string
	http         *http.Client
	calendarBase string
	driveBase    string
	userinfoURL  string
	now          func() time.Time
	openBrowser  func(string) error
	out          io.Writer
}

func (g google) dirs() paths.Dirs { return paths.Resolve(g.getenv) }

func (g google) oauth() oauth.Options {
	return oauth.Options{
		HTTP: g.http, Dirs: g.dirs(), UserinfoURL: g.userinfoURL,
		Now: g.now, OpenBrowser: g.openBrowser, Out: g.out,
	}
}

// Calendar returns the Calendar client with a calendar-scoped token.
func (g google) Calendar(email, clientSecretFile string) (desk.Calendar, error) {
	token, err := g.oauth().AccessToken(email, clientSecretFile, oauth.CalendarScopes)
	if err != nil {
		return nil, err
	}
	return calendar.Client{HTTP: g.http, Base: g.calendarBase, Token: token}, nil
}

// Drive returns the Drive client with a transcript-scoped token.
func (g google) Drive(email, clientSecretFile string) (desk.Drive, error) {
	token, err := g.oauth().AccessToken(email, clientSecretFile, oauth.TranscriptScopes)
	if err != nil {
		return nil, err
	}
	return calendar.Client{HTTP: g.http, Base: g.driveBase, Token: token}, nil
}

// Authorize runs the browser flow with the calendar scopes.
func (g google) Authorize(clientSecretFile string) (string, error) {
	authorized, err := g.oauth().Authorize(clientSecretFile, oauth.CalendarScopes)
	if err != nil {
		return "", err
	}
	return authorized.Email, nil
}

// Exists reports a saved token for email in the sqlx token store.
func (g google) Exists(email string) (bool, error) {
	return sqlxdb.TokenExists("gcal", g.dirs().TokenFile(email))
}
