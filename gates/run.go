// Package gates prints the hardcoded gates an agent judges work against.
// It does not scan a repository, call a model, or decide pass or fail.
package gates

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// sequential is how gates are judged. It belongs in the help, not in list or get output.
const sequential = "Gates run in the list order. A failed gate fails every later gate, and those later gates are not judged."

const frozen = "the gates change only when c3po itself is changed and reinstalled"

// Help is the text for c3po gates and c3po help -f gates.
const Help = `c3po gates
print the hardcoded gates so an agent can judge work against them

  c3po gates
  c3po gates list
    list the types
  c3po gates list <type>
    list the statutes of one type
  c3po gates list <type> <statute>
    list every provision of one statute in section order, one id and
    heading per line
  c3po gates get <id>
    print the body of one provision
  c3po gates help
    show this help
  c3po gates version
    print the c3po version

types: code, essays
statutes: code has architecture; essays has none yet

An id is <type letter>/<statute letter>/<section>, for example C/A/S.1
for code architecture section 1. A section is S and then dot-separated
positive integers: S.1, S.1.1, S.2. Sections order numerically, and a
prefix comes first. Ids are case-sensitive. A body is at most 300
characters.

architecture holds three structural disciplines: data model, state
ownership, and boundaries. Mistakes there are paid for in every future
change, so architecture is stricter than style, framework choice, or
micro-optimization.

` + sequential + `

The agent judges. c3po only prints the text. It does not scan a
repository, call a model, or decide pass or fail. Report one line per id:
  <id>  pass
  <id>  fail
After a fail, each later id is:
  <id>  fail  not judged

The text changes only when c3po itself is changed and reinstalled.
There is no way to edit a gate: no add, edit, or remove command.
There is no gates screen.
`

const usageLine = "usage: c3po gates [list [<type> [<statute>]] | get <id>]"

// Run serves c3po gates. args are the words after gates.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return list(nil, stdout, stderr)
	}
	cmd, rest := args[0], args[1:]
	switch {
	case helpWord(cmd) && len(rest) == 0:
		fmt.Fprint(stdout, Help)
		return 0
	case versionWord(cmd) && len(rest) == 0:
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	case cmd == "list":
		return list(rest, stdout, stderr)
	case cmd == "get":
		return get(rest, stdout, stderr)
	case cmd == "edit" || cmd == "add" || cmd == "remove":
		return usageError(stderr, "there is no %s command; %s", cmd, frozen)
	}
	return usageError(stderr, "unknown command %q; %s", cmd, frozen)
}

func list(args []string, stdout, stderr io.Writer) int {
	if len(args) > 2 {
		return usageError(stderr, "expected at most a type and a statute")
	}
	if len(args) == 0 {
		for _, k := range law {
			fmt.Fprintln(stdout, k.name)
		}
		return 0
	}
	k, ok := findKind(args[0])
	if !ok {
		return usageError(stderr, "unknown type %q; the types are code and essays", args[0])
	}
	if len(args) == 1 {
		listStatutes(stdout, k)
		return 0
	}
	s, ok := k.find(args[1])
	if !ok {
		return usageError(stderr, "unknown statute %q in %s", args[1], k.name)
	}
	for _, p := range s.provisions {
		fmt.Fprintf(stdout, "%s  %s\n", k.id(s, p), p.heading)
	}
	return 0
}

func listStatutes(stdout io.Writer, k kind) {
	if len(k.statutes) == 0 {
		fmt.Fprintln(stdout, "no statutes")
	}
	for _, s := range k.statutes {
		fmt.Fprintln(stdout, s.name)
	}
}

func get(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "expected one id, for example C/A/S.1")
	}
	p, ok := findProvision(args[0])
	if !ok {
		return usageError(stderr, "unknown gate %q", args[0])
	}
	fmt.Fprintln(stdout, p.body)
	return 0
}

func usageError(stderr io.Writer, format string, a ...any) int {
	fmt.Fprintf(stderr, "gates: "+format+"\n", a...)
	fmt.Fprintln(stderr, usageLine)
	return 2
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}
