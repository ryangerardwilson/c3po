package gmail

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

const (
	clickupSenders = "@clickup.com,@mail.clickup.com,@tasks.clickup.com"
	unreadQuery    = "is:unread -in:sent"
	minAction      = 0.8
	minDisposable  = 0.9
	wiomVault      = "Wiom"
	wiomOpenAIItem = "WIOM_OPENAI_API_KEY"
)

func (e Env) tidy() (int, error) {
	app, err := e.load()
	if err != nil {
		return 0, err
	}
	if len(app.Presets) == 0 {
		return 0, errs.Usage("No configured presets found.")
	}
	clickup := config.NormalizeSpamSenderList(csv(clickupSenders))
	for _, preset := range app.Presets {
		acct := app.Accounts[preset]
		merged := mergeUnique(acct.SpamSenders, clickup)
		if err := config.UpdateSpamSenders(app.Path, preset, merged); err != nil {
			return 0, err
		}
	}
	if _, err := e.spamCleanAll(); err != nil {
		return 0, err
	}
	app, err = e.load()
	if err != nil {
		return 0, err
	}
	services := map[string]Service{}
	var briefs []mailBrief
	for _, preset := range app.Presets {
		acct := app.Accounts[preset]
		svc, err := e.service(acct)
		if err != nil {
			return 0, err
		}
		services[preset] = svc
		msgs, err := svc.List(unreadQuery, 0)
		if err != nil {
			return 0, err
		}
		for _, msg := range msgs {
			row := summarize(msg, app.TimezoneOffset, true)
			from := row.FromEmail
			if from == "" {
				from = strings.ToLower(strings.TrimSpace(row.From))
			}
			snippet := row.Snippet
			if snippet == "" {
				snippet = row.Subject
			}
			briefs = append(briefs, mailBrief{
				Preset: preset, ID: msg.ID, From: from, Subject: row.Subject, Snippet: snippet,
			})
		}
	}
	presets := len(app.Presets)
	if len(briefs) == 0 {
		fmt.Fprintf(e.Stdout, "gmail tidy complete: presets=%d unread=0 trashed=0 marked_read=0 spam_added=0 left=0\n", presets)
		return 0, nil
	}
	verdicts, err := e.decide(briefs)
	if err != nil {
		return 0, err
	}
	trashed, marked, left := 0, 0, 0
	disposable := map[string][]string{}
	seenFrom := map[string]bool{}
	for _, brief := range briefs {
		verdict := verdicts[mailKey(brief.Preset, brief.ID)]
		if verdict.Action == "" {
			verdict.Action = "keep"
		}
		switch {
		case verdict.Action == "trash" && verdict.Confidence >= minAction:
			if err := services[brief.Preset].TrashOne(brief.ID); err != nil {
				return 0, err
			}
			trashed++
			if verdict.Disposable >= minDisposable && brief.From != "" {
				key := brief.Preset + "\x1f" + brief.From
				if !seenFrom[key] {
					seenFrom[key] = true
					disposable[brief.Preset] = append(disposable[brief.Preset], brief.From)
				}
			}
		case verdict.Action == "read" && verdict.Confidence >= minAction:
			if _, _, err := services[brief.Preset].Modify(brief.ID, nil, []string{"UNREAD"}); err != nil {
				return 0, err
			}
			marked++
		default:
			left++
		}
	}
	app, err = e.load()
	if err != nil {
		return 0, err
	}
	added := 0
	for _, preset := range app.Presets {
		froms := disposable[preset]
		if len(froms) == 0 {
			continue
		}
		n, err := e.addSpam(app, preset, froms)
		if err != nil {
			return 0, err
		}
		added += n
		app, err = e.load()
		if err != nil {
			return 0, err
		}
	}
	if added > 0 {
		if _, err := e.spamCleanAll(); err != nil {
			return 0, err
		}
	}
	fmt.Fprintf(e.Stdout, "gmail tidy complete: presets=%d unread=%d trashed=%d marked_read=%d spam_added=%d left=%d\n", presets, len(briefs), trashed, marked, added, left)
	return 0, nil
}

func (e Env) addSpam(app *config.App, preset string, froms []string) (int, error) {
	acct := app.Accounts[preset]
	items := config.NormalizeSpamSenderList(froms)
	if len(items) == 0 {
		return 0, nil
	}
	have := map[string]bool{}
	for _, sender := range acct.SpamSenders {
		have[sender] = true
	}
	n := 0
	for _, sender := range items {
		if !have[sender] {
			n++
		}
	}
	if err := config.UpdateSpamSenders(app.Path, preset, mergeUnique(acct.SpamSenders, items)); err != nil {
		return 0, err
	}
	return n, nil
}

func (e Env) decide(mail []mailBrief) (map[string]mailVerdict, error) {
	if e.Decide != nil {
		return e.Decide(mail)
	}
	key, err := e.openAIKey()
	if err != nil {
		return nil, err
	}
	return classifyMail(e.httpClient(), decisionsURL, key, mail)
}

func (e Env) openAIKey() (string, error) {
	if e.OpenAIKey != nil {
		key, err := e.OpenAIKey()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(key) == "" {
			return "", fmt.Errorf("OPENAI_API_KEY is required to classify unread mail")
		}
		return strings.TrimSpace(key), nil
	}
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
		return key, nil
	}
	if testing.Testing() {
		return "", fmt.Errorf("OPENAI_API_KEY is required to classify unread mail")
	}
	key, err := wiomOpenAIKey()
	if err != nil {
		return "", fmt.Errorf("OPENAI_API_KEY is required to classify unread mail: %s", err.Error())
	}
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is required to classify unread mail")
	}
	return strings.TrimSpace(key), nil
}

func wiomOpenAIKey() (string, error) {
	bin, err := exec.LookPath("c3po")
	if err != nil {
		return "", errors.New("c3po is not on PATH")
	}
	cmd := exec.Command(bin, "pass", wiomVault, wiomOpenAIItem)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = wiomOpenAIItem + " was not read from the Wiom vault"
		}
		return "", errors.New(msg)
	}
	key := strings.TrimSpace(stdout.String())
	if key == "" || strings.Contains(key, "\n") {
		return "", errors.New(wiomOpenAIItem + " was not read from the Wiom vault")
	}
	return key, nil
}

func (e Env) httpClient() *http.Client {
	if e.HTTP != nil {
		return e.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}
