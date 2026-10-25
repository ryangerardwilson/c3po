package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// applyFileCommand runs touch, mkdir, or rm in dir. Any other command or
// flag is refused before a process starts. mkdir may take -p.
func applyFileCommand(dir, line string) (status, selectPath string, err error) {
	fields, err := splitCommand(line)
	if err != nil {
		return "", "", err
	}
	if len(fields) == 0 {
		return "", "", errors.New("use touch, mkdir, or rm")
	}
	name := fields[0]
	switch name {
	case "touch", "mkdir", "rm":
	default:
		return "", "", errors.New("use touch, mkdir, or rm")
	}
	args := fields[1:]
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			if name == "mkdir" && arg == "-p" {
				continue
			}
			return "", "", fmt.Errorf("flag %s is not allowed", arg)
		}
		paths = append(paths, arg)
	}
	if len(paths) == 0 {
		return "", "", fmt.Errorf("%s needs a path", name)
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, runErr := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	target := paths[len(paths)-1]
	if !filepath.IsAbs(target) {
		target = filepath.Join(dir, target)
	}
	if runErr != nil {
		if text == "" {
			text = runErr.Error()
		}
		return "", "", errors.New(text)
	}
	return strings.TrimSpace(line), target, nil
}

func splitCommand(line string) ([]string, error) {
	var fields []string
	var b strings.Builder
	var quote rune
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t':
			if b.Len() > 0 {
				fields = append(fields, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, errors.New("unclosed quote")
	}
	if b.Len() > 0 {
		fields = append(fields, b.String())
	}
	return fields, nil
}

// colonShell is the shell command in a colon line that starts with !.
// The bang and any space around the command are not part of it.
func colonShell(line string) (string, bool) {
	if !strings.HasPrefix(line, "!") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "!")), true
}

// shellCommand runs line in the user's shell with dir as its working
// directory. The caller hands it the terminal.
func shellCommand(dir, line string) *exec.Cmd {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	command := exec.Command(shell, "-c", line)
	command.Dir = dir
	return command
}
