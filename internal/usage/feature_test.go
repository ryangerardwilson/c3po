package usage

import (
	"strings"
	"testing"
)

func TestFeatureNames(t *testing.T) {
	cases := []struct {
		argv []string
		want string
	}{
		{nil, "c3po"},
		{[]string{"c3po"}, "c3po"},
		{[]string{"c3po", "help"}, "help"},
		{[]string{"c3po", "help", "-f", "ast", "patch"}, "help"},
		{[]string{"c3po", "-h"}, "help"},
		{[]string{"c3po", "--help"}, "help"},
		{[]string{"c3po", "-h", "extra"}, "unknown"},
		{[]string{"c3po", "version"}, "version"},
		{[]string{"c3po", "--version"}, "version"},
		{[]string{"c3po", "-version"}, "version"},
		{[]string{"c3po", "usage"}, "usage"},
		{[]string{"c3po", "usage", "ast"}, "usage"},
		{[]string{"c3po", "nope"}, "unknown"},
		{[]string{"c3po", "complexity"}, "unknown"},
		{[]string{"c3po", "nlp"}, "unknown"},
		{[]string{"/usr/bin/c3po", "ast", "patch", "file.go"}, "ast patch"},
		{[]string{"rgw-ast", "patch"}, "ast patch"},
		{[]string{"/home/ryan/.local/bin/rgw-ast", "help"}, "help"},
		{[]string{"c3po", "ast"}, "help"},
		{[]string{"c3po", "ast", "help", "patch"}, "help"},
		{[]string{"c3po", "ast", "patch", "--help"}, "help"},
		{[]string{"c3po", "ast", "--root", "/tmp/x", "status"}, "ast status"},
		{[]string{"c3po", "ast", "--root=/tmp/x", "--refresh", "measure"}, "ast measure"},
		{[]string{"c3po", "ast", "search", "--help"}, "ast search"},
		{[]string{"c3po", "ast", "nope"}, "unknown"},
		{[]string{"c3po", "ast", "--root"}, "unknown"},
		{[]string{"c3po", "ast", "cut"}, "ast cut"},
		{[]string{"c3po", "ast", "agents-block"}, "ast agents-block"},
		{[]string{"c3po", "ast", "hook", "lots", "of", "tool", "json"}, "ast hook"},
		{[]string{"c3po", "tui"}, "tui"},
		{[]string{"c3po", "tui", "help"}, "help"},
		{[]string{"c3po", "tui", "o"}, "tui o"},
		{[]string{"c3po", "tui", "o", "help"}, "help"},
		{[]string{"c3po", "tui", "o", "--resume", "abc"}, "tui o"},
		{[]string{"c3po", "tui", "o", "--resume=abc"}, "tui o"},
		{[]string{"c3po", "tui", "nlp"}, "unknown"},
		{[]string{"c3po", "tui", "cyclotree", "main"}, "tui cyclotree"},
		{[]string{"c3po", "deport"}, "deport"},
		{[]string{"c3po", "deport", "list"}, "deport list"},
		{[]string{"c3po", "deport", "8080"}, "deport"},
		{[]string{"c3po", "deport", "help"}, "help"},
		{[]string{"c3po", "tunnel"}, "help"},
		{[]string{"c3po", "tunnel", "help"}, "help"},
		{[]string{"c3po", "tunnel", "add", "lab", "on", "9090"}, "tunnel add"},
		{[]string{"c3po", "tunnel", "open"}, "unknown"},
		{[]string{"c3po", "tunnel", "run"}, "tunnel run"},
		{[]string{"c3po", "gmail"}, "help"},
		{[]string{"c3po", "gmail", "1", "send", "to", "a@b.c"}, "gmail send"},
		{[]string{"c3po", "gmail", "spam", "clean"}, "gmail spam clean"},
		{[]string{"c3po", "gmail", "tidy", "audit"}, "gmail tidy audit"},
		{[]string{"c3po", "gmail", "tidy"}, "gmail tidy"},
		{[]string{"c3po", "gmail", "delete", "unread"}, "gmail delete unread"},
		{[]string{"c3po", "gmail", "2", "inspect-spam"}, "gmail inspect-spam"},
		{[]string{"c3po", "gmail", "timer", "install"}, "gmail timer install"},
		{[]string{"c3po", "gmail", "timer"}, "gmail timer"},
		{[]string{"c3po", "gmail", "conf"}, "gmail"},
		{[]string{"c3po", "gcal"}, "help"},
		{[]string{"c3po", "gcal", "1", "list", "limit", "5"}, "gcal list"},
		{[]string{"c3po", "gcal", "accounts", "list"}, "gcal accounts list"},
		{[]string{"c3po", "gcal", "config"}, "gcal config"},
		{[]string{"c3po", "gdrive"}, "help"},
		{[]string{"c3po", "gdrive", "1", "list", "registrations"}, "gdrive list registrations"},
		{[]string{"c3po", "gdrive", "1", "restore", "registrations"}, "gdrive restore registrations"},
		{[]string{"c3po", "gdrive", "1", "restore", "registration", "9"}, "gdrive restore registration"},
		{[]string{"c3po", "gdrive", "auth", "super-secret-path"}, "gdrive auth"},
		{[]string{"c3po", "gdrive", "sync", "run"}, "gdrive sync run"},
		{[]string{"c3po", "looptab"}, "looptab"},
		{[]string{"c3po", "looptab", "edit", "jobs"}, "looptab edit"},
		{[]string{"c3po", "pass"}, "help"},
		{[]string{"c3po", "pass", "Vault", "--delete", "Title"}, "pass delete"},
		{[]string{"c3po", "pass", "Vault", "item"}, "pass"},
		{[]string{"c3po", "pass", "Vault", "-l"}, "pass"},
		{[]string{"c3po", "pass", "upgrade"}, "pass upgrade"},
		{[]string{"c3po", "slack"}, "help"},
		{[]string{"c3po", "slack", "1", "send", "hello secret"}, "slack send"},
		{[]string{"c3po", "slack", "mark", "all", "read"}, "slack mark"},
		{[]string{"c3po", "slack", "1", "mark", "all", "read"}, "slack mark"},
		{[]string{"c3po", "slack", "accounts", "list"}, "slack accounts list"},
		{[]string{"deport", "list"}, "unknown"},
	}
	for _, tc := range cases {
		got := Feature(tc.argv)
		if got != tc.want {
			t.Fatalf("Feature(%q) = %q, want %q", tc.argv, got, tc.want)
		}
		if got != "unknown" && !Known(got) {
			t.Fatalf("Feature(%q) = %q is outside the catalog", tc.argv, got)
		}
		for _, arg := range tc.argv {
			if strings.Contains(arg, "secret") && strings.Contains(got, "secret") {
				t.Fatalf("feature %q kept an argument from %q", got, tc.argv)
			}
		}
	}
}

func TestCatalogKeysStayPlain(t *testing.T) {
	if len(Catalog) < 100 {
		t.Fatalf("catalog has %d features", len(Catalog))
	}
	seen := map[string]bool{}
	for _, name := range Catalog {
		if seen[name] {
			t.Fatalf("duplicate %s", name)
		}
		seen[name] = true
		if name == "unknown" || strings.ContainsAny(name, "/\\@") {
			t.Fatalf("unstable catalog key %q", name)
		}
		for _, field := range strings.Fields(name) {
			if digits(field) {
				t.Fatalf("catalog key %q contains a preset or port", name)
			}
		}
	}
	if Known("unknown") {
		t.Fatal("unknown is a catalog feature")
	}
	for _, name := range []string{"ast cut", "ast patch", "pass upgrade", "tunnel logs", "gmail tidy audit", "help", "usage"} {
		if !Known(name) {
			t.Fatalf("missing %s", name)
		}
	}
}
