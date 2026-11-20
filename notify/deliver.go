package notify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultBarPath     = ".config/quickshell/omarchy-bar"
	defaultOmarchyPath = "/usr/share/omarchy"
)

// deliver shows the toast through omarchy-shell and falls back to the
// quickshell bar. The error names the omarchy-shell failure.
func deliver(title, body, urgency string) error {
	shellErr := omarchyShellNotify(title, body, urgency)
	if shellErr == nil {
		return nil
	}
	if err := quickshellNotify(title, body, urgency); err == nil {
		return nil
	}
	return fmt.Errorf("omarchy-shell toast unavailable: %w", shellErr)
}

func omarchyShellArgs(title, body, urgency string) []string {
	return []string{"notify", "show", title, body, urgency}
}

func omarchyShellNotify(title, body, urgency string) error {
	bin, err := exec.LookPath("omarchy-shell")
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, omarchyShellArgs(title, body, urgency)...)
	cmd.Env = os.Environ()
	if strings.TrimSpace(os.Getenv("OMARCHY_PATH")) == "" {
		cmd.Env = append(cmd.Env, "OMARCHY_PATH="+defaultOmarchyPath)
	}
	return runQuiet(cmd)
}

func barPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("NOTIFY_QUICKSHELL_BAR")); configured != "" {
		return configured, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultBarPath), nil
}

func quickshellNotify(title, body, urgency string) error {
	bin, err := exec.LookPath("quickshell")
	if err != nil {
		return err
	}
	bar, err := barPath()
	if err != nil {
		return err
	}
	return runQuiet(exec.Command(bin, "ipc", "-p", bar, "call", "bar", "notify", title, body, urgency))
}

func runQuiet(cmd *exec.Cmd) error {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
