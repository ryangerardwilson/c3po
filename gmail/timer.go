package gmail

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

func (e Env) timer(params []string) (int, error) {
	if len(params) != 1 || !word(params[0], "install", "disable", "status") {
		return 0, errs.Usage("Use: c3po gmail timer install|disable|status")
	}
	switch params[0] {
	case "install":
		return e.installTimer()
	case "disable":
		return e.disableTimer()
	default:
		return e.timerStatus()
	}
}

func (e Env) installTimer() (int, error) {
	if err := e.writeTimerUnits(); err != nil {
		return 0, err
	}
	if _, err := e.systemctl("daemon-reload"); err != nil {
		return 0, err
	}
	if _, err := e.systemctl("enable", "gmail.timer"); err != nil {
		return 0, err
	}
	if _, err := e.systemctl("restart", "gmail.timer"); err != nil {
		return 0, err
	}
	fmt.Fprintln(e.Stdout, "timer enabled: gmail.timer")
	return 0, nil
}

func (e Env) disableTimer() (int, error) {
	if err := e.writeTimerUnits(); err != nil {
		return 0, err
	}
	if _, err := e.systemctl("disable", "--now", "gmail.timer"); err != nil {
		return 0, err
	}
	fmt.Fprintln(e.Stdout, "timer disabled: gmail.timer")
	return 0, nil
}

func (e Env) timerStatus() (int, error) {
	out, err := e.systemctl("status", "gmail.timer")
	if err != nil {
		return 0, err
	}
	fmt.Fprintln(e.Stdout, strings.TrimSpace(out))
	return 0, nil
}

func (e Env) timerState() string {
	out, err := e.systemctlLoose("is-active", "gmail.timer")
	if err != nil || strings.TrimSpace(out) == "" {
		return "unknown"
	}
	return strings.TrimSpace(out)
}

func (e Env) systemctl(args ...string) (string, error) {
	if e.Systemctl != nil {
		out, err := e.Systemctl(args)
		if err != nil {
			msg := strings.TrimSpace(out)
			if msg == "" {
				msg = err.Error()
			}
			return out, &errs.Systemctl{Msg: msg}
		}
		return out, nil
	}
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return string(out), &errs.Systemctl{Msg: msg}
	}
	return string(out), nil
}

func (e Env) systemctlLoose(args ...string) (string, error) {
	if e.Systemctl != nil {
		out, _ := e.Systemctl(args)
		return out, nil
	}
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	out, _ := cmd.CombinedOutput()
	return string(out), nil
}

func (e Env) writeTimerUnits() error {
	dir := filepath.Join(e.home(), ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	script := "if " + gmailRun() + "; then " + notifyCall("Hourly spam clean finished successfully", false) + `; else rc=$?; ` + notifyCall("Hourly spam clean failed", true) + `; exit "$rc"; fi`
	service := strings.Join([]string{
		"[Unit]",
		"Description=gmail spam clean all presets",
		"",
		"[Service]",
		"Type=oneshot",
		"ExecStart=/usr/bin/env bash -lc " + shellQuote(script),
		"",
	}, "\n")
	timer := strings.Join([]string{
		"[Unit]",
		"Description=Run gmail spam clean hourly",
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
	if err := os.WriteFile(filepath.Join(dir, "gmail.service"), []byte(service), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "gmail.timer"), []byte(timer), 0o644)
}

func gmailRun() string {
	return shellQuote(c3poBin()) + " gmail spam clean"
}

func c3poBin() string {
	if exe, err := os.Executable(); err == nil && filepath.Base(exe) == "c3po" {
		return exe
	}
	if path, err := exec.LookPath("c3po"); err == nil {
		return path
	}
	return "c3po"
}

// notifyCall is the unit's toast. It runs c3po notify from the same binary
// as the spam clean. Only a failure passes --urgency critical.
func notifyCall(body string, critical bool) string {
	flag := ""
	if critical {
		flag = "--urgency critical "
	}
	return shellQuote(c3poBin()) + " notify " + flag + "heading 'gmail' body " + shellQuote(body)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`;&|()<>") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
