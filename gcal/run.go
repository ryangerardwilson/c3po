// Package gcal is the c3po for Google Calendar: numbered account
// presets that list, create, preview, delete, reschedule, and export
// transcripts of events on the primary calendar. It is the composition
// root: it builds the HTTP client, the Google adapters, the token store,
// and the browser, editor, and terminal hooks, and hands them to desk.
package gcal

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/calendar"
	"github.com/ryangerardwilson/c3po/gcal/internal/desk"
	"github.com/ryangerardwilson/c3po/gcal/internal/oauth"
)

// wiring is the outside world a gcal process is built from.
type wiring struct {
	lookup       func(string) (string, bool)
	stdin        *bufio.Reader
	stdout       io.Writer
	stderr       io.Writer
	http         *http.Client
	calendarBase string
	driveBase    string
	userinfoURL  string
	now          func() time.Time
	openBrowser  func(string) error
	editor       func(argv []string) error
	newID        func() string
	tty          bool
}

// Run runs c3po gcal with the arguments after "gcal".
func Run(args []string, stdout, stderr io.Writer) int {
	w := wiring{
		lookup:       os.LookupEnv,
		stdin:        bufio.NewReader(os.Stdin),
		stdout:       stdout,
		stderr:       stderr,
		http:         &http.Client{Timeout: 2 * time.Minute},
		calendarBase: calendar.CalendarBase,
		driveBase:    calendar.DriveBase,
		userinfoURL:  oauth.UserinfoURL,
		now:          time.Now,
		openBrowser:  openBrowser,
		newID:        uuidHex,
		tty:          isTerminal(stdout),
	}
	w.editor = func(argv []string) error { return editorFailure(runEditor(argv, stdout, stderr)) }
	return desk.Run(args, w.env())
}

// env hands the wiring to desk through its ports.
func (w wiring) env() *desk.Env {
	g := google{
		getenv: func(key string) string {
			v, _ := w.lookup(key)
			return v
		},
		http: w.http, calendarBase: w.calendarBase, driveBase: w.driveBase,
		userinfoURL: w.userinfoURL, now: w.now, openBrowser: w.openBrowser, out: w.stdout,
	}
	return &desk.Env{
		Lookup: w.lookup, Stdin: w.stdin, Stdout: w.stdout, Stderr: w.stderr,
		Now: w.now, NewID: w.newID, TTY: w.tty, Editor: desk.EditorFunc(w.editor),
		Google: g, Authorizer: g, Tokens: g,
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func uuidHex() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return hex.EncodeToString(b[:])
}

// openBrowser starts the desktop browser without waiting for it. The
// URL is printed as well, so a failure here is not fatal.
func openBrowser(url string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	cmd := exec.Command(name, url)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func runEditor(argv []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, stdout, stderr
	return cmd.Run()
}

// editorFailure turns a process error into the desk editor errors: a
// nonzero exit, or a command that does not exist.
func editorFailure(err error) error {
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exitErr):
		return &desk.EditorExitError{Code: exitErr.ExitCode()}
	case errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist):
		return desk.ErrEditorNotFound
	default:
		return err
	}
}
