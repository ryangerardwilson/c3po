package gmail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const decisionChunk = 150

const decisionRubric = `Classify each unread message.
trash: sales, promo, newsletter, digest, iOS or TestFlight, Naukri or a job board, a Slack automated summary, a failed GitHub or gh run, ClickUp, bank marketing about features, schemes, offers, cards, loans, insurance, or app upsells, and cold outreach that looks personal. A human-looking From name still means trash when the mail is sales, lead-gen, a demo booking, link-building, a partnership pitch, or a nurture sequence.
read: a receipt, an invoice, or other automated mail that should be kept and marked read.
keep: direct human mail, security, fraud, OTP, login, or legal, regulatory, or tax mail. Leave it unread.
A disposable sender is a bulk, automated, or one-off promo address that should be added to the spam list. A person, a bank, a government office, and a sender of receipts or security mail is not disposable.`

const decisionAsk = `For evidence index N, emit answer aN (type choice, choice trash, read, or keep, confidence from 0 to 1) and answer dN (type predicate, probability from 0 to 1). Index 0 uses a0 and d0.`

const decisionSchema = `{"type":"object","additionalProperties":false,"required":["answers"],"properties":{"answers":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["name","type"],"properties":{"name":{"type":"string"},"type":{"type":"string","enum":["choice","predicate"]},"choice":{"type":"string","enum":["trash","read","keep"]},"confidence":{"type":"number"},"probability":{"type":"number"}}}}}}`

const grokDisallowedTools = "read_file,grep,list_dir,web_search,web_fetch,write,search_replace,run_terminal_cmd,task,Agent,hashline_read,hashline_grep,hashline_edit"

var (
	errGrokMissing = errors.New("grok is not on PATH")
	errGrokFailed  = errors.New("grok classify failed")
	errGrokEmpty   = errors.New("grok classify returned no decisions")
)

type mailBrief struct {
	Preset  string
	ID      string
	From    string
	Subject string
	Snippet string
}

type mailVerdict struct {
	Action     string
	Confidence float64
	Disposable float64
}

func mailKey(preset, id string) string {
	return preset + "\x1f" + id
}

type grokRunner func(prompt string) ([]byte, error)

// classifyMail asks grok for the same choice and disposable answers a decisions
// call returned. The process gets an empty directory and one turn so it cannot
// walk a repository or run Gmail commands.
func classifyMail(run grokRunner, mail []mailBrief) (map[string]mailVerdict, error) {
	if len(mail) == 0 {
		return map[string]mailVerdict{}, nil
	}
	if run == nil {
		run = runGrok
	}
	out := map[string]mailVerdict{}
	for start := 0; start < len(mail); start += decisionChunk {
		end := start + decisionChunk
		if end > len(mail) {
			end = len(mail)
		}
		part, err := decideChunk(run, mail[start:end])
		if err != nil {
			return nil, err
		}
		for id, verdict := range part {
			out[id] = verdict
		}
	}
	return out, nil
}

func decideChunk(run grokRunner, mail []mailBrief) (map[string]mailVerdict, error) {
	prompt, err := decisionPrompt(mail)
	if err != nil {
		return nil, err
	}
	raw, err := run(prompt)
	if err != nil {
		if errors.Is(err, errGrokMissing) || errors.Is(err, errGrokFailed) || errors.Is(err, errGrokEmpty) {
			return nil, err
		}
		return nil, errGrokFailed
	}
	return parseGrokDecisions(raw, mail)
}

func decisionPrompt(mail []mailBrief) (string, error) {
	evidence := make([]map[string]any, len(mail))
	for i, brief := range mail {
		evidence[i] = map[string]any{
			"index":   i,
			"from":    brief.From,
			"subject": clip(brief.Subject, 300),
			"snippet": clip(brief.Snippet, 500),
		}
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return "", err
	}
	return decisionRubric + "\n\n" + decisionAsk + "\n" + string(raw), nil
}

func grokArgs(dir, promptFile string) []string {
	return []string{
		"--no-auto-update",
		"--cwd", dir,
		"--max-turns", "1",
		"--no-subagents",
		"--disable-web-search",
		"--no-plan",
		"--verbatim",
		"--system-prompt-override", decisionRubric,
		"--disallowed-tools", grokDisallowedTools,
		"--prompt-file", promptFile,
		"--json-schema", decisionSchema,
		"--output-format", "json",
	}
}

func runGrok(prompt string) ([]byte, error) {
	bin, err := exec.LookPath("grok")
	if err != nil {
		return nil, errGrokMissing
	}
	dir, err := os.MkdirTemp("", "c3po-gmail-tidy-")
	if err != nil {
		return nil, errGrokFailed
	}
	defer os.RemoveAll(dir)
	promptFile := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte(prompt), 0o600); err != nil {
		return nil, errGrokFailed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, grokArgs(dir, promptFile)...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, errGrokFailed
	}
	return stdout.Bytes(), nil
}

func parseGrokDecisions(raw []byte, mail []mailBrief) (map[string]mailVerdict, error) {
	var envelope struct {
		Type       string          `json:"type"`
		Text       string          `json:"text"`
		Structured json.RawMessage `json:"structuredOutput"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, errGrokEmpty
	}
	if envelope.Type == "error" {
		return nil, errGrokFailed
	}
	body := bytes.TrimSpace(envelope.Structured)
	if len(body) == 0 || bytes.Equal(body, []byte("null")) {
		body = []byte(strings.TrimSpace(envelope.Text))
	} else if body[0] == '"' {
		var text string
		if err := json.Unmarshal(body, &text); err != nil {
			return nil, errGrokEmpty
		}
		body = []byte(text)
	}
	var parsed struct {
		Answers []struct {
			Type        string  `json:"type"`
			Name        string  `json:"name"`
			Choice      string  `json:"choice"`
			Confidence  float64 `json:"confidence"`
			Probability float64 `json:"probability"`
		} `json:"answers"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Answers) == 0 {
		return nil, errGrokEmpty
	}
	out := make(map[string]mailVerdict, len(mail))
	for _, brief := range mail {
		out[mailKey(brief.Preset, brief.ID)] = mailVerdict{Action: "keep"}
	}
	for _, answer := range parsed.Answers {
		kind, n, ok := answerIndex(answer.Name)
		if !ok || n >= len(mail) {
			continue
		}
		id := mailKey(mail[n].Preset, mail[n].ID)
		verdict := out[id]
		switch {
		case kind == "a" && answer.Type == "choice":
			switch answer.Choice {
			case "trash", "read":
				verdict.Action = answer.Choice
			default:
				verdict.Action = "keep"
			}
			verdict.Confidence = answer.Confidence
		case kind == "d" && answer.Type == "predicate":
			verdict.Disposable = answer.Probability
		}
		out[id] = verdict
	}
	return out, nil
}

func answerIndex(name string) (string, int, bool) {
	if len(name) < 2 || (name[0] != 'a' && name[0] != 'd') {
		return "", 0, false
	}
	n, err := strconv.Atoi(name[1:])
	if err != nil || n < 0 {
		return "", 0, false
	}
	return name[:1], n, true
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
