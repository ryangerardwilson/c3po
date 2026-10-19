package item

import (
	"bytes"
	"strings"
	"testing"
)

func TestTemplateMetaNeedsTitleAndCategory(t *testing.T) {
	cases := map[string]string{
		`[]`:                                 "Template is not a 1Password item.",
		`{"category":"X"}`:                   "Template needs a title.",
		`{"title":"T"}`:                      "Template needs a category.",
		`{"title":" T ","category":"LOGIN"}`: "",
	}
	for body, want := range cases {
		title, _, err := TemplateMeta([]byte(body))
		if want == "" {
			if err != nil || title != "T" {
				t.Fatalf("%s title = %q err = %v", body, title, err)
			}
			continue
		}
		if err == nil || err.Error() != want {
			t.Fatalf("%s err = %v", body, err)
		}
	}
}

func TestSetSecretAddsAMissingField(t *testing.T) {
	doc := map[string]any{"fields": []any{}}
	SetSecret(doc, "PASSWORD", "added-secret")
	body, err := Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"purpose":"PASSWORD"`) || !strings.Contains(string(body), "added-secret") {
		t.Fatalf("body = %s", body)
	}
}

func TestNewRejectsAnotherCategory(t *testing.T) {
	if _, err := New("T", "SECURE_NOTE", "", "s"); err == nil {
		t.Fatal("secure note was built")
	}
}

func TestWriteShowFallsBackToTheListedItem(t *testing.T) {
	var out bytes.Buffer
	WriteShow(&out, map[string]any{"fields": []any{map[string]any{"id": "x"}, "junk", map[string]any{}}}, "T", "id1", "LOGIN")
	if out.String() != "Title: T\nID: id1\nCategory: LOGIN\nx (FIELD)\n" {
		t.Fatalf("show = %q", out.String())
	}
}
