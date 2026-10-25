package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// applyFileCommand runs mv, rm, cp, touch, or mkdir in dir. rm may take
// -rf. Any other command or flag is refused before a process starts.
func applyFileCommand(dir, line string) (status, selectPath string, err error) {
	fields, err := splitCommand(line)
	if err != nil {
		return "", "", err
	}
	if len(fields) == 0 {
		return "", "", errors.New("use mv, rm, cp, touch, or mkdir")
	}
	name := fields[0]
	switch name {
	case "mv", "rm", "cp", "touch", "mkdir":
	default:
		return "", "", errors.New("use mv, rm, cp, touch, or mkdir")
	}
	args := fields[1:]
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			if name == "rm" && arg == "-rf" {
				continue
			}
			return "", "", fmt.Errorf("flag %s is not allowed", arg)
		}
		paths = append(paths, arg)
	}
	need := 1
	if name == "cp" || name == "mv" {
		need = 2
	}
	if len(paths) < need {
		if need == 2 {
			return "", "", fmt.Errorf("%s needs two paths", name)
		}
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

// shellLine returns the shell command when line starts with !.
// The caller trims the line first. Space after the bang is ignored.
// A command that is > and one path is a file redirect, handled by the caller.
func shellLine(line string) (string, bool) {
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

// redirectPath returns the single path of a bare > redirect.
// >> and any extra word stay with the shell.
func redirectPath(command string) (string, bool) {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, ">>") || !strings.HasPrefix(command, ">") {
		return "", false
	}
	rest := strings.TrimSpace(command[len(">"):])
	fields, err := splitCommand(rest)
	if err != nil || len(fields) != 1 || fields[0] == "" {
		return "", false
	}
	return fields[0], true
}

// applyRedirect creates path or empties it. A relative path is inside dir.
func applyRedirect(dir, path string) (string, error) {
	target := path
	if !filepath.IsAbs(path) {
		target = filepath.Join(dir, path)
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return target, nil
}
