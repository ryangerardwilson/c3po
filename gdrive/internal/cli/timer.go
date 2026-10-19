package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/paths"
)

func unitName() string { return "gdrive" }

func c3poBin() string {
	if exe, err := os.Executable(); err == nil && filepath.Base(exe) == "c3po" {
		return exe
	}
	if path, err := exec.LookPath("c3po"); err == nil {
		return path
	}
	return "c3po"
}

func syncRunCommand() string {
	return shellQuote(c3poBin()) + " gdrive sync run"
}

func notificationShellFunction() string {
	return strings.Join([]string{
		"notify() {",
		`summary="$1";`,
		`body="${2:-}";`,
		`urgency="${3:-normal}";`,
		`qs="${XDG_CONFIG_HOME:-$HOME/.config}/quickshell/omarchy-bar";`,
		`if command -v quickshell >/dev/null 2>&1 && quickshell ipc -p "$qs" call bar notify "$summary" "$body" "$urgency" >/dev/null 2>&1; then return 0; fi;`,
		`if command -v notify-send >/dev/null 2>&1; then notify-send -a "$summary" -u "$urgency" "$summary" "$body" || true; fi;`,
		"};",
	}, " ")
}

func buildTimerServiceScript(runCommand string) string {
	return strings.Join([]string{
		notificationShellFunction(),
		"notify 'gdrive' 'Hourly backup started' normal;",
		"if " + runCommand + "; then",
		"notify 'gdrive' 'Hourly backup finished successfully' normal;",
		"else",
		"rc=$?;",
		"notify 'gdrive' 'Hourly backup failed' critical;",
		`exit "$rc";`,
		"fi",
	}, " ")
}

func writeTimerUnits() error {
	if err := paths.EnsureDirs(); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return fmt.Errorf("home directory is not set")
	}
	systemdDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(systemdDir, 0o755); err != nil {
		return err
	}
	serviceBody := strings.Join([]string{
		"[Unit]",
		"Description=gdrive sync all presets",
		"",
		"[Service]",
		"Type=oneshot",
		"ExecStart=/usr/bin/env bash -lc " + shellQuote(buildTimerServiceScript(syncRunCommand())),
		"",
	}, "\n")
	timerBody := strings.Join([]string{
		"[Unit]",
		"Description=Run gdrive hourly",
		"",
		"[Timer]",
		"OnBootSec=5m",
		"OnActiveSec=5m",
		"OnUnitActiveSec=1h",
		"Persistent=true",
		"",
		"[Install]",
		"WantedBy=timers.target",
		"",
	}, "\n")
	servicePath := filepath.Join(systemdDir, unitName()+".service")
	timerPath := filepath.Join(systemdDir, unitName()+".timer")
	if err := os.WriteFile(servicePath, []byte(serviceBody), 0o644); err != nil {
		return err
	}
	return os.WriteFile(timerPath, []byte(timerBody), 0o644)
}

func systemctlUser(args ...string) ([]byte, error) {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	return cmd.CombinedOutput()
}

func installTimer(stdout io.Writer) error {
	if err := writeTimerUnits(); err != nil {
		return err
	}
	steps := [][]string{{"daemon-reload"}, {"enable", unitName() + ".timer"}, {"restart", unitName() + ".timer"}}
	for _, args := range steps {
		if out, err := systemctlUser(args...); err != nil {
			return fmt.Errorf("systemctl failed: %s", strings.TrimSpace(string(out)))
		}
	}
	fmt.Fprintf(stdout, "timer enabled: %s.timer\n", unitName())
	return nil
}

func disableTimer(stdout io.Writer) error {
	if err := writeTimerUnits(); err != nil {
		return err
	}
	out, err := systemctlUser("disable", "--now", unitName()+".timer")
	if err != nil {
		return fmt.Errorf("systemctl failed: %s", strings.TrimSpace(string(out)))
	}
	fmt.Fprintf(stdout, "timer disabled: %s.timer\n", unitName())
	return nil
}

func timerStatus(stdout io.Writer) error {
	out, err := systemctlUser("status", unitName()+".timer")
	if err != nil {
		return fmt.Errorf("systemctl failed: %s", strings.TrimSpace(string(out)))
	}
	fmt.Fprintln(stdout, strings.TrimSpace(string(out)))
	return nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if !strings.ContainsAny(value, " \t\n'\"\\$`!*?[]{}();&|<>") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
