package style

import "testing"

func TestRenderUsesTrueColorWhenAdvertised(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-256color")
	got := NewStyle().Bold(true).Foreground(Color("#B4B400")).Background(Color("238")).Render("x")
	want := "\x1b[1;38;2;180;180;0;48;5;238mx\x1b[0m"
	if got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestRenderFallsBackTo256Colors(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	got := NewStyle().Foreground(Color("#FFFF00")).Render("x")
	if got != "\x1b[38;5;226mx\x1b[0m" {
		t.Fatalf("Render = %q", got)
	}
}

func TestNoColorKeepsBoldOnly(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := NewStyle().Foreground(Color("241")).Render("x"); got != "x" {
		t.Fatalf("Render = %q", got)
	}
	if got := NewStyle().Bold(true).Render("x"); got != "\x1b[1mx\x1b[0m" {
		t.Fatalf("bold Render = %q", got)
	}
}

func TestGettersReportNoColorWhenUnset(t *testing.T) {
	s := NewStyle()
	if s.GetForeground() != (NoColor{}) || s.GetBackground() != (NoColor{}) {
		t.Fatal("unset colors are not NoColor")
	}
	if s.Foreground(Color("80")).GetForeground() != Color("80") {
		t.Fatal("foreground not kept")
	}
	if NewStyle().Foreground(NoColor{}).Render("x") != "x" {
		t.Fatal("NoColor painted")
	}
}
