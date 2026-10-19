package vault

import (
	"testing"

	"github.com/ryangerardwilson/c3po/pass/internal/op"
)

func TestSelectDedupesAndRejectsAmbiguousNames(t *testing.T) {
	items := []op.Item{{ID: "a", Title: "Key"}, {ID: "b", Title: "KEY_TWO"}, {ID: "c", Title: "dup"}, {ID: "d", Title: "Dup"}}
	chosen, err := Select(items, []string{"key", "KEY*"})
	if err != nil || len(chosen) != 2 {
		t.Fatalf("chosen = %#v err = %v", chosen, err)
	}
	if _, err := Select(items, []string{"dup"}); err == nil || err.Error() != "More than one item named dup." {
		t.Fatalf("err = %v", err)
	}
	if _, err := Select(items, []string{" "}); err == nil || err.Error() != "Item name must not be empty." {
		t.Fatalf("err = %v", err)
	}
}

func TestLinesKeepUntitledAndSortWithoutCase(t *testing.T) {
	items := []op.Item{{ID: "1", Title: "b"}, {ID: "1", Title: "b"}, {ID: "2"}, {ID: "3", Title: "A"}}
	got := Lines(items)
	want := []string{"(untitled)", "A", "b"}
	if len(got) != len(want) {
		t.Fatalf("lines = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("lines = %q", got)
		}
	}
}
