package desk

import (
	"errors"
	"fmt"
	"os"
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
	out, err := e.runSystemctl(args)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return out, &errs.Systemctl{Msg: msg}
	}
	return out, nil
}

func (e Env) systemctlLoose(args ...string) (string, error) {
	out, _ := e.runSystemctl(args)
	return out, nil
}

func (e Env) runSystemctl(args []string) (string, error) {
	if e.Systemctl == nil {
		return "", errors.New("systemctl is not available")
	}
	return e.Systemctl(args)
}

func (e Env) writeTimerUnits() error {
	dir := filepath.Join(e.home(), ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	bin := e.c3poBin()
	script := "if " + gmailRun(bin) + "; then " + notifyCall(bin, "Hourly spam clean finished successfully", false) + `; else rc=$?; ` + notifyCall(bin, "Hourly spam clean failed", true) + `; exit "$rc"; fi`
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

func gmailRun(bin string) string {
	return shellQuote(bin) + " gmail spam clean"
}

func (e Env) c3poBin() string {
	if e.C3po != "" {
		return e.C3po
	}
	return "c3po"
}

// notifyCall is the unit's toast. It runs c3po notify from the same binary
// as the spam clean. Only a failure passes --urgency critical.
func notifyCall(bin, body string, critical bool) string {
	flag := ""
	if critical {
		flag = "--urgency critical "
	}
	return shellQuote(bin) + " notify " + flag + "heading 'gmail' body " + shellQuote(body)
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
