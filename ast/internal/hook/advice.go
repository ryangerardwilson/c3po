package hook

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// createTools write a whole file. Aimed at a path that does not exist they
// are a create, which cannot clobber anything, so they are allowed.
var createTools = map[string]bool{
	"write": true, "writefile": true, "create": true, "createorupdatefile": true,
}

// readTools read one file and may bound it with offset/limit.
var readTools = map[string]bool{
	"read": true, "readfile": true, "view": true, "viewfile": true,
}

// read judges a host file read the way rgw-ast read does: a text file longer
// than max_read_lines needs a limit of at most max_read_lines.
func (j *judge) read(input map[string]any, cwd string) (Decision, error) {
	target := targetPath(input, cwd)
	if target == "" || !j.cfg.Enforcement.DenyWholeFileRead {
		return allow(), nil
	}
	max := j.cfg.Enforcement.MaxReadLines
	if limit, ok := intField(input, "limit"); ok && limit > 0 && limit <= max {
		return allow(), nil
	}
	n, text := textLines(target)
	if !text || n <= max {
		return allow(), nil
	}
	repo, enforced, err := j.enforced(target)
	if err != nil || !enforced {
		return allow(), err
	}
	start := 1
	if off, ok := intField(input, "offset"); ok && off > 1 {
		start = off
	}
	end := start + max - 1
	if end > n {
		end = n
	}
	return deny(fmt.Sprintf("enforced workspace %s: %s has %d lines, over max_read_lines %d. Pass limit <= %d to Read, or run: rgw-ast read %s --lines %d-%d",
		repo, filepath.Base(target), n, max, max, j.pathArg(repo, target, cwd), start, end)), nil
}

// fileDenyReason names the rgw-ast command that makes the same change.
func (j *judge) fileDenyReason(repo, target, tool, cwd string) string {
	p := j.pathArg(repo, target, cwd)
	head := "enforced workspace " + repo + ": " + tool + " is denied here. "
	st, err := os.Stat(target)
	switch {
	case err != nil:
		return head + "To create the file run: rgw-ast create " + p + " --expect-absent --parents --stdin"
	case !st.Mode().IsRegular():
		return head + "Use rgw-ast patch/create/move/delete on the files inside it."
	}
	sum, err := hashFile(target)
	if err != nil {
		sum = "<sha from rgw-ast hash " + p + ">"
	}
	switch normalizeTool(tool) {
	case "delete", "deletefile":
		return head + "Run: rgw-ast delete " + p + " --expect-hash " + sum
	}
	return head + "Run: rgw-ast patch " + p + " --expect-hash " + sum + " --old <text> --new <text>" +
		" (or --old-file/--new-file; --all for every occurrence). Several files at once: rgw-ast apply --from-file <manifest.json|git diff>."
}

// pathArg is how an rgw-ast command started from cwd should name target:
// relative to the repository when cwd is in it, otherwise absolute.
func (j *judge) pathArg(repo, target, cwd string) string {
	if j.override == "" {
		if r, ok := repoFor(cwd); !ok || r != repo {
			return target
		}
	}
	rel, err := filepath.Rel(repo, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return target
	}
	return filepath.ToSlash(rel)
}

// textLines counts the lines of a regular text file. It reports false for a
// missing, non-regular, or binary file.
func textLines(path string) (int, bool) {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return 0, false
	}
	data, err := os.ReadFile(path)
	if err != nil || bytes.IndexByte(data[:min(len(data), 8192)], 0) >= 0 {
		return 0, false
	}
	n := bytes.Count(data, []byte{'\n'})
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n, true
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// intField reads a numeric tool input that may arrive as a JSON number or string.
func intField(m map[string]any, key string) (int, bool) {
	switch v := m[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	}
	return 0, false
}
