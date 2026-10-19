// Package cli runs the commands of c3po pass.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/pass/internal/clip"
	"github.com/ryangerardwilson/c3po/pass/internal/op"
)

const usage = "Use: c3po pass <vault> <search> | c3po pass <vault> -l | c3po pass <vault> --create | c3po pass <vault> --edit | c3po pass <vault> --delete | c3po pass <vault> --show"

const upgradeRemoved = "upgrade is not part of c3po pass. Upgrade c3po instead."

// Runner holds the streams and the programs one command uses.
type Runner struct {
	Out     io.Writer
	Err     io.Writer
	In      io.Reader
	Op      op.Runner
	Clip    clip.Clipboard
	Help    string
	Version string
}

type action func(Runner, string, []string) int

var actions = map[string]action{
	"-l":       Runner.listVault,
	"--create": Runner.createItem,
	"--edit":   Runner.editItem,
	"--delete": Runner.deleteItem,
	"--show":   Runner.showItem,
}

// Run runs one command and returns the exit code: 0 on success, 1 on
// any failure.
func (r Runner) Run(args []string) int {
	if len(args) == 0 {
		return r.writeHelp()
	}
	if rc, ok := r.word(args); ok {
		return rc
	}
	if strings.HasPrefix(args[0], "-") || len(args) < 2 {
		return r.fail(usage)
	}
	if act, ok := actions[args[1]]; ok {
		return act(r, args[0], args[2:])
	}
	if len(args) > 2 || isGlobArg(args[1]) {
		return r.mapSecrets(args[0], args[1:])
	}
	return r.lookup(args[0], args[1])
}

// word runs help, version, or upgrade. These words come before a vault,
// so a vault with one of these names cannot be named.
func (r Runner) word(args []string) (int, bool) {
	switch args[0] {
	case "help", "-h", "--help":
		return r.alone(args, "Use help by itself.", r.writeHelp), true
	case "version":
		return r.alone(args, "Use version by itself.", r.writeVersion), true
	case "upgrade":
		return r.fail(upgradeRemoved), true
	}
	return 0, false
}

func (r Runner) alone(args []string, message string, run func() int) int {
	if len(args) != 1 {
		return r.fail(message)
	}
	return run()
}

func (r Runner) writeHelp() int {
	_, _ = fmt.Fprint(r.Out, r.Help)
	return 0
}

func (r Runner) writeVersion() int {
	_, _ = fmt.Fprintln(r.Out, r.Version)
	return 0
}

func (r Runner) fail(message string) int {
	_, _ = fmt.Fprintln(r.Err, message)
	return 1
}

// session returns an op client once the token and op are present.
func (r Runner) session() (op.Client, bool) {
	env, err := op.Env()
	if err != nil {
		r.fail(err.Error())
		return op.Client{}, false
	}
	if err := r.Op.LookPath(); err != nil {
		r.fail("op is required.")
		return op.Client{}, false
	}
	return op.Client{Runner: r.Op, Env: env, Err: r.Err}, true
}

// begin trims vault and opens a session.
func (r Runner) begin(vault string) (string, op.Client, bool) {
	vault = strings.TrimSpace(vault)
	if vault == "" {
		r.fail("Vault must not be empty.")
		return "", op.Client{}, false
	}
	client, ok := r.session()
	return vault, client, ok
}

// forShell is true when stdout is a file that is not a terminal: a pipe,
// a regular file, or command substitution.
func (r Runner) forShell() bool {
	file, ok := r.Out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

func (r Runner) input() io.Reader {
	if r.In == nil {
		return os.Stdin
	}
	return r.In
}
