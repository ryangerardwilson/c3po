package gmail

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGrokMapsAnswers(t *testing.T) {
	var prompts []string
	run := func(prompt string) ([]byte, error) {
		prompts = append(prompts, prompt)
		i := strings.LastIndex(prompt, "\n[")
		if i < 0 {
			t.Fatal("evidence missing")
		}
		var evidence []struct {
			From string `json:"from"`
		}
		if err := json.Unmarshal([]byte(prompt[i+1:]), &evidence); err != nil {
			t.Fatal(err)
		}
		answers := make([]map[string]any, 0, len(evidence)*2)
		for n, row := range evidence {
			choice, conf, prob := "keep", 0.0, 0.0
			if n == 0 && row.From == "promo@bulk.test" {
				choice, conf, prob = "trash", 1.0, 1.0
			}
			answers = append(answers,
				map[string]any{"type": "choice", "name": "a" + itoa(n), "choice": choice, "confidence": conf},
				map[string]any{"type": "predicate", "name": "d" + itoa(n), "probability": prob},
			)
		}
		body, err := json.Marshal(map[string]any{"answers": answers})
		if err != nil {
			t.Fatal(err)
		}
		return json.Marshal(map[string]any{"structuredOutput": json.RawMessage(body), "text": string(body)})
	}
	mail := make([]mailBrief, 151)
	for i := range mail {
		mail[i] = mailBrief{Preset: "1", ID: itoa(i), From: "a@b.test", Subject: "s", Snippet: "n"}
	}
	mail[0].ID = "first"
	mail[0].From = "promo@bulk.test"
	got, err := classifyMail(run, mail)
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 2 {
		t.Fatalf("calls %d", len(prompts))
	}
	if !strings.Contains(prompts[0], "a0") || !strings.Contains(prompts[0], "d0") || !strings.Contains(prompts[0], "promo@bulk.test") {
		t.Fatalf("prompt %s", prompts[0])
	}
	first := got[mailKey("1", "first")]
	if first.Action != "trash" || first.Confidence != 1 || first.Disposable != 1 {
		t.Fatalf("first %+v", first)
	}
	last := got[mailKey("1", mail[150].ID)]
	if last.Action != "keep" || last.Disposable != 0 {
		t.Fatalf("last %+v", last)
	}
}

func TestGrokRejectsAnErrorEnvelope(t *testing.T) {
	_, err := classifyMail(func(string) ([]byte, error) {
		return []byte(`{"type":"error","message":"secret-body"}`), nil
	}, []mailBrief{{Preset: "1", ID: "m"}})
	if err == nil || err.Error() != "grok classify failed" || strings.Contains(err.Error(), "secret-body") {
		t.Fatalf("err %v", err)
	}
}

func TestGrokArgsAreOneTurn(t *testing.T) {
	args := strings.Join(grokArgs("/tmp/empty", "/tmp/empty/prompt.txt"), "\n")
	for _, want := range []string{"--max-turns\n1", "--json-schema\n", "--cwd\n/tmp/empty", "--prompt-file\n/tmp/empty/prompt.txt", "--no-subagents", "--disable-web-search", "--verbatim", "run_terminal_cmd"} {
		if !strings.Contains(args, want) {
			t.Fatalf("missing %q\n%s", want, args)
		}
	}
	if strings.Contains(args, "--always-approve") {
		t.Fatal("always-approve")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
