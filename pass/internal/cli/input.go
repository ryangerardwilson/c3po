package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var itemFlags = map[string]bool{"title": true, "category": true, "username": true, "template": true}

// parseItemFlags reads --title, --category, --username, and --template
// as --name value or --name=value. Any other flag fails, so a secret
// cannot be passed as an argument.
func parseItemFlags(args []string) ([]string, map[string]string, error) {
	flags := map[string]string{}
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") {
			positionals = append(positionals, arg)
			continue
		}
		next, err := takeFlag(args, i, flags)
		if err != nil {
			return nil, nil, err
		}
		i = next
	}
	return positionals, flags, nil
}

// takeFlag stores the flag at args[i] and returns the index of its last
// argument.
func takeFlag(args []string, i int, flags map[string]string) (int, error) {
	name, value, hasValue := strings.Cut(strings.TrimPrefix(args[i], "--"), "=")
	if !itemFlags[name] {
		return i, fmt.Errorf("Unknown flag --%s.", name)
	}
	if !hasValue {
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			return i, fmt.Errorf("--%s needs a value.", name)
		}
		i++
		value = args[i]
	}
	if _, exists := flags[name]; exists {
		return i, fmt.Errorf("--%s was given twice.", name)
	}
	if strings.TrimSpace(value) == "" {
		return i, fmt.Errorf("--%s must not be empty.", name)
	}
	flags[name] = strings.TrimSpace(value)
	return i, nil
}

// readSecret reads the secret from stdin. A terminal is not read, and
// found is false. One trailing newline is dropped.
func (r Runner) readSecret() (string, bool, error) {
	data, terminal, err := r.readInput()
	if err != nil || terminal {
		return "", false, err
	}
	secret := strings.TrimSuffix(string(data), "\n")
	secret = strings.TrimSuffix(secret, "\r")
	if secret == "" {
		return "", false, nil
	}
	return secret, true, nil
}

// readInput reads all of stdin unless it is a terminal.
func (r Runner) readInput() ([]byte, bool, error) {
	in := r.input()
	if file, ok := in.(*os.File); ok {
		info, err := file.Stat()
		if err == nil && info.Mode()&os.ModeCharDevice != 0 {
			return nil, true, nil
		}
	}
	data, err := io.ReadAll(in)
	return data, false, err
}

// templateBytes reads item JSON from path, or from stdin when path is -.
func (r Runner) templateBytes(path string) ([]byte, bool) {
	if path != "-" {
		data, err := os.ReadFile(path)
		if err != nil {
			r.fail(fmt.Sprintf("Unable to read template %s.", path))
			return nil, false
		}
		return data, true
	}
	data, terminal, err := r.readInput()
	if err != nil {
		r.fail("Unable to read the template.")
		return nil, false
	}
	if terminal || len(bytes.TrimSpace(data)) == 0 {
		r.fail("Pass the template on stdin.")
		return nil, false
	}
	return data, true
}

// choose prompts for a match number from 1 to count.
func (r Runner) choose(count int) (int, error) {
	_, _ = fmt.Fprintf(r.Out, "Select a match [1-%d]: ", count)
	line, _ := bufio.NewReader(r.input()).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, errors.New("No selection was entered.")
	}
	number, err := strconv.Atoi(line)
	if err != nil {
		return 0, errors.New("Selection must be a number.")
	}
	if number < 1 || number > count {
		return 0, errors.New("Selection is out of range.")
	}
	return number - 1, nil
}
