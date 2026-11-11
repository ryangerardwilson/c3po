package usage

import "os"

// agentEnv marks a process an agent started. An empty value does not count.
var agentEnv = []string{
	"GROK_AGENT",
	"CLAUDE_CODE",
	"CLAUDECODE",
	"CODEX_SANDBOX",
	"CODEX_THREAD_ID",
	"CURSOR_AGENT",
	"CURSOR_TRACE_ID",
}

// terminal reports whether stdin is a terminal. Tests replace it.
var terminal = func() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Actor classifies this process as human, agent, or service.
// C3PO_ACTOR wins when it is exactly one of those three words.
// An agent marker wins over a terminal and over systemd, because an agent
// shell on this machine can have both.
// A terminal with no agent marker is a human.
// Anything else is a service: a systemd timer, a script, or a pipe.
func Actor() string {
	switch os.Getenv("C3PO_ACTOR") {
	case "human", "agent", "service":
		return os.Getenv("C3PO_ACTOR")
	}
	if agentMarked() {
		return "agent"
	}
	if terminal() {
		return "human"
	}
	return "service"
}

func agentMarked() bool {
	for _, name := range agentEnv {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}
