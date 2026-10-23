package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpAndVersion(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := NewRunner(out, errOut)
	if r.Run(nil) != 0 || !strings.Contains(out.String(), "xl open budget.xlsx") || strings.Contains(out.String(), "commands:") {
		t.Fatalf("bare help:\n%s", out.String())
	}
	out.Reset()
	if r.Run([]string{"help"}) != 0 || !strings.Contains(out.String(), "features:") {
		t.Fatalf("help:\n%s", out.String())
	}
	out.Reset()
	if r.Run([]string{"version"}) != 0 || strings.TrimSpace(out.String()) != "0.1.0" {
		t.Fatalf("version %q", out.String())
	}
}

func TestUnknownAndOpenNeedATerminal(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := NewRunner(out, errOut)
	if r.Run([]string{"nope", "extra"}) != 1 || !strings.Contains(errOut.String(), "xl help") {
		t.Fatalf("stderr %q", errOut.String())
	}
	errOut.Reset()
	if r.Run([]string{"budget"}) != 1 || !strings.Contains(errOut.String(), "terminal") {
		t.Fatalf("file stderr %q", errOut.String())
	}
	errOut.Reset()
	if r.Run([]string{"budget.xlsx"}) != 1 || !strings.Contains(errOut.String(), "terminal") {
		t.Fatalf("xlsx stderr %q", errOut.String())
	}
	errOut.Reset()
	if r.Run([]string{"open", "budget.csv"}) != 1 || !strings.Contains(errOut.String(), "terminal") {
		t.Fatalf("open stderr %q", errOut.String())
	}
	errOut.Reset()
	if r.Run([]string{"open", "a", "b"}) != 1 {
		t.Fatal("open accepted two paths")
	}
}
