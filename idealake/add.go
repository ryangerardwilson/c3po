package idealake

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var stdinTerminal = stdinIsTerminal

func (d *desk) add(args []string) int {
	if len(args) < 1 {
		return d.usage()
	}
	topic, err := cleanTopic(args[0])
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 2
	}
	raw, code := d.ideaArgs(args[1:])
	if code != 0 {
		return code
	}
	texts, err := cleanIdeas(raw)
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 2
	}
	existing, err := loadIdeas()
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	vectors, embedded := embedPassages(texts)
	ideas, err := assignIDs(existing, texts, topic, vectors)
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	if err := saveNew(ideas); err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	if !embedded {
		fmt.Fprintln(d.stderr, "idealake: stored without an embedding")
	}
	return d.printIdeas(ideas)
}

func (d *desk) ideaArgs(args []string) ([]string, int) {
	if len(args) > 0 {
		return args, 0
	}
	if stdinTerminal(d.stdin) {
		return nil, d.usage()
	}
	lines, err := readLines(d.stdin)
	if err != nil {
		fmt.Fprintf(d.stderr, "idealake: %s\n", err)
		return nil, 1
	}
	if len(lines) == 0 {
		fmt.Fprintln(d.stderr, "idealake: no idea")
		return nil, 2
	}
	return lines, 0
}

func readLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func stdinIsTerminal(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
