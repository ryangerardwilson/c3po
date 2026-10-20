package gdrive

import (
	"strings"
	"testing"
)

func TestHelpPointsAtTheScreen(t *testing.T) {
	if !strings.Contains(Help, "c3po tui gdrive") {
		t.Fatal(Help)
	}
	if strings.Contains(Help, "<preset> browse") || strings.Contains(Help, "<preset> upload") {
		t.Fatal(Help)
	}
	if !strings.Contains(ScreenHelp, "preset") || !strings.Contains(ScreenHelp, "browse") || strings.Contains(ScreenHelp, "upload") {
		t.Fatal(ScreenHelp)
	}
}
