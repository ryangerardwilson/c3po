package o

import (
	"strings"
	"testing"
)

func TestHelpNamesTab(t *testing.T) {
	if !strings.Contains(Help, "  tab              complete the :! word") {
		t.Fatal("help missing tab")
	}
}
