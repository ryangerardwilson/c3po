// Package gates prints the hardcoded gates an agent judges work against.
// It does not search code for violations, call a model, or decide pass or
// fail. tally reads only the sheet, its site paths, and git status.
package gates

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// judging is how gates are judged. It belongs in the help, not in probe output.
const judging = "Judge every provision. The first failure is the headline, and later fixes wait until that provision passes. Every body is still judged."

const frozen = "the gates change only when c3po itself is changed and reinstalled"

// Help is the text for c3po gates and c3po help -f gates. Its sheet example
// carries the real receipt of §8.3 for this build.
var Help = helpHead + exampleLine() + helpTail

const helpHead = `c3po gates
print the hardcoded gates so an agent can judge work against them

Each added argument deepens the same probe.

  c3po gates
    list the types
  c3po gates <type>
    list the statutes of one type
  c3po gates <type> <statute>
    list every provision of one statute in section order, one section
    and heading per line
  c3po gates <type> <statute> <section>...
    print the body of one provision, or of several sections in one
    pass
  c3po gates <type> <statute> all
    print every provision in section order as its section, body, and
    receipt
  c3po gates <type> <statute> tally <sheet>
    count a sheet of verdicts against this build and the git tree
  c3po gates help
    show this help
  c3po gates version
    print the c3po version

  c3po gates code architecture §1.1
  c3po gates code architecture 6 7 8.3
  c3po gates code architecture all
  c3po gates code architecture tally sheet.txt

types: code, essays
statutes: code has architecture; essays has none yet

A section is the section sign § and then dot-separated positive
integers, with no space: §1, §1.1, §2. Sections order numerically, and
a prefix comes first. Copy the section from the provision list as the
next argument. A section may be written §1.1, 1.1, or S.1.1; all three
name §1.1. Names are case-sensitive. A body is at most 300 characters.

Every printed body is followed by one receipt line:
  receipt <8 hex>
The receipt is the first 8 hex characters of SHA-256 over the c3po
version, the section, and the body. It changes when the build or the
text changes. Copy it into the sheet.

One section prints its body and its receipt. Several sections after
the statute print in the order written: each section on its own line,
then its body and receipt, with a blank line before the next section.
A section with subsections lists those subsections under the body, one
section and heading per line, one level deeper only, then counts the
subsection bodies this output does not show and gives the command that
prints them:
  4 bodies not shown: c3po gates code architecture 8.{1..4}
If any section is unknown, nothing is printed. all prints every
provision with no subsection list; all is not a section.

architecture holds three structural disciplines: data model, state
ownership, and boundaries. Mistakes there are paid for in every future
change, so architecture is stricter than style, framework choice, or
micro-optimization.

` + judging + `

The agent judges. c3po only prints the text. It does not search code
for violations, call a model, or decide pass or fail. Write a sheet
with one line per provision, in any order:
  §1.2 <receipt> <pass|fail> <site> [files <path>...]
`

const helpTail = `<site> is none when the search found nothing, or path:line, with the
path relative to the current directory. A fail may use none. files
names the changed files that fix the provision. Extra words are a bad
line.

tally prints how many provisions are judged, then, when they apply,
the missing sections and the command that fetches their bodies, the
fails that name no files (unfixed), and the changed files no line
claims (unclaimed). Changed files come from git status in the current
directory; untracked files are ignored. tally exits 0 when every
provision has one line, every receipt matches this build, every site
is none or a real path:line, and every changed file is claimed. In a
dirty tree an unfixed fail exits 2. Otherwise tally exits 2 and names
the bad line or path. tally counts the sheet; it does not judge it.

The text changes only when c3po itself is changed and reinstalled.
There is no way to edit a gate: no add, edit, or remove command.
There is no gates screen.
`

// exampleLine is a sheet line for §8.3 with its receipt in this build.
func exampleLine() string {
	for _, p := range architecture {
		if p.section == "§8.3" {
			return "  " + p.section + " " + receipt(p) + " fail store/order.go:41 files store/order.go\n"
		}
	}
	return ""
}

const usageLine = "usage: c3po gates [<type> [<statute> [<section>...|all|tally <sheet>]]]"

// Run serves c3po gates. args are the words after gates.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && helpWord(args[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(args) == 1 && versionWord(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if len(args) > 0 && commandWord(args[0]) {
		return usageError(stderr, "unknown command %q; %s", args[0], frozen)
	}
	return probe(args, stdout, stderr)
}

// probe prints one level deeper for each argument: types, statutes,
// provisions, then the bodies of the sections named. all and tally take
// the place of the sections.
func probe(args []string, stdout, stderr io.Writer) int {
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
	switch {
	case len(args) == 2:
		printHeadings(stdout, s.provisions)
		return 0
	case len(args) == 3 && args[2] == "all":
		printAll(stdout, s)
		return 0
	case args[2] == "tally":
		return runTally(args[3:], k, s, stdout, stderr)
	}
	return printSections(args[2:], k, s, stdout, stderr)
}

// printHeadings prints one line per provision: the section, two spaces,
// and the heading, so the section can be copied as the next argument.
func printHeadings(stdout io.Writer, ps []provision) {
	for _, p := range ps {
		fmt.Fprintf(stdout, "%s  %s\n", p.section, p.heading)
	}
}

func listStatutes(stdout io.Writer, k kind) {
	if len(k.statutes) == 0 {
		fmt.Fprintln(stdout, "no statutes")
	}
	for _, s := range k.statutes {
		fmt.Fprintln(stdout, s.name)
	}
}

func usageError(stderr io.Writer, format string, a ...any) int {
	fmt.Fprintf(stderr, "gates: "+format+"\n", a...)
	fmt.Fprintln(stderr, usageLine)
	return 2
}

// commandWord is a subcommand gates does not have. The old list and get
// probes and any way to rewrite the text are refused.
func commandWord(word string) bool {
	switch word {
	case "list", "get", "show", "edit", "add", "remove":
		return true
	}
	return false
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}
