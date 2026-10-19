package gmail

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

func interactive(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func (e Env) ask(prompt string) (string, error) {
	fmt.Fprint(e.Stdout, prompt)
	if e.Ask != nil {
		return e.Ask(prompt)
	}
	if !interactive(e.Stdin) {
		return "", nil
	}
	s := bufio.NewScanner(e.Stdin)
	if !s.Scan() {
		return "", nil
	}
	return s.Text(), nil
}

func (e Env) editorCompose(from, signature string, withTo bool) (compose, error) {
	body := editorTemplate(from, signature, withTo)
	f, err := os.CreateTemp("", "gmail-edit-*.txt")
	if err != nil {
		return compose{}, err
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		return compose{}, err
	}
	f.Close()
	if err := e.runEditor(path); err != nil {
		return compose{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return compose{}, err
	}
	return parseTemplate(string(raw)), nil
}

func (e Env) runEditor(path string) error {
	if e.Editor != nil {
		return e.Editor(path)
	}
	cmd := os.Getenv("VISUAL")
	if cmd == "" {
		cmd = os.Getenv("EDITOR")
	}
	if strings.TrimSpace(cmd) == "" {
		cmd = "vim"
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return errs.Usage("Editor command is empty. Set VISUAL or EDITOR.")
	}
	proc := exec.Command(parts[0], append(parts[1:], path)...)
	proc.Stdin, proc.Stdout, proc.Stderr = e.Stdin, e.Stdout, e.Stderr
	if err := proc.Run(); err != nil {
		return errs.Usage("Editor exited with code %s. Message not sent.", err.Error())
	}
	return nil
}

func editorTemplate(from, signature string, withTo bool) string {
	lines := []string{"From: " + from}
	if withTo {
		lines = append(lines, "To:", "Subject:")
	}
	lines = append(lines, "CC:", "BCC:", `Attachments: ""`, "Body:", "", "-- \n"+strings.TrimSpace(signature), "")
	return strings.Join(lines, "\n")
}

func parseTemplate(content string) compose {
	var out compose
	inBody := false
	var body []string
	for _, line := range strings.Split(content, "\n") {
		if inBody {
			body = append(body, line)
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch key {
		case "body":
			inBody = true
			if value != "" {
				body = append(body, value)
			}
		case "to":
			out.To = value
		case "subject":
			out.Subject = value
		case "cc":
			out.CC = value
		case "bcc":
			out.BCC = value
		case "attachments":
			out.Attach = csv(strings.Trim(value, `"`))
		}
	}
	out.Body = strings.TrimSpace(strings.Join(body, "\n"))
	return out
}
