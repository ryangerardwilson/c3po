// Package gmail is c3po gmail, the Gmail desk. It wires the Gmail REST
// client, the OAuth flow, the token store, the terminal, systemd, and the
// editor into the commands in package desk.
package gmail

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/term"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/desk"
	"github.com/ryangerardwilson/c3po/gmail/internal/oauth"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// Run runs c3po gmail and returns the exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return wire(desk.Env{Stdout: stdout, Stderr: stderr, Stdin: os.Stdin}).RunCode(args)
}

func wire(e desk.Env) desk.Env {
	opt := oauth.Options{Paths: config.DefaultPaths(), Out: e.Stdout}
	e.Box = func(acct config.Account) (desk.Service, error) {
		return &api.Client{Access: func() (string, error) {
			return opt.AccessToken(acct.Email, acct.ClientSecretFile)
		}}, nil
	}
	e.Authorize = func(clientSecretFile string) (string, error) {
		authorized, err := opt.Authorize(clientSecretFile)
		return authorized.Email, err
	}
	e.TokenExists = func(tokenPath string) (bool, error) {
		return sqlxdb.TokenExists("gmail", tokenPath)
	}
	e.Terminal = writerTerminal
	e.Systemctl = systemctlUser
	e.Editor = func(argv []string) error {
		proc := exec.Command(argv[0], argv[1:]...)
		proc.Stdin, proc.Stdout, proc.Stderr = e.Stdin, e.Stdout, e.Stderr
		return proc.Run()
	}
	e.C3po = c3poBin()
	return e
}

// systemctlUser runs systemctl --user and returns its combined output.
func systemctlUser(args []string) (string, error) {
	out, err := exec.Command("systemctl", append([]string{"--user"}, args...)...).CombinedOutput()
	return string(out), err
}

// c3poBin is the running c3po, else the c3po on PATH, else "c3po".
func c3poBin() string {
	if exe, err := os.Executable(); err == nil && filepath.Base(exe) == "c3po" {
		return exe
	}
	if path, err := exec.LookPath("c3po"); err == nil {
		return path
	}
	return "c3po"
}

func writerTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
