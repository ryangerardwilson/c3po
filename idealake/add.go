package idealake

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/ryangerardwilson/c3po/idealake/internal/lake"
)

func (d *desk) add(args []string) int {
	if len(args) < 1 {
		return d.usage()
	}
	topic, err := lake.CleanTopic(args[0])
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 2
	}
	raw, code := d.ideaArgs(args[1:])
	if code != 0 {
		return code
	}
	texts, err := lake.CleanIdeas(raw)
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 2
	}
	existing, err := d.store.Ideas()
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	if len(texts) == 0 {
		return d.addTopic(topic, existing)
	}
	vectors, embedded := lake.EmbedPassages(d.embed, texts)
	last, err := d.store.LastIssued()
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	ideas := lake.AssignIDs(last, existing, texts, topic, vectors, d.now())
	if err := d.store.SaveNew(ideas); err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	if !embedded {
		fmt.Fprintln(d.stderr, "idealake: stored without an embedding")
	}
	return d.printIdeas(ideas)
}

func (d *desk) addTopic(topic string, existing []lake.Idea) int {
	if err := d.store.EnsureTopic(topic); err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	return d.printTopics([]lake.TopicCount{{Topic: topic, N: lake.CountTopic(existing, topic)}})
}

func (d *desk) ideaArgs(args []string) ([]string, int) {
	if len(args) > 0 {
		return args, 0
	}
	if d.terminal(d.stdin) {
		return nil, 0
	}
	lines, err := readLines(d.stdin)
	if err != nil {
		fmt.Fprintf(d.stderr, "idealake: %s\n", err)
		return nil, 1
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
