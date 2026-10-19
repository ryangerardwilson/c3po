package query

import "testing"

func TestTimeLimit(t *testing.T) {
	got, err := TimeLimit("2w")
	if err != nil || got != "newer_than:14d" {
		t.Fatalf("2w: %q %v", got, err)
	}
	got, err = TimeLimit("14d")
	if err != nil || got != "newer_than:14d" {
		t.Fatalf("14d: %q %v", got, err)
	}
	got, err = TimeLimit("2025-01-10..2025-01-20")
	if err != nil || got != "after:2025/01/09 before:2025/01/21" {
		t.Fatalf("range: %q %v", got, err)
	}
	_, err = TimeLimit("2025-02-01..2025-01-01")
	if err == nil || err.Error() != "ls -tl range end must be on or after start" {
		t.Fatalf("bad range: %v", err)
	}
	got, err = TimeLimit("jan 2025")
	if err != nil || got != "after:2024/12/31 before:2025/02/01" {
		t.Fatalf("named: %q %v", got, err)
	}
}
