package usage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

type tally struct {
	human   int
	agent   int
	service int
	ok      int
	last    int64
}

func (t *tally) n() int {
	if t == nil {
		return 0
	}
	return t.human + t.agent + t.service
}

// Report reads the stored counters and returns the usage text.
// The caller prints it before Note, so this invocation is not in the text.
func Report(prefix string) (string, error) {
	if skip() {
		return Text(nil, prefix), nil
	}
	rows, err := sqlxdb.ListUsage()
	local, localErr := localRows()
	if err != nil && localErr != nil {
		return "", err
	}
	if err != nil {
		rows = nil
	}
	if localErr == nil {
		rows = append(rows, local...)
	}
	return Text(rows, prefix), nil
}

// Text formats counters. Unused catalog features come first. A feature that
// was used and never exited 0 stays under used. A stored name that is not in
// the catalog is listed after used.
func Text(rows []sqlxdb.UsageRow, prefix string) string {
	totals := map[string]*tally{}
	for _, row := range rows {
		if !matchPrefix(row.Feature, prefix) {
			continue
		}
		t := totals[row.Feature]
		if t == nil {
			t = &tally{}
			totals[row.Feature] = t
		}
		switch row.Actor {
		case "human":
			t.human += row.N
		case "agent":
			t.agent += row.N
		default:
			t.service += row.N
		}
		t.ok += row.OK
		if row.Last > t.last {
			t.last = row.Last
		}
	}

	var unused, used, outside []string
	seen := map[string]bool{}
	for _, name := range Catalog {
		if !matchPrefix(name, prefix) {
			continue
		}
		seen[name] = true
		if totals[name].n() == 0 {
			unused = append(unused, name)
			continue
		}
		used = append(used, name)
	}
	for name, t := range totals {
		if seen[name] || t.n() == 0 {
			continue
		}
		outside = append(outside, name)
	}
	sort.Strings(unused)
	sort.Strings(used)
	sort.Strings(outside)

	width := len("feature")
	for _, name := range used {
		if len(name) > width {
			width = len(name)
		}
	}
	for _, name := range outside {
		if len(name) > width {
			width = len(name)
		}
	}

	var b strings.Builder
	b.WriteString("unused\n")
	for _, name := range unused {
		fmt.Fprintf(&b, "  %s\n", name)
	}
	b.WriteString("used\n")
	writeHead(&b, width)
	for _, name := range used {
		writeRow(&b, width, name, totals[name])
	}
	if len(outside) > 0 {
		b.WriteString("outside the catalog\n")
		writeHead(&b, width)
		for _, name := range outside {
			writeRow(&b, width, name, totals[name])
		}
	}
	return b.String()
}

func matchPrefix(feature, prefix string) bool {
	if prefix == "" {
		return true
	}
	return feature == prefix || strings.HasPrefix(feature, prefix+" ")
}

func writeHead(b *strings.Builder, width int) {
	fmt.Fprintf(b, "  %-*s  %5s  %5s  %7s  %4s  %s\n", width, "feature", "human", "agent", "service", "ok", "last")
}

func writeRow(b *strings.Builder, width int, name string, t *tally) {
	fmt.Fprintf(b, "  %-*s  %5d  %5d  %7d  %4d  %s\n", width, name, t.human, t.agent, t.service, t.ok, dayOf(t.last))
}

func dayOf(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).UTC().Format("2006-01-02")
}
