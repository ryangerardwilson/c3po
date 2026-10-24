package gmail

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
)

func TestTidyNoUnreadSkipsDecisions(t *testing.T) {
	e, _ := tidyEnv(t, nil)
	called := false
	e.Decide = func([]mailBrief) (map[string]mailVerdict, error) {
		called = true
		return nil, nil
	}
	var out, err bytes.Buffer
	e.Stdout, e.Stderr = &out, &err
	if code := e.runCode([]string{"tidy"}); code != 0 {
		t.Fatalf("tidy %d %s", code, err.String())
	}
	if called {
		t.Fatal("decisions ran with no unread mail")
	}
	if !strings.Contains(out.String(), "gmail tidy complete: presets=2 unread=0 trashed=0 marked_read=0 spam_added=0 left=0") {
		t.Fatalf("summary %s", out.String())
	}
	body := readConfig(t, e)
	for _, domain := range []string{"@clickup.com", "@mail.clickup.com", "@tasks.clickup.com"} {
		if !strings.Contains(body, domain) {
			t.Fatalf("missing %s in %s", domain, body)
		}
	}
}

func TestTidyMissingGrokLeavesMail(t *testing.T) {
	e, boxes := tidyEnv(t, []api.Message{{
		ID: "m1", Headers: map[string]string{"from": "Promo <promo@bulk.test>", "subject": "Sale"}, Snippet: "50% off",
	}})
	var out, err bytes.Buffer
	e.Stdout, e.Stderr = &out, &err
	code := e.runCode([]string{"tidy"})
	if code != 1 || !strings.Contains(err.String(), "grok is required to classify unread mail") {
		t.Fatalf("missing grok %d %s", code, err.String())
	}
	if len(boxes["1"].ones) != 0 || len(boxes["1"].reads) != 0 {
		t.Fatalf("mail changed ones=%v reads=%v", boxes["1"].ones, boxes["1"].reads)
	}
}

func TestTidyAppliesVerdicts(t *testing.T) {
	msgs := []api.Message{
		{ID: "t1", Headers: map[string]string{"from": "Promo <promo@bulk.test>", "subject": "Sale"}, Snippet: "50% off"},
		{ID: "r1", Headers: map[string]string{"from": "Billing <bill@shop.test>", "subject": "Receipt"}, Snippet: "thanks for your order"},
		{ID: "k1", Headers: map[string]string{"from": "Ada <ada@example.com>", "subject": "Hello"}, Snippet: "are you free"},
		{ID: "u1", Headers: map[string]string{"from": "Maybe <maybe@bulk.test>", "subject": "Offer"}, Snippet: "limited"},
	}
	e, boxes := tidyEnv(t, msgs)
	e.Decide = func(mail []mailBrief) (map[string]mailVerdict, error) {
		if len(mail) != 4 {
			t.Fatalf("briefs %d", len(mail))
		}
		return map[string]mailVerdict{
			mailKey("1", "t1"): {Action: "trash", Confidence: 0.95, Disposable: 0.97},
			mailKey("1", "r1"): {Action: "read", Confidence: 0.91},
			mailKey("1", "k1"): {Action: "keep", Confidence: 0.99, Disposable: 0.99},
			mailKey("1", "u1"): {Action: "trash", Confidence: 0.4, Disposable: 0.99},
		}, nil
	}
	var out, err bytes.Buffer
	e.Stdout, e.Stderr = &out, &err
	if code := e.runCode([]string{"tidy"}); code != 0 {
		t.Fatalf("tidy %d %s %s", code, out.String(), err.String())
	}
	if strings.Join(boxes["1"].ones, ",") != "t1" {
		t.Fatalf("trashed %v", boxes["1"].ones)
	}
	if strings.Join(boxes["1"].reads, ",") != "r1" {
		t.Fatalf("reads %v", boxes["1"].reads)
	}
	if !strings.Contains(out.String(), "gmail tidy complete: presets=2 unread=4 trashed=1 marked_read=1 spam_added=1 left=2") {
		t.Fatalf("summary %s", out.String())
	}
	body := readConfig(t, e)
	if !strings.Contains(body, "promo@bulk.test") || strings.Contains(body, "ada@example.com") || strings.Contains(body, "maybe@bulk.test") {
		t.Fatalf("senders %s", body)
	}
}

func tidyEnv(t *testing.T, msgs []api.Message) (Env, map[string]*fakeBox) {
	t.Helper()
	e := testEnv(t, nil)
	boxes := map[string]*fakeBox{}
	e.Box = func(acct config.Account) (Service, error) {
		box := boxes[acct.Preset]
		if box == nil {
			box = &fakeBox{}
			if acct.Preset == "1" && msgs != nil {
				box.lists = map[string][]api.Message{unreadQuery: msgs}
			}
			boxes[acct.Preset] = box
		}
		return box, nil
	}
	return e, boxes
}

func readConfig(t *testing.T, e Env) string {
	t.Helper()
	body, err := os.ReadFile(e.paths().ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
