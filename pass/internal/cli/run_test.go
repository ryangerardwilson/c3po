package cli

import (
	"strings"
	"testing"
)

func TestNoArgumentsAndHelpWriteHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		runner, stdout, _, _ := newRunner(t, &fakeOp{})
		if rc := runner.Run(args); rc != 0 || stdout.String() != "help text\n" {
			t.Fatalf("%v rc = %d stdout = %q", args, rc, stdout.String())
		}
	}
}

func TestHelpWithAVaultArgumentIsAnError(t *testing.T) {
	fake := &fakeOp{}
	runner, stdout, stderr, _ := newRunner(t, fake)
	rc := runner.Run([]string{"help", "github"})
	if rc != 1 || stderr.String() != "Use help by itself.\n" || stdout.Len() != 0 {
		t.Fatalf("rc = %d stdout = %q stderr = %q", rc, stdout.String(), stderr.String())
	}
	if len(fake.calls) != 0 {
		t.Fatal("help called op")
	}
}

func TestVersionPrintsTheGivenVersion(t *testing.T) {
	runner, stdout, _, _ := newRunner(t, &fakeOp{})
	if rc := runner.Run([]string{"version"}); rc != 0 || stdout.String() != "9.9.9\n" {
		t.Fatalf("rc = %d stdout = %q", rc, stdout.String())
	}
	runner, _, stderr, _ := newRunner(t, &fakeOp{})
	if rc := runner.Run([]string{"version", "x"}); rc != 1 || stderr.String() != "Use version by itself.\n" {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
}

func TestUpgradeIsRemoved(t *testing.T) {
	for _, args := range [][]string{{"upgrade"}, {"upgrade", "x"}} {
		fake := &fakeOp{}
		runner, _, stderr, _ := newRunner(t, fake)
		if rc := runner.Run(args); rc != 1 || stderr.String() != upgradeRemoved+"\n" {
			t.Fatalf("%v rc = %d stderr = %q", args, rc, stderr.String())
		}
		if len(fake.calls) != 0 {
			t.Fatal("upgrade called op")
		}
	}
}

func TestOneArgumentIsUsage(t *testing.T) {
	for _, args := range [][]string{{"open"}, {"config"}, {"ls"}, {"-l", "x"}} {
		runner, _, stderr, _ := newRunner(t, &fakeOp{})
		rc := runner.Run(args)
		if rc != 1 {
			t.Fatalf("%v rc = %d", args, rc)
		}
		if !strings.Contains(stderr.String(), "Use: c3po pass <vault> <search>") {
			t.Fatalf("%v stderr = %q", args, stderr.String())
		}
	}
}

func TestListWithExtraArgumentsIsUsage(t *testing.T) {
	fake := &fakeOp{}
	runner, _, stderr, _ := newRunner(t, fake)
	if rc := runner.Run([]string{"Root", "-l", "x"}); rc != 1 || stderr.String() != "Use: c3po pass <vault> -l\n" {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
	if len(fake.calls) != 0 {
		t.Fatal("usage called op")
	}
}

func TestMissingOpIsReported(t *testing.T) {
	runner, _, stderr, _ := newRunner(t, &fakeOp{missing: true})
	if rc := runner.Run([]string{"Root", "github"}); rc != 1 || stderr.String() != "op is required.\n" {
		t.Fatalf("rc = %d stderr = %q", rc, stderr.String())
	}
}

func TestEmptyVaultIsRejected(t *testing.T) {
	cases := map[string][]string{
		"Vault and search must not be empty.\n": {" ", "github"},
		"Vault must not be empty.\n":            {" ", "-l"},
	}
	for want, args := range cases {
		runner, _, stderr, _ := newRunner(t, &fakeOp{})
		if rc := runner.Run(args); rc != 1 || stderr.String() != want {
			t.Fatalf("%q rc = %d stderr = %q", args, rc, stderr.String())
		}
	}
}
