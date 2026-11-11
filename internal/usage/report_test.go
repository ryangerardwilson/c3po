package usage

import (
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

func TestTextLeadsWithUnused(t *testing.T) {
	text := Text(nil, "")
	if !strings.HasPrefix(text, "unused\n") {
		t.Fatalf("prefix %q", text[:20])
	}
	usedAt := strings.Index(text, "\nused\n")
	cutAt := strings.Index(text, "\n  ast cut\n")
	if usedAt < 0 || cutAt < 0 || cutAt > usedAt {
		t.Fatalf("ast cut not in unused\n%s", text)
	}
	if strings.Contains(text, "outside the catalog") {
		t.Fatal("empty store listed an outside feature")
	}
	if strings.Contains(text[:usedAt], "\n  unknown\n") {
		t.Fatal("unknown is listed as unused")
	}
}

func TestTextSplitsActorsAndKeepsFailuresUsed(t *testing.T) {
	last := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC).Unix()
	earlier := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC).Unix()
	text := Text([]sqlxdb.UsageRow{
		{Feature: "ast patch", Actor: "agent", N: 3, OK: 3, Last: last},
		{Feature: "ast patch", Actor: "human", N: 1, OK: 0, Last: earlier},
		{Feature: "ast move", Actor: "agent", N: 2, OK: 0, Last: last},
		{Feature: "made up", Actor: "service", N: 4, OK: 1, Last: last},
		{Feature: "unknown", Actor: "agent", N: 1, OK: 0, Last: last},
	}, "")
	usedAt := strings.Index(text, "\nused\n")
	outsideAt := strings.Index(text, "\noutside the catalog\n")
	if usedAt < 0 || outsideAt < 0 || outsideAt < usedAt {
		t.Fatalf("sections\n%s", text)
	}
	unused := text[:usedAt]
	used := text[usedAt:outsideAt]
	outside := text[outsideAt:]
	if strings.Contains(unused, "\n  ast patch\n") || strings.Contains(unused, "\n  ast move\n") {
		t.Fatalf("used features listed as unused\n%s", unused)
	}
	if strings.Contains(unused, "\n  ast cut\n") == false {
		t.Fatal("ast cut left the unused list")
	}
	patch := lineWith(used, "ast patch")
	if got, want := tail(patch), []string{"1", "3", "0", "3", "2026-11-11"}; !same(got, want) {
		t.Fatalf("ast patch %q", patch)
	}
	move := lineWith(used, "ast move")
	if got, want := tail(move), []string{"0", "2", "0", "0", "2026-11-11"}; !same(got, want) {
		t.Fatalf("ast move %q", move)
	}
	if strings.Contains(used, "made up") || strings.Contains(used, "unknown") {
		t.Fatalf("outside names under used\n%s", used)
	}
	if !strings.Contains(outside, "made up") || !strings.Contains(outside, "unknown") {
		t.Fatalf("outside\n%s", outside)
	}
}

func TestTextPrefix(t *testing.T) {
	text := Text(nil, "ast")
	if !strings.Contains(text, "\n  ast cut\n") {
		t.Fatal("missing ast cut")
	}
	if strings.Contains(text, "\n  gmail") || strings.Contains(text, "\n  deport") || strings.Contains(text, "\n  tunnel") {
		t.Fatalf("prefix leaked\n%s", text)
	}
}

func lineWith(section, name string) string {
	for _, line := range strings.Split(section, "\n") {
		if strings.Contains(line, name) {
			return line
		}
	}
	return ""
}

func tail(line string) []string {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return fields
	}
	return fields[len(fields)-5:]
}

func same(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
