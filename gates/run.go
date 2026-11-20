// Package gates prints the hardcoded gates an agent judges work against.
// It does not scan a repository, call a model, or decide pass or fail.
package gates

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// sequential is printed whenever a type's gates are listed or shown as a set.
const sequential = "Gates of a type run in section order. A failed gate fails every later gate, and those later gates are not judged."

const frozen = "the gates change only when c3po itself is changed and reinstalled"

// Help is the text for c3po gates and c3po help -f gates.
const Help = `c3po gates
print the hardcoded gates so an agent can judge work against them

  c3po gates
  c3po gates list
    list the types
  c3po gates list <type>
    list the gates of one type, one per line
  c3po gates show <type>
    print every gate of one type in section order
  c3po gates show <type> <section>
    print one gate
  c3po gates help
    show this help
  c3po gates version
    print the c3po version

types: code, essays

` + sequential + `

The agent judges. c3po only prints the text. It does not scan a
repository, call a model, or decide pass or fail. Report one line per
section:
  Section <number>  <title>  pass
  Section <number>  <title>  fail
After a fail, each later section is:
  Section <number>  <title>  fail  not judged

The text changes only when c3po itself is changed and reinstalled. There
is no add, edit, or remove command. There is no gates screen.
`

const usageLine = "usage: c3po gates [list [<type>] | show <type> [<section>]]"

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
	case cmd == "show":
		return show(rest, stdout, stderr)
	}
	return usageError(stderr, "unknown command %q; %s", cmd, frozen)
}

func list(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		return usageError(stderr, "expected at most one type")
	}
	if len(args) == 0 {
		for _, k := range statute {
			fmt.Fprintln(stdout, k.name)
		}
		return 0
	}
	k, ok := findKind(args[0])
	if !ok {
		return unknownType(stderr, args[0])
	}
	fmt.Fprintln(stdout, sequential)
	if len(k.gates) == 0 {
		fmt.Fprintln(stdout, "no gates")
	}
	for _, g := range k.gates {
		fmt.Fprintf(stdout, "Section %s  %s\n", g.section, g.title)
	}
	return 0
}

func show(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || len(args) > 2 {
		return usageError(stderr, "expected a type and an optional section")
	}
	k, ok := findKind(args[0])
	if !ok {
		return unknownType(stderr, args[0])
	}
	if len(args) == 2 {
		g, ok := k.find(args[1])
		if !ok {
			return usageError(stderr, "unknown section %q in %s", args[1], k.name)
		}
		writeGate(stdout, g)
		return 0
	}
	fmt.Fprintln(stdout, sequential)
	if len(k.gates) == 0 {
		fmt.Fprintln(stdout, "no gates")
	}
	for _, g := range k.gates {
		fmt.Fprintln(stdout)
		writeGate(stdout, g)
	}
	return 0
}

func writeGate(w io.Writer, g gate) {
	fmt.Fprintf(w, "Section %s\n%s\n\n%s", g.section, g.title, g.body)
}

func unknownType(stderr io.Writer, name string) int {
	return usageError(stderr, "unknown type %q; the types are code and essays", name)
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
