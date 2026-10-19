package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const searchList = `[
  {
    "id": "one",
    "title": "GitHub Admin",
    "category": "LOGIN",
    "vault": {"id": "vault-id"},
    "additional_information": "admin@example.com",
    "fields": [
      {"label": "password", "value": "one-secret"}
    ]
  },
  {
    "id": "two",
    "title": "GitHub Bot",
    "category": "LOGIN",
    "vault": {"id": "vault-id"},
    "fields": [
      {"label": "username", "value": "bot@example.com"}
    ]
  },
  {
    "id": "note",
    "title": "Notes",
    "category": "SECURE_NOTE",
    "vault": {"id": "vault-id"},
    "additional_information": "hidden@example.com"
  }
]`

func searchOp() *fakeOp {
	return &fakeOp{
		list:  searchList,
		reads: map[string]string{"/one/password": "one-secret", "/two/password": "two-secret"},
	}
}

const mapList = `[
  {"id":"hissa","title":"hissa","category":"LOGIN","vault":{"id":"vault-id"}},
  {"id":"keka","title":"keka","category":"LOGIN","vault":{"id":"vault-id"}},
  {"id":"acct","title":"WIOM_SNOWFLAKE_ACCOUNT","category":"API_CREDENTIAL","vault":{"id":"vault-id"}},
  {"id":"user","title":"WIOM_SNOWFLAKE_USER","category":"API_CREDENTIAL","vault":{"id":"vault-id"}},
  {"id":"odd","title":"not a shell name","category":"LOGIN","vault":{"id":"vault-id"}}
]`

func mapOp() *fakeOp {
	return &fakeOp{
		list: mapList,
		reads: map[string]string{
			"/hissa/password":  "hissa-secret",
			"/keka/password":   "keka-secret",
			"/acct/credential": "acct-secret",
			"/user/credential": "user-secret",
			"/odd/password":    "odd' secret",
		},
	}
}

func TestLookupCopiesOneTitleMatch(t *testing.T) {
	fake := searchOp()
	runner, stdout, stderr, clipboard := newRunner(t, fake)
	rc := runner.Run([]string{"Root", "GitHub Admin"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Copied password for GitHub Admin (admin@example.com)\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if clipboard.text != "one-secret" {
		t.Fatal("clipboard did not receive the selected password")
	}
	if _, ok := fake.call("read --no-newline op://vault-id/one/password"); !ok {
		t.Fatalf("calls = %#v", fake.calls)
	}
	if _, ok := fake.call("item list --vault Root --long --format=json"); !ok {
		t.Fatalf("calls = %#v", fake.calls)
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "one-secret")
}

func TestLookupCopiesUsernameMatch(t *testing.T) {
	runner, stdout, stderr, clipboard := newRunner(t, searchOp())
	rc := runner.Run([]string{"Root", "bot@example.com"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Copied password for GitHub Bot (bot@example.com)\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if clipboard.text != "two-secret" {
		t.Fatal("clipboard did not receive the username match")
	}
}

func TestLookupCopiesTheNumberedChoice(t *testing.T) {
	runner, stdout, stderr, clipboard := newRunner(t, searchOp())
	runner.In = strings.NewReader("2\n")
	rc := runner.Run([]string{"Root", "github"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	for _, needle := range []string{
		"1) GitHub Admin | admin@example.com",
		"2) GitHub Bot | bot@example.com",
		"Select a match [1-2]: ",
		"Copied password for GitHub Bot (bot@example.com)",
	} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("stdout missing %q:\n%s", needle, stdout.String())
		}
	}
	if clipboard.text != "two-secret" {
		t.Fatal("numbered choice did not copy that password")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "two-secret")
}

func TestLookupRejectsABadSelection(t *testing.T) {
	cases := map[string]string{
		"3\n":   "Selection is out of range.\n",
		"two\n": "Selection must be a number.\n",
		"":      "No selection was entered.\n",
	}
	for input, want := range cases {
		runner, _, stderr, clipboard := newRunner(t, searchOp())
		runner.In = strings.NewReader(input)
		if rc := runner.Run([]string{"Root", "github"}); rc != 1 || stderr.String() != want {
			t.Fatalf("%q rc = %d stderr = %q", input, rc, stderr.String())
		}
		if clipboard.copied {
			t.Fatal("bad selection copied a password")
		}
	}
}

func TestLookupIgnoresUsernameOnNonLoginItems(t *testing.T) {
	runner, _, stderr, _ := newRunner(t, searchOp())
	rc := runner.Run([]string{"Root", "hidden@example.com"})
	if rc != 1 {
		t.Fatal("secure note username matched")
	}
	if !strings.Contains(stderr.String(), "No matches for hidden@example.com in vault Root.") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLookupReadsTokenFileWhenEnvironmentIsEmpty(t *testing.T) {
	runner, stdout, stderr, clipboard := newRunner(t, searchOp())
	home := t.TempDir()
	tokenPath := filepath.Join(home, ".config", "c3po", "pass", "service-account-token")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokenPath, []byte(fakeToken+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN_FILE", "")
	rc := runner.Run([]string{"Root", "GitHub Admin"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if clipboard.text != "one-secret" {
		t.Fatal("token file was not used")
	}
	assertNoSecret(t, stdout.String(), stderr.String())
}

func TestLookupReportsMissingToken(t *testing.T) {
	fake := searchOp()
	runner, _, stderr, _ := newRunner(t, fake)
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "")
	rc := runner.Run([]string{"Root", "github"})
	if rc != 1 {
		t.Fatal("missing token succeeded")
	}
	if stderr.String() != "1Password service account token is missing.\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if len(fake.calls) != 0 {
		t.Fatal("missing token called op")
	}
}

func TestRejectedTokenFailsTheList(t *testing.T) {
	runner, _, stderr, _ := newRunner(t, searchOp())
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "other-token")
	rc := runner.Run([]string{"Root", "github"})
	if rc != 1 || !strings.Contains(stderr.String(), "Unable to list 1Password items in vault Root.") {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
}

func TestListVaultPrintsTitlesWithoutPasswords(t *testing.T) {
	runner, stdout, stderr, _ := newRunner(t, searchOp())
	rc := runner.Run([]string{"root", "-l"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "GitHub Admin | admin@example.com\nGitHub Bot | bot@example.com\nNotes\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "one-secret", "hidden@example.com")
}

func TestListEmptyVault(t *testing.T) {
	runner, stdout, _, _ := newRunner(t, &fakeOp{})
	if rc := runner.Run([]string{"Empty", "-l"}); rc != 0 || stdout.String() != "No items in vault Empty.\n" {
		t.Fatalf("rc = %d stdout = %q", rc, stdout.String())
	}
}

func TestNamedItemsCopyJSON(t *testing.T) {
	runner, stdout, stderr, clipboard := newRunner(t, mapOp())
	rc := runner.Run([]string{"wiom", "hissa", "keka"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Copied 2 secrets to the clipboard.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(clipboard.text), &got); err != nil {
		t.Fatal(err)
	}
	if got["hissa"] != "hissa-secret" || got["keka"] != "keka-secret" || len(got) != 2 {
		t.Fatalf("map has wrong keys")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "hissa-secret")
}

func TestGlobLoadsExportsIntoTheShell(t *testing.T) {
	runner, _, stderr, clipboard := newRunner(t, mapOp())
	stdout, rc := runPiped(t, runner, "wiom", "WIOM_SNOWFLAKE_*")
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	want := "export WIOM_SNOWFLAKE_ACCOUNT='acct-secret'\nexport WIOM_SNOWFLAKE_USER='user-secret'\n"
	if stdout != want {
		t.Fatalf("stdout = %q", stdout)
	}
	if clipboard.copied {
		t.Fatal("shell load copied to the clipboard")
	}
}

func TestShellGetsJSONWhenATitleIsNotAShellName(t *testing.T) {
	runner, _, stderr, clipboard := newRunner(t, mapOp())
	stdout, rc := runPiped(t, runner, "wiom", "hissa", "not a shell name")
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	want := "{\n  \"hissa\": \"hissa-secret\",\n  \"not a shell name\": \"odd' secret\"\n}\n"
	if stdout != want {
		t.Fatalf("stdout = %q", stdout)
	}
	if clipboard.copied {
		t.Fatal("shell map copied to the clipboard")
	}
}

func TestGlobCopiesJSONInATerminal(t *testing.T) {
	runner, _, stderr, clipboard := newRunner(t, mapOp())
	rc := runner.Run([]string{"wiom", "wiom_snowflake_*"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(clipboard.text), &got); err != nil {
		t.Fatal(err)
	}
	if got["WIOM_SNOWFLAKE_ACCOUNT"] != "acct-secret" || got["WIOM_SNOWFLAKE_USER"] != "user-secret" {
		t.Fatal("map has wrong values")
	}
}

func TestAgentPrintsOneSecret(t *testing.T) {
	runner, _, stderr, clipboard := newRunner(t, searchOp())
	stdout, rc := runPiped(t, runner, "Root", "GitHub Admin")
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout != "one-secret\n" {
		t.Fatalf("stdout = %q", stdout)
	}
	if clipboard.copied {
		t.Fatal("agent copied a single secret")
	}
}

func TestAgentDoesNotPromptForSeveralMatches(t *testing.T) {
	runner, _, stderr, clipboard := newRunner(t, searchOp())
	runner.In = strings.NewReader("1\n")
	stdout, rc := runPiped(t, runner, "Root", "github")
	if rc != 1 {
		t.Fatal("several matches succeeded")
	}
	if !strings.Contains(stderr.String(), "More than one match for github in vault Root.") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout, "1) GitHub Admin | admin@example.com") || strings.Contains(stdout, "Select a match") {
		t.Fatalf("stdout = %q", stdout)
	}
	if clipboard.copied {
		t.Fatal("agent copied a password")
	}
}

func TestMapReportsAMissingName(t *testing.T) {
	runner, _, stderr, clipboard := newRunner(t, mapOp())
	rc := runner.Run([]string{"wiom", "hissa", "missing"})
	if rc != 1 {
		t.Fatal("missing name succeeded")
	}
	if !strings.Contains(stderr.String(), "No item named missing.") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if clipboard.copied {
		t.Fatal("missing name copied a secret")
	}
}

func TestMapReportsAnUnmatchedGlob(t *testing.T) {
	runner, _, stderr, _ := newRunner(t, mapOp())
	if rc := runner.Run([]string{"wiom", "NOPE_*"}); rc != 1 || stderr.String() != "No matches for NOPE_*.\n" {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
}
