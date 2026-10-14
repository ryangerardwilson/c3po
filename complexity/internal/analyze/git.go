package analyze

import (
	"bytes"
	"os/exec"
	"strings"
)

func changedFiles(root, ref string) (map[string]bool, error) {
	set := map[string]bool{}
	out, err := git(root, "diff", "--name-only", "--diff-filter=ACMR", ref)
	if err != nil {
		return nil, err
	}
	addLines(set, out)
	other, err := git(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	addLines(set, other)
	return set, nil
}

func gitShow(root, ref, path string) ([]byte, bool) {
	out, err := git(root, "show", ref+":"+path)
	if err != nil {
		return nil, false
	}
	return out, true
}

func git(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, &gitError{msg}
	}
	return out, nil
}

type gitError struct{ msg string }

func (e *gitError) Error() string { return e.msg }

func addLines(set map[string]bool, out []byte) {
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			set[line] = true
		}
	}
}
