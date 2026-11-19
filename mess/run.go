// Package mess reports how messy the current repository's linked worktrees
// are, what an agent should clean up, and what it cannot do until it does.
package mess

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// Help is the text for c3po mess help and c3po help -f mess.
const Help = `c3po mess
how messy this repository's linked worktrees are

  c3po mess
    print the report; nothing is deleted
  c3po mess tidy [--session <id>]
    remove the worktrees the report calls clean
  c3po mess tidy add <path> [-b <branch>] [--session <id>]
    create a linked worktree and mark it for tidy
  c3po mess tidy --hook
    read a SessionEnd JSON object on stdin and tidy that session
  c3po mess help
    show this help
  c3po mess version
    print the c3po version

clean    a marked worktree, pushed and clean; c3po mess tidy removes it
keep     a marked worktree that is not eligible, with the first reason
other    an unmarked linked worktree; reported, never removed
blocked  what the agent cannot do until it cleans up

Only the current repository is read. The main worktree never appears.
Removal is git worktree remove without --force. Nothing is stashed,
reset, pruned, or deleted from a branch.
`

// Run serves c3po mess. args are the words after mess.
func Run(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "mess: %s\n", err)
		return 1
	}
	return run(args, cwd, os.Stdin, stdout, stderr, DefaultProbe())
}

func run(args []string, cwd string, stdin io.Reader, stdout, stderr io.Writer, p Probe) int {
	if len(args) == 1 && helpWord(args[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(args) == 1 && versionWord(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if len(args) == 0 {
		return report(cwd, stdout, stderr, p)
	}
	if args[0] != "tidy" {
		return usageError(stderr, fmt.Errorf("unknown command %q", args[0]))
	}
	o, err := parseTidy(args[1:])
	if err != nil {
		return usageError(stderr, err)
	}
	switch {
	case o.hook:
		return hookTidy(stdin, stdout, p)
	case o.add:
		return add(cwd, o, stdout, stderr, p)
	default:
		return tidy(cwd, o.session, stdout, stderr, p)
	}
}

func report(cwd string, stdout, stderr io.Writer, p Probe) int {
	rep, err := Inspect(cwd, p)
	if err != nil {
		fmt.Fprintf(stderr, "mess: %s\n", err)
		return 1
	}
	rep.Write(stdout)
	return 0
}

func usageError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "mess: %s\n", err)
	fmt.Fprint(stderr, Help)
	return 2
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}

// tidyOpts are the words after c3po mess tidy.
type tidyOpts struct {
	hook    bool
	add     bool
	session *string // nil: every session
	path    string
	branch  string
}

func parseTidy(args []string) (tidyOpts, error) {
	var o tidyOpts
	if len(args) > 0 && args[0] == "add" {
		o.add = true
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--hook":
			o.hook = true
		case "--session", "-b":
			if i+1 >= len(args) {
				return o, fmt.Errorf("%s needs a value", args[i])
			}
			o.set(args[i], args[i+1])
			i++
		default:
			if !o.add || o.path != "" {
				return o, fmt.Errorf("unexpected argument %q", args[i])
			}
			o.path = args[i]
		}
	}
	return o, o.check()
}

func (o *tidyOpts) set(flag, value string) {
	if flag == "-b" {
		o.branch = value
		return
	}
	o.session = &value
}

func (o tidyOpts) check() error {
	if o.add && o.path == "" {
		return errors.New("tidy add needs a path")
	}
	if o.add && o.hook {
		return errors.New("tidy add does not take --hook")
	}
	if !o.add && o.branch != "" {
		return errors.New("-b belongs to tidy add")
	}
	return nil
}
