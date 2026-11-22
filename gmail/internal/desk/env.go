// Package desk is the c3po gmail commands and their rules. It reaches
// Gmail, OAuth, the token store, and the terminal only through the ports
// on Env, which package gmail wires.
package desk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/gmail/internal/mailbox"
	"github.com/ryangerardwilson/c3po/internal/version"
)

// Service is the mailbox the commands call. The Gmail REST client implements it.
type Service interface {
	List(query string, limit int) ([]mailbox.Message, error)
	ListIDs(query string) ([]string, error)
	ListPage(query string, limit int, page string) ([]mailbox.Message, string, error)
	Get(id string) (mailbox.Message, error)
	Meta(id string) (mailbox.Message, error)
	Thread(id string) ([]mailbox.Message, error)
	Send(raw []byte, threadID string) (string, string, error)
	TrashIDs(ids []string) (int, error)
	TrashOne(id string) error
	Modify(id string, add, remove []string) (string, string, error)
	BatchModify(ids, add, remove []string) (int, error)
	Attachment(msgID, attID string) ([]byte, error)
}

// Env is one invocation. Box, Authorize, TokenExists, Terminal, Systemctl,
// and Editor are the ports to Gmail, the OAuth flow, the token store, the
// terminal, systemd, and the editor. C3po is the binary the timer runs.
type Env struct {
	Stdout, Stderr io.Writer
	Stdin          io.Reader
	Paths          config.Paths
	Home           string
	Dir            string
	Now            func() time.Time
	OpenBrowser    func(string) error
	Systemctl      func(args []string) (string, error)
	Editor         func(argv []string) error
	C3po           string
	Box            func(config.Account) (Service, error)
	Authorize      func(clientSecretFile string) (email string, err error)
	TokenExists    func(tokenPath string) (bool, error)
	Terminal       func(io.Writer) bool
	Ask            func(prompt string) (string, error)
}

// RunCode runs c3po gmail and returns the exit code.
func (e Env) RunCode(args []string) int {
	if e.Stdout == nil {
		e.Stdout = io.Discard
	}
	if e.Stderr == nil {
		e.Stderr = io.Discard
	}
	code, err := e.run(args)
	if err == nil {
		return code
	}
	msg, exit := errs.Report(err)
	fmt.Fprintln(e.Stderr, msg)
	return exit
}

func (e Env) run(args []string) (int, error) {
	switch {
	case len(args) == 0, one(args, "help"), one(args, "--help"), one(args, "-h"):
		fmt.Fprint(e.Stdout, Help)
		return 0, nil
	case one(args, "version"), one(args, "--version"), one(args, "-v"):
		fmt.Fprintln(e.Stdout, "c3po "+version.String())
		return 0, nil
	case one(args, "upgrade"):
		fmt.Fprintln(e.Stderr, "upgrade is not part of c3po gmail. Upgrade c3po instead.")
		return 1, nil
	case len(args) > 0 && word(args[0], "help", "version", "upgrade"):
		return 0, errs.Usage("Use: c3po gmail %s", args[0])
	default:
		return e.dispatch(args)
	}
}

func (e Env) paths() config.Paths {
	if e.Paths.Getenv != nil || e.Paths.Home != "" {
		return e.Paths
	}
	return config.DefaultPaths()
}

func (e Env) load() (*config.App, error) {
	p := e.paths()
	return p.Load(p.ConfigPath())
}

func (e Env) home() string {
	if e.Home != "" {
		return e.Home
	}
	if e.Paths.Home != "" {
		return e.Paths.Home
	}
	home, _ := os.UserHomeDir()
	return home
}

func (e Env) service(acct config.Account) (Service, error) {
	if e.Box == nil {
		return nil, errs.Usage("Gmail is not available.")
	}
	return e.Box(acct)
}

func (e Env) authorize(clientSecretFile string) (string, error) {
	if e.Authorize == nil {
		return "", errs.Usage("Gmail authorization is not available.")
	}
	return e.Authorize(clientSecretFile)
}

func (e Env) tokenExists(tokenPath string) bool {
	if e.TokenExists == nil {
		return false
	}
	ok, err := e.TokenExists(tokenPath)
	return err == nil && ok
}

func (e Env) edit(argv []string) error {
	if e.Editor == nil {
		return errors.New("no editor is available")
	}
	return e.Editor(argv)
}

func (e Env) terminal(out io.Writer) bool {
	return e.Terminal != nil && e.Terminal(out)
}

func one(args []string, word string) bool {
	return len(args) == 1 && args[0] == word
}

func word(got string, words ...string) bool {
	for _, w := range words {
		if got == w {
			return true
		}
	}
	return false
}
