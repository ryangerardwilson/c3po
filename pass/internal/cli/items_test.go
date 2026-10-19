package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func decode(t *testing.T, text string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestCreateAPICredentialSendsTemplateOnStdin(t *testing.T) {
	fake := &fakeOp{list: "[]"}
	runner, stdout, stderr, _ := newRunner(t, fake)
	runner.In = strings.NewReader("new-credential\n")
	rc := runner.Run([]string{"Root", "--create", "--title", "NEW_TOKEN", "--category", "api-credential"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Created API credential NEW_TOKEN in vault Root.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	call, ok := fake.call("item create")
	if !ok || call.args != "item create --vault Root -" {
		t.Fatalf("args = %q", call.args)
	}
	doc := decode(t, call.stdin)
	if doc["title"] != "NEW_TOKEN" || doc["category"] != "API_CREDENTIAL" {
		t.Fatal("wrong title or category")
	}
	fields, _ := doc["fields"].([]any)
	if len(fields) != 1 {
		t.Fatalf("fields = %d", len(fields))
	}
	field, _ := fields[0].(map[string]any)
	if field["id"] != "credential" || field["value"] != "new-credential" {
		t.Fatal("credential field is wrong")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "new-credential")
}

func TestCreateLoginRequiresUsernameAndHidesPassword(t *testing.T) {
	fake := &fakeOp{list: "[]"}
	runner, stdout, stderr, _ := newRunner(t, fake)
	runner.In = strings.NewReader("login-password\n")
	rc := runner.Run([]string{"Root", "--create", "--title", "Example", "--category", "login"})
	if rc != 1 || stderr.String() != "A login needs --username.\n" {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
	stderr.Reset()
	rc = runner.Run([]string{"Root", "--create", "--title=Example", "--category", "login", "--username", "ada@example.com"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Created login Example in vault Root.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	call, _ := fake.call("item create")
	raw, _ := json.Marshal(decode(t, call.stdin)["fields"])
	if !strings.Contains(string(raw), "ada@example.com") || !strings.Contains(string(raw), "login-password") {
		t.Fatal("login fields are wrong")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "login-password")
}

func TestCreateRejectsATerminalAndADuplicate(t *testing.T) {
	fake := &fakeOp{list: `[{"id":"one","title":"NEW_TOKEN","category":"API_CREDENTIAL","vault":{"id":"vault-id"}}]`}
	runner, _, stderr, _ := newRunner(t, fake)
	rc := runner.Run([]string{"Root", "--create", "--title", "NEW_TOKEN", "--category", "api-credential"})
	if rc == 0 || !strings.Contains(stderr.String(), "Pass the secret on stdin.") {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
	if len(fake.calls) != 0 {
		t.Fatal("empty stdin create called op")
	}

	stderr.Reset()
	runner.In = strings.NewReader("new-credential\n")
	rc = runner.Run([]string{"Root", "--create", "--title", "NEW_TOKEN", "--category", "api-credential"})
	if rc == 0 || !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
	if _, ok := fake.call("item create"); ok {
		t.Fatal("duplicate create called op item create")
	}
	assertNoSecret(t, "", stderr.String(), "new-credential")
}

func TestCreateRejectsASecretFlag(t *testing.T) {
	fake := &fakeOp{}
	runner, stdout, stderr, _ := newRunner(t, fake)
	rc := runner.Run([]string{"Root", "--create", "--title", "NEW_TOKEN", "--category", "api-credential", "--password", "flag-secret"})
	if rc == 0 || !strings.Contains(stderr.String(), "Unknown flag --password.") {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
	if len(fake.calls) != 0 {
		t.Fatal("flag create called op")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "flag-secret")
}

func TestItemFlagErrors(t *testing.T) {
	cases := map[string][]string{
		"--title needs a value.\n":                   {"Root", "--create", "--title"},
		"--title was given twice.\n":                 {"Root", "--create", "--title", "a", "--title", "b"},
		"--title must not be empty.\n":               {"Root", "--create", "--title= "},
		createUsage + "\n":                           {"Root", "--create", "stray"},
		"Use the template or the flags, not both.\n": {"Root", "--create", "--template", "x.json", "--title", "a"},
		"Category must be api-credential or login. Use --template for another category.\n": {"Root", "--create", "--title", "a", "--category", "note"},
	}
	for want, args := range cases {
		runner, _, stderr, _ := newRunner(t, &fakeOp{})
		if rc := runner.Run(args); rc != 1 || stderr.String() != want {
			t.Fatalf("%q rc = %d stderr = %q", args, rc, stderr.String())
		}
	}
}

func TestCreateFromTemplate(t *testing.T) {
	fake := &fakeOp{list: "[]"}
	runner, stdout, stderr, _ := newRunner(t, fake)
	template := filepath.Join(t.TempDir(), "item.json")
	body := `{"title":"From Template","category":"SECURE_NOTE","fields":[{"id":"notesPlain","value":"template-secret"}]}`
	if err := os.WriteFile(template, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	rc := runner.Run([]string{"Root", "--create", "--template", template})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Created SECURE_NOTE From Template in vault Root.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if call, _ := fake.call("item create"); call.stdin != body {
		t.Fatal("template was not sent unchanged")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "template-secret")
}

func TestCreateFromTemplateOnStdin(t *testing.T) {
	fake := &fakeOp{list: "[]"}
	runner, stdout, stderr, _ := newRunner(t, fake)
	body := `{"title":"Piped","category":"LOGIN"}`
	runner.In = strings.NewReader(body)
	rc := runner.Run([]string{"Root", "--create", "--template", "-"})
	if rc != 0 || stdout.String() != "Created login Piped in vault Root.\n" {
		t.Fatalf("rc = %d stdout = %q stderr = %q", rc, stdout.String(), stderr.String())
	}
	if call, _ := fake.call("item create"); call.stdin != body {
		t.Fatal("stdin template was not sent unchanged")
	}
	runner, _, stderr, _ = newRunner(t, &fakeOp{})
	if rc := runner.Run([]string{"Root", "--create", "--template", "-"}); rc != 1 || stderr.String() != "Pass the template on stdin.\n" {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
}

func TestEditReplacesCredentialWithoutPrintingIt(t *testing.T) {
	fake := &fakeOp{
		list: `[{"id":"one","title":"NEW_TOKEN","category":"API_CREDENTIAL","vault":{"id":"vault-id"}}]`,
		get:  `{"id":"one","title":"NEW_TOKEN","category":"API_CREDENTIAL","fields":[{"id":"credential","type":"CONCEALED","label":"credential","value":"old-secret"}]}`,
	}
	runner, stdout, stderr, _ := newRunner(t, fake)
	runner.In = strings.NewReader("rotated-secret\n")
	rc := runner.Run([]string{"Wiom", "--edit", "new_token"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Updated NEW_TOKEN in vault Wiom.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	call, _ := fake.call("item edit")
	if call.args != "item edit one --vault Wiom" {
		t.Fatalf("args = %q", call.args)
	}
	if _, ok := fake.call("item get one --vault Wiom --format json"); !ok {
		t.Fatal("edit did not read the item")
	}
	fields, _ := decode(t, call.stdin)["fields"].([]any)
	field, _ := fields[0].(map[string]any)
	if field["value"] != "rotated-secret" {
		t.Fatal("credential was not replaced")
	}
	if strings.Contains(call.stdin, "old-secret") {
		t.Fatal("old secret was left on the item")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "rotated-secret", "old-secret")
}

func TestEditRenamesWithoutReadingStdin(t *testing.T) {
	fake := &fakeOp{
		list: `[{"id":"one","title":"Example","category":"LOGIN","vault":{"id":"vault-id"}}]`,
		get:  `{"id":"one","title":"Example","category":"LOGIN","fields":[{"id":"password","type":"CONCEALED","label":"password","value":"kept-secret"}]}`,
	}
	runner, stdout, stderr, _ := newRunner(t, fake)
	rc := runner.Run([]string{"ADE", "--edit", "Example", "--title", "Renamed", "--username", "ada@example.com"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Updated Renamed in vault ADE.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	call, _ := fake.call("item edit")
	doc := decode(t, call.stdin)
	if doc["title"] != "Renamed" {
		t.Fatalf("title = %#v", doc["title"])
	}
	if !strings.Contains(call.stdin, "kept-secret") || !strings.Contains(call.stdin, `"purpose":"USERNAME"`) {
		t.Fatal("edit lost the secret or the username")
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "kept-secret")
}

func TestEditProblems(t *testing.T) {
	list := `[{"id":"n","title":"Note","category":"SECURE_NOTE","vault":{"id":"vault-id"}}]`
	cases := []struct {
		args  []string
		input string
		want  string
	}{
		{[]string{"Root", "--edit", "Note"}, "", "Nothing to change.\n"},
		{[]string{"Root", "--edit", "Note"}, "s3cret\n", "This category has no single secret field. Use --template.\n"},
		{[]string{"Root", "--edit", "Note", "--username", "u"}, "", "--username applies to an API credential or a login. Use --template for another category.\n"},
		{[]string{"Root", "--edit"}, "", editUsage + "\n"},
		{[]string{"Root", "--edit", "Note", "--template", "x.json", "--title", "a"}, "", "Use the template or the flags, not both.\n"},
	}
	for _, c := range cases {
		fake := &fakeOp{list: list}
		runner, _, stderr, _ := newRunner(t, fake)
		runner.In = strings.NewReader(c.input)
		if rc := runner.Run(c.args); rc != 1 || stderr.String() != c.want {
			t.Fatalf("%q rc = %d stderr = %q", c.args, rc, stderr.String())
		}
		if _, ok := fake.call("item edit"); ok {
			t.Fatal("failed edit called op item edit")
		}
	}
}

func TestEditFromTemplate(t *testing.T) {
	fake := &fakeOp{list: `[{"id":"n","title":"Note","category":"SECURE_NOTE","vault":{"id":"vault-id"}}]`}
	runner, stdout, stderr, _ := newRunner(t, fake)
	body := `{"title":"Note","category":"SECURE_NOTE","notesPlain":"note-secret"}`
	runner.In = strings.NewReader(body)
	rc := runner.Run([]string{"Root", "--edit", "Note", "--template", "-"})
	if rc != 0 || stdout.String() != "Updated Note in vault Root.\n" {
		t.Fatalf("rc = %d stdout = %q stderr = %q", rc, stdout.String(), stderr.String())
	}
	if call, _ := fake.call("item edit"); call.stdin != body || call.args != "item edit n --vault Root" {
		t.Fatalf("args = %q", call.args)
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "note-secret")
}

func TestEditRequiresOneExactTitle(t *testing.T) {
	fake := &fakeOp{list: `[
	  {"id":"one","title":"Same","category":"LOGIN","vault":{"id":"vault-id"}},
	  {"id":"two","title":"same","category":"LOGIN","vault":{"id":"vault-id"}}
	]`}
	runner, _, stderr, _ := newRunner(t, fake)
	runner.In = strings.NewReader("rotated-secret\n")
	rc := runner.Run([]string{"Root", "--edit", "Same"})
	if rc == 0 || !strings.Contains(stderr.String(), "More than one item titled Same") {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
	if _, ok := fake.call("item edit"); ok {
		t.Fatal("ambiguous edit called op item edit")
	}
	assertNoSecret(t, "", stderr.String(), "rotated-secret")
}

func TestDeleteUsesTheExactTitleID(t *testing.T) {
	fake := &fakeOp{list: `[
	  {"id":"one","title":"Token","category":"API_CREDENTIAL","vault":{"id":"vault-id"}},
	  {"id":"two","title":"Token Extra","category":"API_CREDENTIAL","vault":{"id":"vault-id"}}
	]`}
	runner, stdout, stderr, _ := newRunner(t, fake)
	rc := runner.Run([]string{"Root", "--delete", "Tok"})
	if rc == 0 || !strings.Contains(stderr.String(), "No item titled Tok") {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
	if _, ok := fake.call("item delete"); ok {
		t.Fatal("substring delete called op")
	}
	stdout.Reset()
	stderr.Reset()
	rc = runner.Run([]string{"Root", "--delete", "Token"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	if stdout.String() != "Deleted Token from vault Root.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if call, _ := fake.call("item delete"); call.args != "item delete one --vault Root" {
		t.Fatalf("args = %q", call.args)
	}
}

func TestDeleteAndShowUsage(t *testing.T) {
	cases := map[string][]string{
		"Use: c3po pass <vault> --delete <title>\n": {"Root", "--delete", "--title"},
		"Use: c3po pass <vault> --show <title>\n":   {"Root", "--show"},
	}
	for want, args := range cases {
		runner, _, stderr, _ := newRunner(t, &fakeOp{})
		if rc := runner.Run(args); rc != 1 || stderr.String() != want {
			t.Fatalf("%q rc = %d stderr = %q", args, rc, stderr.String())
		}
	}
}

func TestShowPrintsUsernameAndHidesSecrets(t *testing.T) {
	fake := &fakeOp{
		list: `[{"id":"one","title":"Example","category":"LOGIN","vault":{"id":"vault-id"}}]`,
		get: `{
		  "id":"one",
		  "title":"Example",
		  "category":"LOGIN",
		  "fields":[
		    {"id":"username","type":"STRING","purpose":"USERNAME","label":"username","value":"ada@example.com"},
		    {"id":"password","type":"CONCEALED","purpose":"PASSWORD","label":"password","value":"show-secret"},
		    {"id":"notesPlain","type":"STRING","purpose":"NOTES","label":"notesPlain","value":"note-secret"}
		  ]
		}`,
	}
	runner, stdout, stderr, _ := newRunner(t, fake)
	rc := runner.Run([]string{"Root", "--show", "Example"})
	if rc != 0 {
		t.Fatalf("rc = %d\nstderr: %s", rc, stderr.String())
	}
	want := "Title: Example\nID: one\nCategory: LOGIN\nusername (STRING): ada@example.com\npassword (CONCEALED)\nnotesPlain (STRING)\n"
	if stdout.String() != want {
		t.Fatalf("show = %q", stdout.String())
	}
	assertNoSecret(t, stdout.String(), stderr.String(), "show-secret", "note-secret")
}
