// Package gmail is c3po gmail, the Gmail desk.
package gmail

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/internal/version"
)

// Service is the mailbox the commands call. *api.Client implements it.
type Service interface {
	List(query string, limit int) ([]api.Message, error)
	ListIDs(query string) ([]string, error)
	ListPage(query string, limit int, page string) ([]api.Message, string, error)
	Get(id string) (api.Message, error)
	Meta(id string) (api.Message, error)
	Thread(id string) ([]api.Message, error)
	Send(raw []byte, threadID string) (string, string, error)
	TrashIDs(ids []string) (int, error)
	TrashOne(id string) error
	Modify(id string, add, remove []string) (string, string, error)
	BatchModify(ids, add, remove []string) (int, error)
	Attachment(msgID, attID string) ([]byte, error)
	UnreadSenders(progress func(string, int, int)) (map[string]int, error)
}

// Env is one invocation. Tests replace Box and Systemctl.
type Env struct {
	Stdout, Stderr io.Writer
	Stdin          io.Reader
	Paths          config.Paths
	Home           string
	Dir            string
	Now            func() time.Time
	OpenBrowser    func(string) error
	Systemctl      func(args []string) (string, error)
	Editor         func(path string) error
	Box            func(config.Account) (Service, error)
	Ask            func(prompt string) (string, error)
	HTTP           *http.Client
	Decide         func([]mailBrief) (map[string]mailVerdict, error)
}

// Run runs c3po gmail and returns the exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return Env{Stdout: stdout, Stderr: stderr, Stdin: os.Stdin}.runCode(args)
}

func (e Env) runCode(args []string) int {
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
