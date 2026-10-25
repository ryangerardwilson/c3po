package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func keyNamesOf(b string) []string {
	var names []string
	for _, k := range ParseKeys([]byte(b)) {
		names = append(names, k.String())
	}
	return names
}

func TestParseKeysNamesBindings(t *testing.T) {
	cases := map[string][]string{
		"j":              {"j"},
		"\r":             {"enter"},
		"\x1b":           {"esc"},
		"\x7f":           {"backspace"},
		"\x08":           {"ctrl+h"},
		"\x0a":           {"ctrl+j"},
		"\x0b":           {"ctrl+k"},
		"\x03":           {"ctrl+c"},
		"\x1b[A\x1b[B":   {"up", "down"},
		"\x1bOC\x1b[D":   {"right", "left"},
		" ":              {" "},
		"ab c":           {"ab", " ", "c"},
		"é":              {"é"},
		"\x1bx":          {"alt+x"},
		"\x1b[3~":        {"delete"},
		"\x1b[200~":      {""},
		"?":              {"?"},
		"=\r":            {"=", "enter"},
		"\x1b[1;5A":      {""},
		"\x1b\x1b[A":     {"esc", "up"},
		"\x1b\x03":       {"alt+ctrl+c"},
		"\x1b[5~\x1b[6~": {"pgup", "pgdown"},
	}
	for input, want := range cases {
		got := keyNamesOf(input)
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("ParseKeys(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCtrlKeysCarryNoRunes(t *testing.T) {
	for _, k := range ParseKeys([]byte("\x0b\r\x7f\x1b[A")) {
		if len(k.Runes) != 0 {
			t.Fatalf("%s has runes %q", k.String(), k.Runes)
		}
	}
	k := ParseKeys([]byte("r"))[0]
	if k.Type != KeyRunes || string(k.Runes) != "r" {
		t.Fatalf("r = %#v", k)
	}
}

func TestBatchDropsNilAndUnwrapsSingle(t *testing.T) {
	if Batch(nil, nil) != nil {
		t.Fatal("empty batch is not nil")
	}
	one := Batch(nil, Quit)
	if _, ok := one().(QuitMsg); !ok {
		t.Fatal("single batch did not unwrap")
	}
	many := Batch(Quit, Quit)
	if msg, ok := many().(BatchMsg); !ok || len(msg) != 2 {
		t.Fatalf("batch = %#v", many())
	}
}

func TestTickFiresAfterDelay(t *testing.T) {
	start := time.Now()
	msg := Tick(5*time.Millisecond, func(at time.Time) Msg { return at })()
	if at, ok := msg.(time.Time); !ok || at.Sub(start) < 5*time.Millisecond {
		t.Fatalf("tick = %#v", msg)
	}
}

func TestRendererRewritesPreviousFrameInPlace(t *testing.T) {
	var out bytes.Buffer
	r := renderer{out: &out, width: 10, height: 5}
	r.paint("one\ntwo\nthree")
	first := out.String()
	if strings.Contains(first, "\x1b[2A") || !strings.Contains(first, "one\x1b[K\r\ntwo") {
		t.Fatalf("first frame = %q", first)
	}
	out.Reset()
	r.paint("one\ntwo\nthree")
	if out.Len() != 0 {
		t.Fatalf("unchanged frame repainted: %q", out.String())
	}
	r.paint("a")
	second := out.String()
	if !strings.HasPrefix(second, "\x1b[4A\r") || strings.Contains(second, "\x1b[J") || r.lines != 5 {
		t.Fatalf("second frame = %q lines = %d", second, r.lines)
	}
	out.Reset()
	r.stop()
	if out.String() != "\x1b[2K\r" {
		t.Fatalf("stop = %q", out.String())
	}
}

func TestRendererKeepsLastLinesThatFitHeight(t *testing.T) {
	var out bytes.Buffer
	r := renderer{out: &out, width: 20, height: 2}
	r.paint("a\nb\nc")
	if strings.Contains(out.String(), "a") || r.lines != 2 {
		t.Fatalf("frame = %q lines = %d", out.String(), r.lines)
	}
}

func TestFitLineTruncatesVisibleCellsOnly(t *testing.T) {
	got := fitLine("\x1b[1mhello\x1b[0m world", 4)
	if got != "\x1b[1mhell\x1b[0m" {
		t.Fatalf("fitLine = %q", got)
	}
	if got := fitLine("hi", 4); got != "hi\x1b[K" {
		t.Fatalf("short fitLine = %q", got)
	}
	if got := fitLine("long", 0); got != "long" {
		t.Fatalf("unknown width = %q", got)
	}
}

func TestCursorShapeIsASteadyBlock(t *testing.T) {
	if cursorShape(true) != "\x1b[2 q" {
		t.Fatalf("block %q", cursorShape(true))
	}
	if cursorShape(false) != "\x1b[0 q" {
		t.Fatalf("default %q", cursorShape(false))
	}
}
