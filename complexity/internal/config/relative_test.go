package config

import "testing"

func TestRelativeToDir(t *testing.T) {
	if got := Relative("/repo/pkg", "/repo/pkg/hot.go"); got != "hot.go" {
		t.Fatalf("file %s", got)
	}
	if got := Relative("/repo/pkg", "hot.go"); got != "hot.go" {
		t.Fatalf("already %s", got)
	}
	if got := Relative("/repo/pkg", "/repo"); got != ".." {
		t.Fatalf("parent %s", got)
	}
	if got := Relative("/repo", "/repo"); got != "." {
		t.Fatalf("same %s", got)
	}
	if got := Relative("/repo", "/repo/.config/complexity/settings.json"); got != ".config/complexity/settings.json" {
		t.Fatalf("config %s", got)
	}
}
