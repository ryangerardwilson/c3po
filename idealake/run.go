// Package idealake stores short ideas by topic and searches them by meaning.
package idealake

import (
	"fmt"
	"io"
	"os"

	"github.com/ryangerardwilson/c3po/internal/version"
)

const (
	maxIdea         = 300
	maxTopic        = 80
	searchLimit     = 20
	embedModel      = "v1"
	passagePrefix   = "passage: "
	queryPrefix     = "query: "
	recordApp       = "idealake"
	recordKind      = "idea"
	recordKindTopic = "topic"
)

// Help is the text for c3po idealake and c3po help -f idealake.
const Help = `c3po idealake
keep short ideas, organized by topic

  c3po idealake add <topic> [<idea>...]
    store that topic, and store any ideas given with it
  c3po idealake list [topic]
    print ideas, or the ideas in one topic
  c3po idealake search <topic> <query>
    rank ideas in that topic by how close they are to the query
  c3po idealake contains <text> [topic]
    print ideas whose text contains that text, ignoring case
  c3po idealake show <id>
    print one idea
  c3po idealake remove <id>
    delete one idea
  c3po idealake topics
    list topics and how many ideas each has
  c3po idealake help
    show this help
  c3po idealake version
    print the c3po version

Pass each idea as its own argument. With no ideas, add stores the topic and nothing else. The numbers in a brainstorm are not ids.
An idea is one line of at most 300 characters. A topic is one line of at most
80 characters. Put --json first to print JSON for an agent.

  c3po idealake --json add xyz "a short idea"

Ideas are stored in the c3po sqlite file. nebula s3 backs that file up.
Search ranks ideas in one topic by meaning, using the xAI embeddings API,
model v1. An idea can rank high even when it does not contain the query.
contains is the lowercase text match and does not use embeddings.
Set XAI_API_KEY. The key is not printed. An idea is still stored when the
embedding cannot be made.
`

// Run serves c3po idealake. args are the words after idealake.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, os.Stdin)
}

func run(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	asJSON := false
	if len(args) > 0 && args[0] == "--json" {
		asJSON = true
		args = args[1:]
	}
	desk := &desk{stdout: stdout, stderr: stderr, stdin: stdin, json: asJSON}
	if len(args) == 0 {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if helpWord(args[0]) {
		if len(args) != 1 {
			return desk.usage()
		}
		fmt.Fprint(stdout, Help)
		return 0
	}
	if versionWord(args[0]) {
		if len(args) != 1 {
			return desk.usage()
		}
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	switch args[0] {
	case "add":
		return desk.add(args[1:])
	case "list":
		return desk.list(args[1:])
	case "search":
		return desk.search(args[1:])
	case "contains":
		return desk.contains(args[1:])
	case "show":
		return desk.show(args[1:])
	case "remove":
		return desk.remove(args[1:])
	case "topics":
		return desk.topics(args[1:])
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return desk.usage()
	}
}

type desk struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
	json   bool
}

func (d *desk) usage() int {
	fmt.Fprint(d.stderr, Help)
	return 2
}

func helpWord(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func versionWord(s string) bool {
	return s == "version" || s == "--version" || s == "-v"
}
