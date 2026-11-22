package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/version"
)

func TestHelpAndParseContract(t *testing.T) {
	var stdout bytes.Buffer
	rt := NewRuntime()
	rt.Stdout = &stdout
	rt.Stderr = &bytes.Buffer{}
	if err := rt.Run(nil); err != nil {
		t.Fatalf("Run help: %v", err)
	}
	for _, want := range []string{
		"c3po slack <preset> list channels",
		"c3po slack <preset> list dms",
		"c3po slack accounts list",
		"c3po slack 1 inspect message",
		"c3po slack 1 preview send",
		"c3po slack <preset> delete message",
		"c3po slack <preset> edit message",
		"c3po slack <preset> reply to <message_id>",
		"c3po slack <preset> thread <message_id>",
		"c3po slack config | c3po slack config edit",
		"c3po slack mark all read",
		"c3po slack <preset> unread count",
		"output json",
		"--json",
		"in <channel>",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %q\nhelp:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "sluser") {
		t.Fatal("help still names sluser")
	}

	globalMark, err := parseArgs([]string{"mark", "all", "read"})
	if err != nil {
		t.Fatalf("parse global mark: %v", err)
	}
	if globalMark.Command != "mra" || globalMark.Preset != "" {
		t.Fatalf("unexpected global mark parse: %+v", globalMark)
	}
	presetMark, err := parseArgs([]string{"2", "mark", "all", "read"})
	if err != nil {
		t.Fatalf("parse preset mark: %v", err)
	}
	if presetMark.Command != "mra" || presetMark.Preset != "2" {
		t.Fatalf("unexpected preset mark parse: %+v", presetMark)
	}
	if _, err := parseArgs([]string{"mark", "read"}); err == nil {
		t.Fatalf("expected malformed mark to fail")
	}
	unreadArgs, err := parseArgs([]string{"1", "unread", "count", "output", "json"})
	if err != nil || unreadArgs.Command != "unread-count" || unreadArgs.Preset != "1" || !unreadArgs.OutputJSON {
		t.Fatalf("unexpected unread count parse: %+v err=%v", unreadArgs, err)
	}
	if _, err := parseArgs([]string{"1", "unread"}); err == nil {
		t.Fatalf("expected unread without count to fail")
	}
	for _, argv := range [][]string{
		{"1", "open", "tui"},
		{"1", "tui"},
		{"1", "ls"},
		{"1", "mra"},
		{"1", "cfg"},
	} {
		_, err := parseArgs(argv)
		if err == nil {
			t.Fatalf("expected unknown command to fail: %v", argv)
		}
		if strings.Contains(err.Error(), "Retired command") {
			t.Fatalf("unknown command %v must not print Retired command: %v", argv, err)
		}
	}

	helpParseListAndEvents(t)
}

func helpParseListAndEvents(t *testing.T) {
	channelsArgs, err := parseArgs([]string{"2", "list", "channels"})
	if err != nil {
		t.Fatalf("parse list channels: %v", err)
	}
	if channelsArgs.Command != "list-channels" || channelsArgs.Preset != "2" {
		t.Fatalf("unexpected list channels parse: %+v", channelsArgs)
	}
	dmsJSON, err := parseArgs([]string{"2", "list", "dms", "output", "json"})
	if err != nil || dmsJSON.Command != "list-dms" || !dmsJSON.OutputJSON {
		t.Fatalf("unexpected list dms json parse: %+v err=%v", dmsJSON, err)
	}
	contactsArgs, err := parseArgs([]string{"1", "list", "contacts"})
	if err != nil || contactsArgs.Command != "list-contacts" {
		t.Fatalf("unexpected list contacts parse: %+v err=%v", contactsArgs, err)
	}
	if _, err := parseArgs([]string{"2", "conversations", "list"}); err == nil {
		t.Fatalf("expected conversations list to be rejected")
	}
	if _, err := parseArgs([]string{"2", "contacts", "list"}); err == nil {
		t.Fatalf("expected contacts list to be rejected")
	}
	cleanArgs, err := parseArgs([]string{"2", "conversations", "clean"})
	if err != nil || cleanArgs.Command != "sc" {
		t.Fatalf("unexpected conversations clean parse: %+v err=%v", cleanArgs, err)
	}

	eventsSync, err := parseArgs([]string{"2", "events", "sync"})
	if err != nil || eventsSync.Command != "events" || eventsSync.EventsAction != "sync" {
		t.Fatalf("unexpected events sync parse: %+v err=%v", eventsSync, err)
	}
	for _, argv := range [][]string{
		{"2", "events", "once"},
		{"2", "events", "service"},
		{"2", "events", "logs"},
		{"2", "events", "timer", "install"},
		{"2", "events", "timer", "disable"},
	} {
		_, err := parseArgs(argv)
		if err == nil {
			t.Fatalf("expected removed event command to fail: %v", argv)
		}
		if strings.Contains(err.Error(), "Retired command") {
			t.Fatalf("removed event command %v must not print Retired command: %v", argv, err)
		}
	}
}

func TestUpgradeIsRejected(t *testing.T) {
	rt := NewRuntime()
	rt.Stdout = &bytes.Buffer{}
	rt.Stderr = &bytes.Buffer{}
	err := rt.Run([]string{"upgrade"})
	if err == nil {
		t.Fatal("upgrade should be rejected")
	}
	if _, ok := err.(UsageError); !ok {
		t.Fatalf("upgrade error = %T %v", err, err)
	}
}

func TestChannelNameQuery(t *testing.T) {
	if got := channelNameQuery("#blog"); got != "blog" {
		t.Fatalf("channelNameQuery(#blog)=%q", got)
	}
	if got := channelNameQuery("C0123AB"); got != "" {
		t.Fatalf("channelNameQuery(channel id)=%q", got)
	}
}

func TestAuthStoresTokensInsidePreset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	var stdout bytes.Buffer
	rt := NewRuntime()
	rt.Stdout = &stdout
	rt.Stderr = &bytes.Buffer{}

	err := rt.Run([]string{"auth", "2", "user", "xoxp-user", "name", "work"})
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(home, "config", "c3po", "slack", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	account := accounts(cfg)["2"]
	tokens := tokenMap(account)
	if tokens["user"] != "xoxp-user" {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
	if _, ok := tokens["bot"]; ok {
		t.Fatalf("bot token must not be stored: %#v", tokens)
	}
	rawToken := asMap(account["token"])
	if _, ok := rawToken["bot"]; ok {
		t.Fatalf("token.bot must not be written: %#v", rawToken)
	}
	if account["name"] != "work" {
		t.Fatalf("unexpected account name: %#v", account["name"])
	}
}

func TestAuthRejectsBotKeyword(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	rt := NewRuntime()
	rt.Stdout = &bytes.Buffer{}
	rt.Stderr = &bytes.Buffer{}
	err := rt.Run([]string{"auth", "2", "user", "xoxp-user", "bot", "xoxb-bot", "name", "work"})
	if err == nil {
		t.Fatal("expected bot keyword to fail")
	}
	if !strings.Contains(err.Error(), "does not accept bot tokens") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, "config", "c3po", "slack", "config.json")); !os.IsNotExist(statErr) {
		t.Fatalf("config should not be written on rejected auth: %v", statErr)
	}
}

func TestSenderFilterResolvesSavedContact(t *testing.T) {
	contacts := Contacts{"mike": "mike@willbanks.dev"}
	terms := senderFilterTerms(contacts, "mike")
	if len(terms) < 2 {
		t.Fatalf("expected multiple sender terms, got %#v", terms)
	}
	if resolveSenderSearchTerm(contacts, "mike") != "mike@willbanks.dev" {
		t.Fatalf("unexpected search term: %s", resolveSenderSearchTerm(contacts, "mike"))
	}
	entry := MessageEntry{
		Sender:  map[string]any{"name": "Mike Willbanks", "email": "mike@willbanks.dev", "label": "Mike Willbanks", "id": "U2"},
		Message: map[string]any{"text": "works for me"},
	}
	if !entryPassesFilters(entry, contacts, "", "mike", "", 0, false) {
		t.Fatal("expected contact label to match sender")
	}
}

func TestDeleteEditParseContract(t *testing.T) {
	deleteArgs, err := parseArgs([]string{"2", "delete", "message", "C0B7V41SRST:1781778512.813869"})
	if err != nil || deleteArgs.Command != "delete" || deleteArgs.Recipient != "C0B7V41SRST:1781778512.813869" {
		t.Fatalf("unexpected delete parse: %+v err=%v", deleteArgs, err)
	}
	editArgs, err := parseArgs([]string{"2", "preview", "edit", "message", "C0B7V41SRST:1781778811.092529", "body", "updated"})
	if err != nil || editArgs.Command != "preview-edit" || editArgs.Message != "updated" {
		t.Fatalf("unexpected preview edit parse: %+v err=%v", editArgs, err)
	}
	if _, err := parseArgs([]string{"2", "delete", "message", "bad-id"}); err == nil {
		t.Fatal("expected invalid message id to fail")
	}
}

func TestReplyParseIsExecutable(t *testing.T) {
	// Regression: reply was incorrectly treated as a retired alias, so
	// preview reply worked while real reply was rejected.
	replyArgs, err := parseArgs([]string{"1", "reply", "to", "C123ABC:1712764800.000100", "body", "Example"})
	if err != nil {
		t.Fatalf("parse reply: %v", err)
	}
	if replyArgs.Command != "reply" || replyArgs.Recipient != "C123ABC:1712764800.000100" || replyArgs.Message != "Example" {
		t.Fatalf("unexpected reply parse: %+v", replyArgs)
	}
	previewArgs, err := parseArgs([]string{"1", "preview", "reply", "to", "C123ABC:1712764800.000100", "body", "Example"})
	if err != nil || previewArgs.Command != "preview-reply" {
		t.Fatalf("unexpected preview reply parse: %+v err=%v", previewArgs, err)
	}
	if _, err := parseArgs([]string{"1", "reply", "to", "not-a-message", "body", "x"}); err == nil {
		t.Fatal("expected invalid reply target to fail")
	}
}

func TestListChannelAndSinceParse(t *testing.T) {
	args, err := parseArgs([]string{"1", "list", "in", "#genie", "from", "ryan", "since", "4h", "limit", "20", "output", "json"})
	if err != nil {
		t.Fatalf("parse list in/since: %v", err)
	}
	if args.Command != "ls" || args.ListIn != "#genie" || args.ListFrom != "ryan" || args.ListTimeLimit != "4h" || args.ListLimit != 20 || !args.OutputJSON {
		t.Fatalf("unexpected list parse: %+v", args)
	}
	if _, err := parseArgs([]string{"1", "list", "since", "not-a-window"}); err == nil {
		t.Fatal("expected invalid since window to fail")
	}
	threadArgs, err := parseArgs([]string{"1", "thread", "C123ABC:1712764800.000100", "limit", "25", "output", "json"})
	if err != nil || threadArgs.Command != "thread" || threadArgs.ListLimit != 25 || !threadArgs.OutputJSON {
		t.Fatalf("unexpected thread parse: %+v err=%v", threadArgs, err)
	}
	configArgs, err := parseArgs([]string{"config"})
	if err != nil || configArgs.Command != "config" {
		t.Fatalf("unexpected config parse: %+v err=%v", configArgs, err)
	}
	editArgs, err := parseArgs([]string{"config", "edit"})
	if err != nil || editArgs.Command != "config-edit" {
		t.Fatalf("unexpected config edit parse: %+v err=%v", editArgs, err)
	}
}

func TestStartTSSupportsHours(t *testing.T) {
	cutoff, ok := startTS("4h", time.Now())
	if !ok {
		t.Fatal("expected 4h to parse")
	}
	now := float64(time.Now().Unix())
	if cutoff < now-4*3600-5 || cutoff > now-4*3600+5 {
		t.Fatalf("unexpected 4h cutoff: %v (now=%v)", cutoff, now)
	}
	if _, ok := startTS("0h", time.Now()); ok {
		t.Fatal("expected 0h to be invalid")
	}
	if _, ok := startTS("banana", time.Now()); ok {
		t.Fatal("expected banana to be invalid")
	}
}

func TestConfigDefaultsToRedactedSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"name":"work","token":{"user":"xoxp-super-secret-token","bot":"xoxb-bot-secret"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	rt := NewRuntime()
	rt.Stdout = &stdout
	rt.Stderr = &bytes.Buffer{}
	if err := rt.Run([]string{"config"}); err != nil {
		t.Fatalf("config: %v", err)
	}
	out := stdout.String()
	if strings.Contains(out, "xoxp-super-secret-token") || strings.Contains(out, "xoxb-bot-secret") {
		t.Fatalf("config leaked tokens: %s", out)
	}
	if !strings.Contains(out, "has_user_token: true") || !strings.Contains(out, "preset: 1") {
		t.Fatalf("expected redacted summary, got: %s", out)
	}
}

func TestConfigEditRequiresTTY(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	var stdout bytes.Buffer
	rt := NewRuntime()
	rt.Stdout = &stdout
	rt.Stderr = &bytes.Buffer{}
	rt.IsTTY = func() bool { return false }
	opened := false
	rt.OpenEditor = func(path string, bootstrap string) error {
		opened = true
		return nil
	}
	err := rt.Run([]string{"config", "edit"})
	if err == nil {
		t.Fatal("expected config edit without TTY to fail")
	}
	if opened {
		t.Fatal("editor must not open without TTY")
	}
	if !strings.Contains(err.Error(), "interactive TTY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONFlagAlias(t *testing.T) {
	listArgs, err := parseArgs([]string{"1", "list", "in", "C123", "limit", "5", "--json"})
	if err != nil || !listArgs.OutputJSON || listArgs.ListIn != "C123" {
		t.Fatalf("unexpected --json list parse: %+v err=%v", listArgs, err)
	}
	threadArgs, err := parseArgs([]string{"1", "thread", "C123:1712764800.000100", "--json"})
	if err != nil || !threadArgs.OutputJSON {
		t.Fatalf("unexpected --json thread parse: %+v err=%v", threadArgs, err)
	}
	configArgs, err := parseArgs([]string{"config", "--json"})
	if err != nil || configArgs.Command != "config" || !configArgs.OutputJSON {
		t.Fatalf("unexpected config --json parse: %+v err=%v", configArgs, err)
	}
	accountsArgs, err := parseArgs([]string{"accounts", "list", "--json"})
	if err != nil || !accountsArgs.OutputJSON {
		t.Fatalf("unexpected accounts --json parse: %+v err=%v", accountsArgs, err)
	}
}

func TestVerboseFlagParse(t *testing.T) {
	args, err := parseArgs([]string{"1", "list", "in", "C123", "since", "4h", "verbose", "--json"})
	if err != nil || !args.Verbose || !args.OutputJSON || args.ListIn != "C123" {
		t.Fatalf("unexpected verbose list parse: %+v err=%v", args, err)
	}
	threadArgs, err := parseArgs([]string{"1", "thread", "C123:1712764800.000100", "--verbose", "limit", "5"})
	if err != nil || !threadArgs.Verbose || threadArgs.ListLimit != 5 {
		t.Fatalf("unexpected verbose thread parse: %+v err=%v", threadArgs, err)
	}
}

func TestVersionPrintsC3POVersion(t *testing.T) {
	var stdout bytes.Buffer
	rt := NewRuntime()
	rt.Stdout = &stdout
	rt.Stderr = &bytes.Buffer{}
	if err := rt.Run([]string{"version"}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) != version.String() {
		t.Fatalf("version %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "sluser") {
		t.Fatal("version names sluser")
	}
}

func TestListUnreadBypassesCache(t *testing.T) {
	if !shouldBypassMessageCache(Args{ListFilter: "unread"}) || !shouldBypassMessageCache(Args{ListFilter: "read"}) {
		t.Fatal("unread/read list must bypass cache")
	}
	if shouldBypassMessageCache(Args{ListFilter: "all"}) {
		t.Fatal("unfiltered list should still allow cache")
	}
}
