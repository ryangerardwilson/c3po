package idealake

import (
	"fmt"
	"sort"
	"strings"
)

func (d *desk) list(args []string) int {
	if len(args) > 1 {
		return d.usage()
	}
	topic, code := d.optionalTopic(args)
	if code != 0 {
		return code
	}
	ideas, code := d.load()
	if code != 0 {
		return code
	}
	return d.printIdeas(filterTopic(ideas, topic))
}

func (d *desk) show(args []string) int {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return d.usage()
	}
	ideas, code := d.load()
	if code != 0 {
		return code
	}
	idea, ok := findIdea(ideas, args[0])
	if !ok {
		fmt.Fprintf(d.stderr, "idealake: no idea %s\n", args[0])
		return 1
	}
	if d.json {
		return d.emit(viewIdea(idea))
	}
	fmt.Fprintf(d.stdout, "%s  %s  %s\n%s\n", idea.ID, idea.Topic, idea.Created, idea.Text)
	return 0
}

func (d *desk) remove(args []string) int {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return d.usage()
	}
	ideas, code := d.load()
	if code != 0 {
		return code
	}
	if _, ok := findIdea(ideas, args[0]); !ok {
		fmt.Fprintf(d.stderr, "idealake: no idea %s\n", args[0])
		return 1
	}
	if err := deleteIdea(args[0]); err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 1
	}
	if d.json {
		return d.emit(removedDoc{Removed: args[0]})
	}
	fmt.Fprintf(d.stdout, "removed %s\n", args[0])
	return 0
}

func (d *desk) topics(args []string) int {
	if len(args) != 0 {
		return d.usage()
	}
	ideas, code := d.load()
	if code != 0 {
		return code
	}
	return d.printTopics(topicCounts(ideas))
}

func (d *desk) search(args []string) int {
	if len(args) != 2 {
		return d.usage()
	}
	topic, err := cleanTopic(args[0])
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return 2
	}
	query := strings.TrimSpace(args[1])
	if query == "" {
		fmt.Fprintln(d.stderr, "idealake: query is empty")
		return 2
	}
	ideas, code := d.load()
	if code != 0 {
		return code
	}
	ideas = filterTopic(ideas, topic)
	missing := unembedded(ideas)
	if len(ideas) == 0 || missing == len(ideas) {
		if missing > 0 {
			fmt.Fprint(d.stderr, missingNote(missing))
		}
		return d.printHits(nil)
	}
	vectors, ok := embedQuery(query)
	if !ok {
		fmt.Fprintln(d.stderr, "idealake: embedding unavailable")
		return 1
	}
	if missing > 0 {
		fmt.Fprint(d.stderr, missingNote(missing))
	}
	return d.printHits(rankIdeas(ideas, vectors))
}

func (d *desk) contains(args []string) int {
	if len(args) < 1 || len(args) > 2 {
		return d.usage()
	}
	query := strings.TrimSpace(args[0])
	if query == "" {
		fmt.Fprintln(d.stderr, "idealake: query is empty")
		return 2
	}
	topic, code := d.optionalTopic(args[1:])
	if code != 0 {
		return code
	}
	ideas, code := d.load()
	if code != 0 {
		return code
	}
	return d.printIdeas(containsIdeas(filterTopic(ideas, topic), query))
}

func (d *desk) optionalTopic(args []string) (string, int) {
	if len(args) == 0 {
		return "", 0
	}
	topic, err := cleanTopic(args[0])
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return "", 2
	}
	return topic, 0
}

func (d *desk) load() ([]Idea, int) {
	ideas, err := loadIdeas()
	if err != nil {
		fmt.Fprintln(d.stderr, err.Error())
		return nil, 1
	}
	return ideas, 0
}

func missingNote(n int) string {
	if n == 1 {
		return "idealake: 1 idea has no embedding\n"
	}
	return fmt.Sprintf("idealake: %d ideas have no embedding\n", n)
}

func topicCounts(ideas []Idea) []topicCount {
	counts := map[string]int{}
	var names []string
	for _, idea := range ideas {
		if _, ok := counts[idea.Topic]; !ok {
			names = append(names, idea.Topic)
		}
		counts[idea.Topic]++
	}
	sort.Strings(names)
	out := make([]topicCount, 0, len(names))
	for _, name := range names {
		out = append(out, topicCount{Topic: name, N: counts[name]})
	}
	return out
}
