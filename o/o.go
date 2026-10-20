// Package o is the c3po tui o file navigator: a Vim-inspired terminal
// file browser with open, reveal, pick, save, and config commands.
package o

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/o/internal/app"
	"github.com/ryangerardwilson/c3po/o/internal/config"
	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
	opener "github.com/ryangerardwilson/c3po/o/internal/open"
)

// shellCDEnv names the file a shell wrapper reads to cd after exit.
const shellCDEnv = "O_SHELL_CD_FILE"

const (
	openUsage   = "Use: c3po tui o open <path> [<path> ...]"
	revealUsage = "Use: c3po tui o reveal <path>"
	pickUsage   = "Use: c3po tui o pick from <dir> [files <exts>|directories] [multiple]"
	saveUsage   = "Use: c3po tui o save from <dir> [extension <ext>]"
)

// runApp shows the navigator. Tests replace it so no terminal is needed.
var runApp = app.Run

type command func(args []string, stdout, stderr io.Writer) int

var commands = map[string]command{
	"open":   runOpen,
	"reveal": runReveal,
	"pick":   runPick,
	"save":   runSave,
	"config": runConfig,
}

// Run executes c3po tui o with the arguments after "o". The navigator
// itself draws on the terminal; help, results, and errors use the writers.
func Run(args []string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && isHelp(args[0]):
		fmt.Fprint(stdout, Help)
		return 0
	case len(args) == 1 && args[0] == "version":
		fmt.Fprintln(stdout, version.String())
		return 0
	case len(args) > 0 && (isHelp(args[0]) || args[0] == "version"):
		fmt.Fprintln(stderr, "Use help or version by itself.")
		return 1
	}
	return dispatch(args, stdout, stderr)
}

func isHelp(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func dispatch(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		start, _ := os.Getwd()
		return runNavigator(app.Config{StartDir: start, Mode: app.ModeBrowse}, stdout, stderr)
	}
	run, ok := commands[args[0]]
	if !ok {
		fmt.Fprintln(stderr, Usage)
		return 1
	}
	return run(args, stdout, stderr)
}

func runOpen(args []string, stdout, stderr io.Writer) int {
	paths := args[1:]
	if len(paths) == 0 {
		fmt.Fprintln(stderr, openUsage)
		return 1
	}
	targets := make([]string, 0, len(paths))
	for _, arg := range paths {
		target := fsnav.NormalizePath(arg)
		info, err := os.Stat(target)
		if err != nil {
			fmt.Fprintf(stderr, "Path not found: %s\n", target)
			return 1
		}
		if len(paths) == 1 && info.IsDir() {
			return runNavigator(app.Config{StartDir: target, Mode: app.ModeBrowse}, stdout, stderr)
		}
		if info.IsDir() {
			fmt.Fprintln(stderr, "Multiple open targets must all be files")
			return 1
		}
		targets = append(targets, target)
	}
	return opener.OpenFiles(targets)
}

func runReveal(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, revealUsage)
		return 1
	}
	target := fsnav.NormalizePath(args[1])
	start := target
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		start = filepath.Dir(target)
	}
	return runNavigator(app.Config{StartDir: start, RevealTarget: target, Mode: app.ModeBrowse}, stdout, stderr)
}

func runPick(args []string, stdout, stderr io.Writer) int {
	return runParsed(parsePick, args, stdout, stderr)
}

func runSave(args []string, stdout, stderr io.Writer) int {
	return runParsed(parseSave, args, stdout, stderr)
}

func runParsed(parse func([]string) (app.Config, error), args []string, stdout, stderr io.Writer) int {
	cfg, err := parse(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return runNavigator(cfg, stdout, stderr)
}

func runConfig(args []string, _, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, Usage)
		return 1
	}
	return openConfig(stderr)
}

func parsePick(args []string) (app.Config, error) {
	if len(args) < 3 || args[1] != "from" {
		return app.Config{}, errors.New(pickUsage)
	}
	cfg := app.Config{StartDir: fsnav.NormalizePath(args[2]), Mode: app.ModePick, Allowed: "any"}
	rest, ok := parseAllowed(&cfg, args[3:])
	if !ok {
		return app.Config{}, errors.New(pickUsage)
	}
	if len(rest) > 0 && rest[0] == "multiple" {
		cfg.Multi = true
		rest = rest[1:]
	}
	if len(rest) != 0 {
		return app.Config{}, errors.New(pickUsage)
	}
	return cfg, nil
}

// parseAllowed reads an optional "files [<exts>]" or "directories" and
// returns the words after it.
func parseAllowed(cfg *app.Config, rest []string) ([]string, bool) {
	if len(rest) == 0 {
		return rest, true
	}
	switch rest[0] {
	case "files":
		cfg.Allowed = "file"
		rest = rest[1:]
		if len(rest) > 0 && rest[0] != "multiple" {
			cfg.Extensions = parseExtensions(rest[0])
			rest = rest[1:]
		}
		return rest, true
	case "directories":
		cfg.Allowed = "dir"
		return rest[1:], true
	case "multiple":
		return rest, true
	}
	return rest, false
}

func parseSave(args []string) (app.Config, error) {
	if len(args) < 3 || args[1] != "from" {
		return app.Config{}, errors.New(saveUsage)
	}
	cfg := app.Config{StartDir: fsnav.NormalizePath(args[2]), Mode: app.ModeSave, Allowed: "any"}
	rest := args[3:]
	if len(rest) == 0 {
		return cfg, nil
	}
	if len(rest) != 2 || rest[0] != "extension" {
		return app.Config{}, errors.New(saveUsage)
	}
	cfg.Extensions = parseExtensions(rest[1])
	return cfg, nil
}

// Pick opens the navigator at startDir. l enters a directory. enter chooses
// the highlighted file or directory. An empty path means the operator quit.
func Pick(startDir string, stderr io.Writer) (string, int) {
	cfg := app.Config{
		StartDir: fsnav.NormalizePath(startDir),
		Mode:     app.ModePick,
		Allowed:  "any",
		Descend:  true,
	}
	var stdout bytes.Buffer
	if code := runNavigator(cfg, &stdout, stderr); code != 0 {
		return "", code
	}
	return strings.TrimSpace(stdout.String()), 0
}

func runNavigator(cfg app.Config, stdout, stderr io.Writer) int {
	result, code := runApp(cfg, stderr)
	if code != 0 {
		return code
	}
	if result.ShellCD != "" {
		writeShellCD(result.ShellCD)
	}
	if len(result.Selections) > 0 {
		_ = fsnav.WriteSelectionCache(result.Selections)
		for _, selection := range result.Selections {
			fmt.Fprintln(stdout, selection)
		}
	}
	return 0
}

func writeShellCD(dir string) {
	if path := strings.TrimSpace(os.Getenv(shellCDEnv)); path != "" {
		_ = os.WriteFile(path, []byte(dir+"\n"), 0o644)
	}
}

func openConfig(stderr io.Writer) int {
	if err := config.Ensure(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cmd := exec.Command(opener.Editor(), config.Path())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	fmt.Fprintln(stderr, err)
	return 1
}

func parseExtensions(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' })
	extensions := []string{}
	for _, part := range parts {
		part = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(part)), ".")
		if part != "" {
			extensions = append(extensions, part)
		}
	}
	return extensions
}
