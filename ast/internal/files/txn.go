package files

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ryangerardwilson/c3po/ast/internal/trace"
)

type FileChange struct {
	Path       string `json:"path"`
	ExpectHash string `json:"expect_hash"`
	Old        string `json:"old,omitempty"`
	New        string `json:"new,omitempty"`
	Symbol     string `json:"symbol,omitempty"`
	Op         string `json:"op,omitempty"`
	Diff       string `json:"diff,omitempty"`
	To         string `json:"to,omitempty"` // move destination
}

type ApplyResult struct {
	Path      string `json:"path"`
	To        string `json:"to,omitempty"`
	Hash      string `json:"hash,omitempty"`
	Formatter string `json:"formatter,omitempty"`
	Op        string `json:"op"`
	Diff      string `json:"diff,omitempty"`
}

// ApplyOpts changes how a transaction writes.
type ApplyOpts struct {
	NoFormat bool // write the edited bytes exactly
	// Check sees the bytes every edit and create would write, after
	// formatting. An error from it writes nothing, in a dry run too.
	Check func([]Written) error
}

// Written is the bytes one edit or create would write to the slash path
// Rel. Old is nil for a create.
type Written struct {
	Rel      string
	Old, New []byte
}

type planned struct {
	abs, rel, content, formatter, op string
	dst, dstRel                      string // move destination
	original                         []byte // edit only
	hash                             string // prepare-time hash of a move or delete
}

// ApplyAll validates every change before writing any file.
// A failure leaves the tree untouched. Touched Go, Python, TypeScript, and
// declared-Markdown files are formatted with the checkout's tool.
func ApplyAll(root string, changes []FileChange) ([]ApplyResult, error) {
	return ApplyAllOpts(root, changes, ApplyOpts{})
}

// ApplyAllOpts is ApplyAll with options.
func ApplyAllOpts(root string, changes []FileChange, opts ApplyOpts) ([]ApplyResult, error) {
	plan, err := prepareChecked(root, changes, opts)
	if err != nil {
		return nil, err
	}
	return commit(plan)
}

// DryRun validates and formats without writing. Diff is the final text.
func DryRun(root string, changes []FileChange) ([]ApplyResult, error) {
	return DryRunOpts(root, changes, ApplyOpts{})
}

// DryRunOpts is DryRun with options.
func DryRunOpts(root string, changes []FileChange, opts ApplyOpts) ([]ApplyResult, error) {
	plan, err := prepareChecked(root, changes, opts)
	if err != nil {
		return nil, err
	}
	var out []ApplyResult
	for _, item := range plan {
		res := ApplyResult{Path: item.rel, To: item.dstRel, Op: item.op, Formatter: item.formatter}
		// A move or delete has no diff; a delete never copies its body out.
		if item.op == "edit" || item.op == "create" {
			res.Diff = unifiedDiff(item.rel, string(item.original), item.content)
		}
		out = append(out, res)
	}
	return out, nil
}

// prepareChecked is prepare, then opts.Check on the bytes of every edit and
// create in the plan.
func prepareChecked(root string, changes []FileChange, opts ApplyOpts) ([]planned, error) {
	plan, err := prepare(root, changes, opts)
	if err != nil || opts.Check == nil {
		return plan, err
	}
	var writes []Written
	for _, item := range plan {
		if item.op == "edit" || item.op == "create" {
			writes = append(writes, Written{Rel: item.rel, Old: item.original, New: []byte(item.content)})
		}
	}
	if err := opts.Check(writes); err != nil {
		return nil, err
	}
	return plan, nil
}

func prepare(root string, changes []FileChange, opts ApplyOpts) ([]planned, error) {
	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes")
	}
	format := func(abs, content string) (string, string) {
		if opts.NoFormat {
			return content, ""
		}
		out, tool := Format(root, abs, []byte(content))
		return string(out), tool
	}
	var plan []planned
	seen := map[string]bool{}
	for i, ch := range changes {
		op := ch.Op
		if op == "" {
			op = "edit"
		}
		if op != "edit" && op != "create" && op != "delete" && op != "move" {
			return nil, fmt.Errorf("change %d: unknown op %s", i, op)
		}
		abs, err := ResolvePath(root, ch.Path)
		if err != nil {
			return nil, fmt.Errorf("change %d: %w", i, err)
		}
		if seen[abs] {
			return nil, fmt.Errorf("change %d: duplicate path %s", i, ch.Path)
		}
		seen[abs] = true
		rel, _ := filepath.Rel(root, abs)
		rel = filepath.ToSlash(rel)
		switch op {
		case "move":
			if ch.To == "" || ch.Symbol != "" || ch.Diff != "" || ch.Old != "" || ch.New != "" {
				return nil, fmt.Errorf("change %d: move takes only to and a hash", i)
			}
			dst, err := ResolvePath(root, ch.To)
			if err != nil {
				return nil, fmt.Errorf("change %d: %w", i, err)
			}
			if seen[dst] {
				return nil, fmt.Errorf("change %d: duplicate path %s", i, ch.To)
			}
			seen[dst] = true
			if _, err := os.Lstat(dst); err == nil {
				return nil, fmt.Errorf("change %d: %s already exists", i, ch.To)
			}
			if _, err := os.Stat(filepath.Dir(dst)); err != nil {
				return nil, fmt.Errorf("change %d: destination directory missing", i)
			}
			cur, err := HashFile(abs)
			if err != nil {
				return nil, fmt.Errorf("change %d: %w", i, err)
			}
			if !strings.EqualFold(cur, ch.ExpectHash) {
				return nil, fmt.Errorf("change %d (%s): %w", i, ch.Path, StaleHash(ch.ExpectHash, cur))
			}
			dstRel, _ := filepath.Rel(root, dst)
			plan = append(plan, planned{abs: abs, rel: rel, op: op, dst: dst, dstRel: filepath.ToSlash(dstRel), hash: cur})
		case "create":
			if ch.Symbol != "" {
				return nil, fmt.Errorf("change %d: symbol is only for edit", i)
			}
			if _, err := os.Stat(abs); err == nil {
				return nil, fmt.Errorf("change %d: %s already exists", i, ch.Path)
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("change %d: %w", i, err)
			}
			if ch.ExpectHash != "" && !strings.EqualFold(ch.ExpectHash, "absent") {
				return nil, fmt.Errorf("change %d: create expects an absent hash", i)
			}
			if _, err := os.Stat(filepath.Dir(abs)); err != nil {
				return nil, fmt.Errorf("change %d: parent directory missing", i)
			}
			content, err := newContent("", ch)
			if err != nil {
				return nil, fmt.Errorf("change %d: %w", i, err)
			}
			formatted, tool := format(abs, content)
			plan = append(plan, planned{abs: abs, rel: rel, content: formatted, formatter: tool, op: op})
		case "delete":
			if ch.Symbol != "" || ch.Old != "" || ch.New != "" {
				return nil, fmt.Errorf("change %d: delete only takes a hash or a diff", i)
			}
			var cur string
			if ch.ExpectHash == "" && ch.Diff != "" {
				// A deletion diff guards by removing every line of the file.
				orig, err := os.ReadFile(abs)
				if err != nil {
					return nil, fmt.Errorf("change %d: %w", i, err)
				}
				if rest, err := applyUnified(string(orig), ch.Diff); err != nil || strings.TrimSpace(rest) != "" {
					return nil, fmt.Errorf("change %d: deletion diff does not match %s", i, ch.Path)
				}
				cur = hashBytes(orig)
			} else {
				if cur, err = HashFile(abs); err != nil {
					return nil, fmt.Errorf("change %d: %w", i, err)
				}
				if !strings.EqualFold(cur, ch.ExpectHash) {
					return nil, fmt.Errorf("change %d (%s): %w", i, ch.Path, StaleHash(ch.ExpectHash, cur))
				}
			}
			plan = append(plan, planned{abs: abs, rel: rel, op: op, hash: cur})
		default:
			if ch.Symbol != "" && ch.Diff != "" {
				return nil, fmt.Errorf("change %d: symbol cannot be combined with a diff", i)
			}
			orig, err := os.ReadFile(abs)
			if err != nil {
				return nil, fmt.Errorf("change %d: %w", i, err)
			}
			// A diff edit without a hash is guarded by its context lines.
			if cur := hashBytes(orig); (ch.ExpectHash != "" || ch.Diff == "") && !strings.EqualFold(cur, ch.ExpectHash) {
				return nil, fmt.Errorf("change %d (%s): %w", i, ch.Path, StaleHash(ch.ExpectHash, cur))
			}
			content := string(orig)
			if ch.Symbol != "" {
				start, end, ok := trace.SymbolSpan(abs, rel, ch.Symbol)
				if !ok {
					return nil, fmt.Errorf("change %d: symbol %s not in %s", i, ch.Symbol, ch.Path)
				}
				content, err = replaceOnceInSpan(content, ch.Old, ch.New, start, end)
				if err != nil {
					return nil, fmt.Errorf("change %d: %w", i, err)
				}
			} else {
				content, err = newContent(content, ch)
				if err != nil {
					return nil, fmt.Errorf("change %d: %w", i, err)
				}
			}
			formatted, tool := format(abs, content)
			plan = append(plan, planned{abs: abs, rel: rel, content: formatted, formatter: tool, op: op, original: orig})
		}
	}
	return plan, nil
}

func replaceOnceInSpan(content, old, new string, start, end int) (string, error) {
	if old == "" && new == "" {
		return "", fmt.Errorf("edit needs old/new")
	}
	nl := strings.HasSuffix(content, "\n")
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if end > len(lines) {
		end = len(lines)
	}
	if start < 1 || end < start {
		return "", fmt.Errorf("symbol span out of range")
	}
	body := strings.Join(lines[start-1:end], "\n")
	if strings.Count(body, old) != 1 {
		return "", fmt.Errorf("inside symbol: %w", uniqueMatchError(body, old))
	}
	body = strings.Replace(body, old, new, 1)
	var out []string
	out = append(out, lines[:start-1]...)
	out = append(out, strings.Split(body, "\n")...)
	out = append(out, lines[end:]...)
	s := strings.Join(out, "\n")
	if nl {
		s += "\n"
	}
	return s, nil
}

func newContent(content string, ch FileChange) (string, error) {
	if ch.Op == "delete" {
		return "", nil
	}
	if ch.Diff != "" {
		if ch.Old != "" || ch.New != "" {
			return "", fmt.Errorf("diff cannot be combined with old/new")
		}
		return applyUnified(content, ch.Diff)
	}
	if ch.Op == "create" {
		return ch.New, nil
	}
	if ch.Old == "" && ch.New == "" {
		return "", fmt.Errorf("edit needs old/new or a diff")
	}
	if ch.Symbol == "" && strings.Count(content, ch.Old) != 1 {
		return "", uniqueMatchError(content, ch.Old)
	}
	if !strings.Contains(content, ch.Old) {
		return "", fmt.Errorf("old string not found")
	}
	return strings.Replace(content, ch.Old, ch.New, 1), nil
}

// keepPrefix names the sibling that holds a deleted file until commit ends.
const keepPrefix = ".c3po-ast-keep-"

// linkFile is os.Link; tests replace it to force the copy fallback.
var linkFile = os.Link

func commit(plan []planned) ([]ApplyResult, error) {
	type done struct {
		item planned
		keep string // sibling holding a deleted file
	}
	var wrote []done
	rollback := func() error {
		var errs []error
		for i := len(wrote) - 1; i >= 0; i-- {
			prev := wrote[i]
			switch prev.item.op {
			case "create":
				_ = os.Remove(prev.item.abs)
			case "move":
				_ = os.Rename(prev.item.dst, prev.item.abs)
			case "delete":
				if err := restoreDeleted(prev.item.abs, prev.keep); err != nil {
					errs = append(errs, err)
				}
			default:
				_, _ = writeAtomic(prev.item.abs, prev.item.original)
			}
		}
		return errors.Join(errs...)
	}
	out := make([]ApplyResult, 0, len(plan))
	for _, item := range plan {
		res := ApplyResult{Path: item.rel, To: item.dstRel, Formatter: item.formatter, Op: item.op, Hash: item.hash}
		var keep string
		var err error
		switch item.op {
		case "delete":
			keep, err = stageDelete(item.abs)
		case "move":
			err = os.Rename(item.abs, item.dst)
		default:
			res.Hash, err = writeAtomic(item.abs, []byte(item.content))
		}
		if err != nil {
			if rbErr := rollback(); rbErr != nil {
				return nil, errors.Join(err, rbErr)
			}
			return nil, err
		}
		wrote = append(wrote, done{item: item, keep: keep})
		out = append(out, res)
	}
	for _, d := range wrote {
		if d.keep != "" {
			_ = os.Remove(d.keep)
		}
	}
	return out, nil
}

// stageDelete moves abs aside to a unique sibling and returns the sibling.
// A hard link keeps mode, owner, and inode; where the filesystem cannot
// link, the file is streamed to the sibling with its mode.
func stageDelete(abs string) (string, error) {
	keep, err := placeKeep(abs, func(keep string) error { return linkFile(abs, keep) })
	if err != nil && cannotLink(err) {
		keep, err = placeKeep(abs, func(keep string) error { return copyKeep(abs, keep) })
	}
	if err != nil {
		return "", err
	}
	if err := os.Remove(abs); err != nil {
		_ = os.Remove(keep)
		return "", err
	}
	return keep, nil
}

// placeKeep calls place with fresh sibling names until one is not taken.
func placeKeep(abs string, place func(keep string) error) (string, error) {
	for range 16 {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		keep := filepath.Join(filepath.Dir(abs), keepPrefix+filepath.Base(abs)+"-"+hex.EncodeToString(b[:]))
		err := place(keep)
		if err == nil {
			return keep, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("no free name to keep %s", abs)
}

// cannotLink reports a filesystem that does not support the hard link.
func cannotLink(err error) bool {
	return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EMLINK) || errors.Is(err, syscall.EXDEV)
}

// copyKeep streams abs to the new path keep with the source mode.
func copyKeep(abs, keep string) error {
	src, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(keep, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	if err == nil {
		err = dst.Chmod(info.Mode().Perm())
	}
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(keep)
	}
	return err
}

// restoreDeleted renames keep back to abs. An occupied abs is not
// overwritten: keep stays and the error names it.
func restoreDeleted(abs, keep string) error {
	if _, err := os.Lstat(abs); err == nil {
		return fmt.Errorf("restore %s: path is occupied; the deleted file is kept at %s", abs, keep)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("restore %s: %w; the deleted file is kept at %s", abs, err, keep)
	}
	return os.Rename(keep, abs)
}

// Format runs the checkout formatter for this path only.
// A missing tool returns the original bytes and an empty tool name.
func Format(root, abs string, content []byte) ([]byte, string) {
	ext := strings.ToLower(filepath.Ext(abs))
	switch ext {
	case ".go":
		if out, err := runTool(root, content, "gofmt"); err == nil {
			return out, "gofmt"
		}
	case ".py":
		if hasRuff(root) {
			if out, err := runTool(root, content, "ruff", "format", "--stdin-filename", abs, "-"); err == nil {
				return out, "ruff"
			}
		}
	case ".ts", ".tsx", ".js", ".jsx":
		if fileExists(filepath.Join(root, "biome.json")) || fileExists(filepath.Join(root, "biome.jsonc")) {
			if out, err := runTool(root, content, "biome", "format", "--stdin-file-path", abs); err == nil {
				return out, "biome"
			}
		}
	case ".md", ".markdown":
		if b, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil && bytes.Contains(bytes.ToLower(b), []byte("prettier")) {
			if out, err := runTool(root, content, "prettier", "--stdin-filepath", abs); err == nil {
				return out, "prettier"
			}
		}
		if fileExists(filepath.Join(root, "dprint.json")) {
			if out, err := runTool(root, content, "dprint", "fmt", "--stdin", abs); err == nil {
				return out, "dprint"
			}
		}
	}
	return content, ""
}

func hasRuff(root string) bool {
	for _, rel := range []string{"pyproject.toml", "ruff.toml"} {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err == nil && bytes.Contains(bytes.ToLower(b), []byte("ruff")) {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func runTool(root string, content []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(content)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return content, nil
	}
	return out, nil
}

func ParseApplyFile(path string) ([]FileChange, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseApplyBytes(b)
}

func ParseApplyBytes(b []byte) ([]FileChange, error) {
	trim := bytes.TrimSpace(b)
	if len(trim) == 0 {
		return nil, fmt.Errorf("empty manifest")
	}
	if trim[0] == '{' || trim[0] == '[' {
		return parseApplyJSON(b)
	}
	return parseDiffBundle(string(b))
}

func parseApplyJSON(b []byte) ([]FileChange, error) {
	var wrap struct {
		Files []FileChange `json:"files"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Files) == 0 {
		var list []FileChange
		if err := json.Unmarshal(b, &list); err != nil {
			return nil, err
		}
		return list, nil
	}
	return wrap.Files, nil
}

func parseDiffBundle(text string) ([]FileChange, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var sections []string
	var cur strings.Builder
	started := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# path:") {
			if started {
				sections = append(sections, cur.String())
				cur.Reset()
			}
			started = true
		}
		if started {
			cur.WriteString(line)
			cur.WriteByte('\n')
		}
	}
	if started {
		sections = append(sections, cur.String())
	}
	if len(sections) == 0 {
		return parsePlainDiff(text)
	}
	var out []FileChange
	for _, sec := range sections {
		ch := FileChange{Op: "edit"}
		var diffLines []string
		inDiff := false
		for _, line := range strings.Split(sec, "\n") {
			switch {
			case strings.HasPrefix(line, "# path:"):
				ch.Path = strings.TrimSpace(strings.TrimPrefix(line, "# path:"))
			case strings.HasPrefix(line, "# expect-hash:"):
				ch.ExpectHash = strings.TrimSpace(strings.TrimPrefix(line, "# expect-hash:"))
			case strings.HasPrefix(line, "# op:"):
				ch.Op = strings.TrimSpace(strings.TrimPrefix(line, "# op:"))
			case strings.HasPrefix(line, "# symbol:"):
				ch.Symbol = strings.TrimSpace(strings.TrimPrefix(line, "# symbol:"))
			case strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "@@") || (inDiff && line != ""):
				inDiff = true
				diffLines = append(diffLines, line)
			}
		}
		ch.Diff = strings.Join(diffLines, "\n")
		if ch.Path == "" {
			return nil, fmt.Errorf("diff section missing path")
		}
		out = append(out, ch)
	}
	return out, nil
}

// parsePlainDiff splits a git diff or diff -u into one change per file,
// taking paths from the ---/+++ headers. /dev/null on the old side is a
// create, on the new side a delete.
func parsePlainDiff(text string) ([]FileChange, error) {
	lines := strings.Split(text, "\n")
	var out []FileChange
	var cur *FileChange
	var body []string
	flush := func() {
		if cur != nil {
			cur.Diff = strings.Join(body, "\n")
			out = append(out, *cur)
		}
		cur, body = nil, nil
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ") {
			flush()
			oldPath, newPath := diffPath(line[4:]), diffPath(lines[i+1][4:])
			if oldPath == newPath+".orig" {
				oldPath = newPath // gofmt -d labels the old side file.orig
			}
			ch := FileChange{Op: "edit", Path: newPath}
			switch {
			case oldPath == "/dev/null":
				ch.Op = "create"
			case newPath == "/dev/null":
				ch.Op, ch.Path = "delete", oldPath
			case oldPath != newPath:
				return nil, fmt.Errorf("diff renames %s to %s; use a move op in a JSON manifest", oldPath, newPath)
			}
			cur = &ch
			body = []string{line, lines[i+1]}
			i++
			continue
		}
		if cur == nil {
			continue // preamble: diff --git, index, mode lines
		}
		if line == "" || strings.HasPrefix(line, "@@") || line[0] == ' ' || line[0] == '+' || line[0] == '-' || line[0] == '\\' {
			body = append(body, line)
		}
	}
	flush()
	if len(out) == 0 {
		return nil, fmt.Errorf("no diff sections: give a unified diff with ---/+++ headers, or a bundle with # path: lines")
	}
	return out, nil
}

// diffPath strips a diff header's a/ or b/ prefix and diff -u timestamp.
func diffPath(s string) string {
	if i := strings.IndexByte(s, '\t'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if s == "/dev/null" {
		return s
	}
	if strings.HasPrefix(s, "a/") || strings.HasPrefix(s, "b/") {
		s = s[2:]
	}
	return s
}

func applyUnified(content, diff string) (string, error) {
	lines, nl := splitKeep(content)
	hunks, err := parseHunks(diff)
	if err != nil {
		return "", err
	}
	// Hunk positions refer to the original file, so apply the last hunk
	// first: an earlier hunk that changes the line count cannot shift it.
	for i := len(hunks) - 1; i >= 0; i-- {
		lines, err = applyHunk(lines, hunks[i])
		if err != nil {
			return "", err
		}
	}
	out := joinKeep(lines, nl)
	if content == "" && out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

type hunk struct {
	oldStart, oldCount int
	lines              []string
}

func parseHunks(diff string) ([]hunk, error) {
	var hunks []hunk
	var cur *hunk
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "@@") {
			if cur != nil {
				hunks = append(hunks, *cur)
			}
			start, count, err := parseAt(line)
			if err != nil {
				return nil, err
			}
			cur = &hunk{oldStart: start, oldCount: count}
			continue
		}
		if cur == nil || line == "" || strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "\\") {
			continue
		}
		cur.lines = append(cur.lines, line)
	}
	if cur != nil {
		hunks = append(hunks, *cur)
	}
	if len(hunks) == 0 {
		return nil, fmt.Errorf("diff has no hunks")
	}
	return hunks, nil
}

func parseAt(line string) (int, int, error) {
	i := strings.Index(line, "-")
	j := strings.Index(line, " +")
	if i < 0 || j < i {
		return 0, 0, fmt.Errorf("hunk header %q", line)
	}
	spec := strings.TrimSpace(line[i+1 : j])
	parts := strings.Split(spec, ",")
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("hunk header %q", line)
	}
	count := 1
	if len(parts) == 2 {
		count, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("hunk header %q", line)
		}
	}
	return start, count, nil
}

func applyHunk(lines []string, h hunk) ([]string, error) {
	idx := h.oldStart - 1
	if h.oldStart == 0 {
		idx = 0
	}
	if idx < 0 {
		return nil, fmt.Errorf("hunk starts before the file")
	}
	var repl []string
	oldCount := 0
	check := idx
	for _, hl := range h.lines {
		if hl == "" {
			continue
		}
		switch hl[0] {
		case ' ':
			if check >= len(lines) || lines[check] != hl[1:] {
				return nil, fmt.Errorf("hunk context mismatch")
			}
			repl = append(repl, hl[1:])
			check++
			oldCount++
		case '-':
			if check >= len(lines) || lines[check] != hl[1:] {
				return nil, fmt.Errorf("hunk context mismatch")
			}
			check++
			oldCount++
		case '+':
			repl = append(repl, hl[1:])
		default:
			return nil, fmt.Errorf("bad diff line %q", hl)
		}
	}
	if h.oldCount != oldCount && !(h.oldStart == 0 && h.oldCount == 0) {
		return nil, fmt.Errorf("hunk length mismatch")
	}
	if idx+oldCount > len(lines) {
		return nil, fmt.Errorf("hunk out of range")
	}
	out := append([]string{}, lines[:idx]...)
	out = append(out, repl...)
	out = append(out, lines[idx+oldCount:]...)
	return out, nil
}

func splitKeep(s string) ([]string, bool) {
	nl := strings.HasSuffix(s, "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil, nl
	}
	return strings.Split(s, "\n"), nl
}

func joinKeep(lines []string, nl bool) string {
	s := strings.Join(lines, "\n")
	if nl {
		s += "\n"
	}
	return s
}

func unifiedDiff(path, old, new string) string {
	if old == new {
		return ""
	}
	dir, err := os.MkdirTemp("", "rgw-diff")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(dir)
	oldp := filepath.Join(dir, "old")
	newp := filepath.Join(dir, "new")
	if err := os.WriteFile(oldp, []byte(old), 0o600); err != nil {
		return ""
	}
	if err := os.WriteFile(newp, []byte(new), 0o600); err != nil {
		return ""
	}
	cmd := exec.Command("diff", "-u", "--label", "a/"+path, "--label", "b/"+path, oldp, newp)
	out, _ := cmd.Output()
	return string(out)
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
