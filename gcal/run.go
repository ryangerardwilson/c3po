// Package gcal is the c3po for Google Calendar: numbered account
// presets that list, create, preview, delete, reschedule, and export
// transcripts of events on the primary calendar.
package gcal

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/calendar"
	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
	"github.com/ryangerardwilson/c3po/gcal/internal/oauth"
	"github.com/ryangerardwilson/c3po/gcal/internal/paths"
	"github.com/ryangerardwilson/c3po/internal/version"
)

const (
	ansiGray  = "\x1b[38;5;245m"
	ansiReset = "\x1b[0m"
)

// env is everything a command touches outside its arguments.
type env struct {
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

func (e *env) getenv(key string) string {
	v, _ := e.lookup(key)
	return v
}

func (e *env) dirs() paths.Dirs { return paths.Resolve(e.getenv) }

func (e *env) oauth() oauth.Options {
	return oauth.Options{
		HTTP: e.http, Dirs: e.dirs(), UserinfoURL: e.userinfoURL,
		Now: e.now, OpenBrowser: e.openBrowser, Out: e.stdout,
	}
}

// Run runs c3po gcal with the arguments after "gcal".
func Run(args []string, stdout, stderr io.Writer) int {
	e := &env{
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
	e.editor = func(argv []string) error { return runEditor(argv, stdout, stderr) }
	return run(args, e)
}

func run(args []string, e *env) int {
	err := dispatchTop(args, e)
	if err == nil {
		return 0
	}
	if ge, ok := errs.As(err); ok {
		fmt.Fprintln(e.stderr, ge.Msg)
		return ge.Code
	}
	fmt.Fprintf(e.stderr, "gcal: %s\n", err.Error())
	return 1
}

func dispatchTop(args []string, e *env) error {
	if len(args) == 0 || (len(args) == 1 && isHelp(args[0])) {
		printHelp(e)
		return nil
	}
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintln(e.stdout, version.String())
		return nil
	}
	if args[0] == "upgrade" {
		return errs.Usage("upgrade is not part of c3po gcal; upgrade c3po itself")
	}
	if isHelp(args[0]) || args[0] == "version" {
		return errs.Usage("Use: c3po gcal " + args[0])
	}
	if len(args) == 1 && args[0] == "config" {
		return openConfig(e)
	}
	return dispatch(args, e)
}

func isHelp(s string) bool { return s == "help" || s == "-h" || s == "--help" }

func printHelp(e *env) {
	_, noColor := e.lookup("NO_COLOR")
	if !e.tty || noColor {
		fmt.Fprint(e.stdout, Help)
		return
	}
	fmt.Fprint(e.stdout, ansiGray+Help[:len(Help)-1]+ansiReset+"\n")
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
