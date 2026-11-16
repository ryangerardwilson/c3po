package tunnel

import (
	"bytes"
	"strings"
	"testing"
)

func TestScreenHelpNamesTheC3POEntry(t *testing.T) {
	if !strings.Contains(ScreenHelp, "r2d2 tunnel") || strings.Contains(ScreenHelp, "c3po tunnel open") {
		t.Fatal(ScreenHelp)
	}
}

func TestOpenIsNotACommand(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"open"}, &out, &errb)
	text := out.String() + errb.String()
	if code != 2 || strings.Contains(text, "c3po tunnel open") || !strings.Contains(text, "nebula tunnel screen") {
		t.Fatalf("open %d\n%s", code, text)
	}
}
