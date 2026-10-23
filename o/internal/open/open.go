// Package open launches files in the user's editor or desktop opener.
package open

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
)

// Editor is $VISUAL, then $EDITOR, then vim.
func Editor() string {
	if value := os.Getenv("VISUAL"); value != "" {
		return value
	}
	if value := os.Getenv("EDITOR"); value != "" {
		return value
	}
	return "vim"
}

// OpenFiles opens every path in the editor when all of them are text.
// A workbook opens in the xl screen, in the foreground. Anything else
// is handed to a detached opener. The return value is the first
// foreground failure, or 0 when every file was handed off.
func OpenFiles(paths []string) int {
	if len(paths) == 0 {
		return 0
	}
	if allTextLike(paths) {
		cmd := exec.Command(Editor(), paths...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return exitCode(cmd.Run())
	}
	code := 0
	for _, path := range paths {
		if IsWorkbook(path) {
			cmd := WorkbookCommand(path)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if c := exitCode(cmd.Run()); c != 0 && code == 0 {
				code = c
			}
			continue
		}
		_ = detached(path)
	}
	return code
}

// detached starts a non-workbook opener. Tests replace it.
var detached = OpenDetached

// IsWorkbook reports an xlsx path. The suffix match ignores case.
func IsWorkbook(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".xlsx")
}

// WorkbookCommand starts the xl screen on one workbook. Tests replace it.
var WorkbookCommand = defaultWorkbookCommand

func defaultWorkbookCommand(path string) *exec.Cmd {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		exe = "c3po"
	}
	return exec.Command(exe, "tui", "xl", path)
}

// EditorCommand is the foreground editor command for one file. The
// caller attaches the terminal before running it.
func EditorCommand(path string) *exec.Cmd {
	return exec.Command(Editor(), path)
}

// OpenDetached starts the opener for one file without waiting for it.
// Text-like files use the editor and everything else uses xdg-open.
func OpenDetached(path string) error {
	command := "xdg-open"
	if fsnav.IsTextLike(path) {
		command = Editor()
	}
	cmd := exec.Command(command, path)
	cmd.Dir = filepath.Dir(path)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func allTextLike(paths []string) bool {
	for _, path := range paths {
		if !fsnav.IsTextLike(path) {
			return false
		}
	}
	return true
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}
