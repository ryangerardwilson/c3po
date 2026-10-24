package gmail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	decisionsURL   = "https://api.openai.com/v1/decisions"
	decisionsModel = "gpt-6-luna"
	decisionChunk  = 150
)

const decisionRubric = `Classify each unread message.
trash: sales, promo, newsletter, digest, iOS or TestFlight, Naukri or a job board, a Slack automated summary, a failed GitHub or gh run, ClickUp, bank marketing about features, schemes, offers, cards, loans, insurance, or app upsells, and cold outreach that looks personal. A human-looking From name still means trash when the mail is sales, lead-gen, a demo booking, link-building, a partnership pitch, or a nurture sequence.
read: a receipt, an invoice, or other automated mail that should be kept and marked read.
keep: direct human mail, security, fraud, OTP, login, or legal, regulatory, or tax mail. Leave it unread.
A disposable sender is a bulk, automated, or one-off promo address that should be added to the spam list. A person, a bank, a government office, and a sender of receipts or security mail is not disposable.`

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

func classifyMail(client *http.Client, endpoint, key string, mail []mailBrief) (map[string]mailVerdict, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required to classify unread mail")
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if endpoint == "" {
		endpoint = decisionsURL
	}
	out := map[string]mailVerdict{}
	for start := 0; start < len(mail); start += decisionChunk {
		end := start + decisionChunk
		if end > len(mail) {
			end = len(mail)
		}
		part, err := decideChunk(client, endpoint, key, mail[start:end])
		if err != nil {
			return nil, err
		}
		for id, verdict := range part {
			out[id] = verdict
		}
	}
	return out, nil
}

func decideChunk(client *http.Client, endpoint, key string, mail []mailBrief) (map[string]mailVerdict, error) {
	evidence := make([]map[string]any, len(mail))
	questions := make([]any, 0, len(mail)*2)
	for i, brief := range mail {
		evidence[i] = map[string]any{
			"index":   i,
			"from":    brief.From,
			"subject": clip(brief.Subject, 300),
			"snippet": clip(brief.Snippet, 500),
		}
		questions = append(questions, choiceQuestion(i), predicateQuestion(i))
	}
	payload := map[string]any{
		"model": decisionsModel,
		"input": []any{
			map[string]any{"role": "developer", "content": decisionRubric},
			map[string]any{"role": "user", "content": []any{
				map[string]any{"type": "input_text", "text": "Question aN and dN refer to evidence index N."},
				map[string]any{"type": "input_json", "data": evidence},
			}},
		},
		"questions": questions,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("decisions api returned %d", resp.StatusCode)
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
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
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

func choiceQuestion(i int) map[string]any {
	return map[string]any{
		"type":         "choice",
		"name":         fmt.Sprintf("a%d", i),
		"instructions": fmt.Sprintf("For evidence index %d, choose trash, read, or keep.", i),
		"choices": []map[string]string{
			{"value": "trash", "description": "Sales, promo, newsletter, digest, TestFlight, job board, Slack summary, failed GitHub run, ClickUp, bank marketing, or cold outreach, including mail with a human-looking From name."},
			{"value": "read", "description": "A receipt, invoice, or other automated mail to keep and mark read."},
			{"value": "keep", "description": "Direct human mail, or security, fraud, OTP, login, legal, regulatory, or tax mail. Leave it unread."},
		},
	}
}

func predicateQuestion(i int) map[string]any {
	return map[string]any{
		"type":         "predicate",
		"name":         fmt.Sprintf("d%d", i),
		"instructions": fmt.Sprintf("For evidence index %d, the From address is a disposable sender that should be added to the spam list.", i),
	}
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
