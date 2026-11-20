package notify

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sent struct {
	title, body, urgency string
	called               bool
}

func (s *sent) send(err error) sender {
	return func(title, body, urgency string) error {
		s.title, s.body, s.urgency, s.called = title, body, urgency, true
		return err
	}
}

func TestParseMessage(t *testing.T) {
	title, body, err := parseMessage([]string{"heading", "gdrive", "body", "backup finished"})
	if err != nil {
		t.Fatal(err)
	}
	if title != "gdrive" || body != "backup finished" {
		t.Fatalf("unexpected labeled message: %q %q", title, body)
	}

	title, body, err = parseMessage([]string{"gdrive", "backup finished"})
	if err != nil {
		t.Fatal(err)
	}
	if title != "gdrive" || body != "backup finished" {
		t.Fatalf("unexpected legacy message: %q %q", title, body)
	}

	title, body, err = parseMessage([]string{"done"})
	if err != nil {
		t.Fatal(err)
	}
	if title != "done" || body != "" {
		t.Fatalf("unexpected message: %q %q", title, body)
	}
}

func TestParseMessageRejectsLabeledWithoutHeading(t *testing.T) {
	_, _, err := parseMessage([]string{"body", "backup finished"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseMessageRejectsBadInput(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"heading", "", "body", "x"},
		{"  "},
		{"heading", "gdrive", "body", "x", "extra"},
		{"heading"},
	} {
		if _, _, err := parseMessage(args); err == nil {
			t.Fatalf("expected error for %q", args)
		}
	}
}

func TestNormalizeUrgency(t *testing.T) {
	if normalizeUrgency("CRITICAL") != "critical" {
		t.Fatal("expected critical")
	}
	if normalizeUrgency("low") != "low" {
		t.Fatal("expected low")
	}
	if normalizeUrgency("weird") != "normal" {
		t.Fatal("expected normal default")
	}
}

func TestOmarchyShellArgs(t *testing.T) {
	got := omarchyShellArgs("gmail", "morning inbox review finished", "normal")
	if len(got) != 5 || got[0] != "notify" || got[1] != "show" || got[2] != "gmail" || got[3] != "morning inbox review finished" || got[4] != "normal" {
		t.Fatalf("unexpected args: %#v", got)
	}
}

func TestRunSendsSilently(t *testing.T) {
	var s sent
	var out, errb bytes.Buffer
	code := run([]string{"-u", "critical", "heading", "gdrive", "body", "backup failed"}, &out, &errb, s.send(nil))
	if code != 0 || out.Len() != 0 || errb.Len() != 0 {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if s.title != "gdrive" || s.body != "backup failed" || s.urgency != "critical" {
		t.Fatalf("sent %+v", s)
	}
}

func TestRunUnknownUrgencyIsNormal(t *testing.T) {
	var s sent
	var out, errb bytes.Buffer
	code := run([]string{"--urgency", "loud", "heading", "x"}, &out, &errb, s.send(nil))
	if code != 0 || s.urgency != "normal" {
		t.Fatalf("code %d urgency %q", code, s.urgency)
	}
}

func TestRunRefusesBadArguments(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"body", "only"},
		{"heading", ""},
		{"heading", "x", "body", "y", "z"},
		{"--urgency"},
		{"--loud", "x"},
	} {
		var s sent
		var out, errb bytes.Buffer
		code := run(args, &out, &errb, s.send(nil))
		if code != 2 || s.called || !strings.Contains(errb.String(), "usage: c3po notify") {
			t.Fatalf("%q code %d called %v err %q", args, code, s.called, errb.String())
		}
	}
}

func TestRunReportsDeliveryFailure(t *testing.T) {
	var s sent
	var out, errb bytes.Buffer
	code := run([]string{"heading", "x"}, &out, &errb, s.send(errors.New("omarchy-shell toast unavailable: boom")))
	if code != 1 || !strings.Contains(errb.String(), "omarchy-shell") {
		t.Fatalf("code %d err %q", code, errb.String())
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"-u", "low", "--help"}} {
		var s sent
		var out, errb bytes.Buffer
		code := run(args, &out, &errb, s.send(nil))
		if code != 0 || out.String() != Help || s.called {
			t.Fatalf("%q code %d called %v\n%s", args, code, s.called, out.String())
		}
	}
	if !strings.Contains(Help, "c3po notify heading") {
		t.Fatalf("help does not name c3po notify\n%s", Help)
	}
	var out, errb bytes.Buffer
	code := run([]string{"version"}, &out, &errb, nil)
	if code != 0 || !strings.HasPrefix(out.String(), "c3po ") {
		t.Fatalf("code %d out %q", code, out.String())
	}
}

func fakeBin(t *testing.T, dir, name, script string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDeliverUsesOmarchyShell(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fakeBin(t, dir, "omarchy-shell", `printf '%s|' "$OMARCHY_PATH" "$@" > "`+log+`"`+"\n")
	t.Setenv("PATH", dir)
	t.Setenv("OMARCHY_PATH", "")
	if err := deliver("gmail", "done", "low"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(log)
	if string(got) != "/usr/share/omarchy|notify|show|gmail|done|low|" {
		t.Fatalf("args %q", got)
	}
}

func TestDeliverFallsBackToQuickshell(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fakeBin(t, dir, "omarchy-shell", "echo down >&2\nexit 3\n")
	fakeBin(t, dir, "quickshell", `printf '%s|' "$@" > "`+log+`"`+"\n")
	t.Setenv("PATH", dir)
	t.Setenv("NOTIFY_QUICKSHELL_BAR", "/tmp/bar")
	if err := deliver("gmail", "done", "critical"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(log)
	if string(got) != "ipc|-p|/tmp/bar|call|bar|notify|gmail|done|critical|" {
		t.Fatalf("args %q", got)
	}
}

func TestDeliverNamesTheOmarchyShellError(t *testing.T) {
	dir := t.TempDir()
	fakeBin(t, dir, "omarchy-shell", "echo down >&2\nexit 3\n")
	fakeBin(t, dir, "quickshell", "exit 1\n")
	t.Setenv("PATH", dir)
	err := deliver("gmail", "done", "normal")
	if err == nil || !strings.Contains(err.Error(), "omarchy-shell toast unavailable") || !strings.Contains(err.Error(), "down") {
		t.Fatalf("err %v", err)
	}
}
