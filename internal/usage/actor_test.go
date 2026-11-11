package usage

import "testing"

func clearActorEnv(t *testing.T) {
	t.Helper()
	t.Setenv("C3PO_ACTOR", "")
	for _, name := range agentEnv {
		t.Setenv(name, "")
	}
	t.Setenv("INVOCATION_ID", "")
	t.Setenv("SYSTEMD_EXEC_PID", "")
}

func withTerminal(t *testing.T, on bool) {
	t.Helper()
	old := terminal
	terminal = func() bool { return on }
	t.Cleanup(func() { terminal = old })
}

func TestActorOverrideWins(t *testing.T) {
	clearActorEnv(t)
	t.Setenv("C3PO_ACTOR", "service")
	t.Setenv("GROK_AGENT", "1")
	withTerminal(t, true)
	if Actor() != "service" {
		t.Fatalf("actor %s", Actor())
	}
}

func TestActorIgnoresBadOverride(t *testing.T) {
	clearActorEnv(t)
	t.Setenv("C3PO_ACTOR", "root")
	t.Setenv("GROK_AGENT", "1")
	withTerminal(t, true)
	if Actor() != "agent" {
		t.Fatalf("actor %s", Actor())
	}
}

func TestActorAgentBeatsTerminalAndSystemd(t *testing.T) {
	clearActorEnv(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("INVOCATION_ID", "unit")
	withTerminal(t, true)
	if Actor() != "agent" {
		t.Fatalf("actor %s", Actor())
	}
}

func TestActorHumanOnTerminal(t *testing.T) {
	clearActorEnv(t)
	t.Setenv("SYSTEMD_EXEC_PID", "9")
	withTerminal(t, true)
	if Actor() != "human" {
		t.Fatalf("actor %s", Actor())
	}
}

func TestActorServiceWithoutTerminal(t *testing.T) {
	clearActorEnv(t)
	t.Setenv("INVOCATION_ID", "gmail-timer")
	withTerminal(t, false)
	if Actor() != "service" {
		t.Fatalf("actor %s", Actor())
	}
	t.Setenv("INVOCATION_ID", "")
	if Actor() != "service" {
		t.Fatalf("pipe actor %s", Actor())
	}
}
