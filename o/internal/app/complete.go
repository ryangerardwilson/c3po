package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// candidate is one completion. insert is the unescaped text that replaces
// the word. dir is true when a unique match should end in a slash.
type candidate struct {
	insert string
	dir    bool
}

// wordSpan is the shell word Tab replaces. start and end are rune indexes
// into the command line, and end is past the word so the tail after the
// cursor is discarded. prefix is the unescaped text before the cursor.
type wordSpan struct {
	start   int
	end     int
	at      int
	prefix  string
	quote   rune
	command bool
	file    bool
}

// completeCommand finishes the :! word under the cursor.
func (m model) completeCommand() model {
	next, at, matches := completeShell(m.currentDir, m.input, m.inputAt, os.Getenv("PATH"), homeDir())
	m = m.setInput([]rune(next), at)
	m.matches = matches
	return m
}

func homeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return dir
}

// completeShell completes one word the way bash does for a command or a path.
// dir is the listed directory. A line that does not start with ! is unchanged.
func completeShell(dir, line string, at int, pathEnv, home string) (string, int, []string) {
	word, ok := shellWordAt(line, at)
	if !ok {
		return line, at, nil
	}
	cands := commandCandidates(pathEnv, word.prefix)
	if word.file {
		cands = fileCandidates(dir, word.prefix, home)
	}
	return applyCandidates(line, word, cands)
}

func shellWordAt(line string, at int) (wordSpan, bool) {
	rs := []rune(line)
	at = clampIndex(at, len(rs))
	bang := bangIndex(rs)
	if bang < 0 || at <= bang {
		return wordSpan{}, false
	}
	return wordAfterBang(rs, bang, at)
}

func bangIndex(rs []rune) int {
	i := skipSpace(rs, 0)
	if i < len(rs) && rs[i] == '!' {
		return i
	}
	return -1
}

func wordAfterBang(rs []rune, bang, at int) (wordSpan, bool) {
	pos := bang + 1
	index := 0
	for {
		pos = skipSpace(rs, pos)
		if at < pos || pos >= len(rs) {
			return emptyWord(at, index == 0), true
		}
		end, prefix := scanWord(rs, pos, at)
		if end <= pos && pos < len(rs) {
			end = pos + 1
		}
		if at <= end {
			return wordAt(rs, pos, end, at, prefix, index == 0), true
		}
		pos = end
		index++
	}
}

func emptyWord(at int, command bool) wordSpan {
	return wordSpan{start: at, end: at, at: at, command: command, file: !command}
}

func wordAt(rs []rune, start, end, at int, prefix string, command bool) wordSpan {
	quote := rune(0)
	if start < len(rs) && (rs[start] == '\'' || rs[start] == '"') {
		quote = rs[start]
	}
	file := !command || strings.Contains(prefix, "/") || strings.HasPrefix(prefix, "~")
	return wordSpan{
		start: start, end: end, at: at, prefix: prefix,
		quote: quote, command: command, file: file,
	}
}

func scanWord(rs []rune, start, at int) (int, string) {
	if start < len(rs) && (rs[start] == '\'' || rs[start] == '"') {
		return scanQuoted(rs, start, at, rs[start])
	}
	return scanBare(rs, start, at)
}

func scanBare(rs []rune, start, at int) (int, string) {
	var b strings.Builder
	i := start
	for i < len(rs) {
		r := rs[i]
		if r == ' ' || r == '\t' {
			break
		}
		if r == '\\' && i+1 < len(rs) {
			if i+1 < at {
				b.WriteRune(rs[i+1])
			}
			i += 2
			continue
		}
		if i < at {
			b.WriteRune(r)
		}
		i++
	}
	return i, b.String()
}

func scanQuoted(rs []rune, start, at int, quote rune) (int, string) {
	var b strings.Builder
	i := start + 1
	for i < len(rs) {
		r := rs[i]
		if quote == '"' && r == '\\' && i+1 < len(rs) && isDoubleEscape(rs[i+1]) {
			if i+1 < at {
				b.WriteRune(rs[i+1])
			}
			i += 2
			continue
		}
		if r == quote {
			return i + 1, b.String()
		}
		if i < at {
			b.WriteRune(r)
		}
		i++
	}
	return i, b.String()
}

func isDoubleEscape(r rune) bool {
	return r == '\\' || r == '"' || r == '$' || r == '`' || r == '\n'
}

func clampIndex(at, n int) int {
	if at < 0 {
		return 0
	}
	if at > n {
		return n
	}
	return at
}

func skipSpace(rs []rune, i int) int {
	for i < len(rs) && (rs[i] == ' ' || rs[i] == '\t') {
		i++
	}
	return i
}

func fileCandidates(current, prefix, home string) []candidate {
	typedDir, base := splitPathPrefix(prefix)
	dir, ok := resolveDir(current, typedDir, home)
	if !ok {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []candidate
	for _, entry := range entries {
		if cand, ok := fileCandidate(dir, typedDir, base, entry); ok {
			out = append(out, cand)
		}
	}
	sortCandidates(out)
	return out
}

func fileCandidate(dir, typedDir, base string, entry os.DirEntry) (candidate, bool) {
	name := entry.Name()
	if name == "." || name == ".." {
		return candidate{}, false
	}
	if strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".") {
		return candidate{}, false
	}
	if !strings.HasPrefix(name, base) {
		return candidate{}, false
	}
	return candidate{insert: typedDir + name, dir: entryIsDir(dir, entry)}, true
}

func entryIsDir(dir string, entry os.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	info, err := os.Stat(filepath.Join(dir, entry.Name()))
	return err == nil && info.IsDir()
}

func splitPathPrefix(prefix string) (string, string) {
	if prefix == "~" {
		return "~/", ""
	}
	slash := strings.LastIndex(prefix, "/")
	if slash < 0 {
		return "", prefix
	}
	return prefix[:slash+1], prefix[slash+1:]
}

func resolveDir(current, typed, home string) (string, bool) {
	switch {
	case typed == "":
		return statDir(current)
	case typed == "~" || strings.HasPrefix(typed, "~/"):
		return statDir(joinHome(home, typed))
	case strings.HasPrefix(typed, "~"):
		return "", false
	case filepath.IsAbs(typed):
		return statDir(typed)
	default:
		return statDir(filepath.Join(current, typed))
	}
}

func joinHome(home, typed string) string {
	if home == "" {
		return ""
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(typed, "~"), "/")
	if rest == "" {
		return home
	}
	return filepath.Join(home, rest)
}

func statDir(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return path, true
}

func commandCandidates(pathEnv, prefix string) []candidate {
	seen := map[string]bool{}
	var out []candidate
	for _, dir := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if cand, ok := commandCandidate(dir, prefix, entry, seen); ok {
				out = append(out, cand)
			}
		}
	}
	sortCandidates(out)
	return out
}

func commandCandidate(dir, prefix string, entry os.DirEntry, seen map[string]bool) (candidate, bool) {
	name := entry.Name()
	if name == "." || name == ".." || seen[name] {
		return candidate{}, false
	}
	if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
		return candidate{}, false
	}
	if !strings.HasPrefix(name, prefix) {
		return candidate{}, false
	}
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return candidate{}, false
	}
	seen[name] = true
	return candidate{insert: name}, true
}

func sortCandidates(cands []candidate) {
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].insert < cands[j].insert
	})
}

func applyCandidates(line string, word wordSpan, cands []candidate) (string, int, []string) {
	if len(cands) == 0 {
		return line, word.at, nil
	}
	if len(cands) == 1 {
		return applyOne(line, word, cands[0])
	}
	return applyPartial(line, word, commonPrefix(insertNames(cands)), cands)
}

func applyOne(line string, word wordSpan, cand candidate) (string, int, []string) {
	text := quoteInsert(cand.insert, word.quote, !cand.dir)
	if cand.dir {
		text += "/"
	} else if !nextIsSpace(line, word.end) {
		text += " "
	}
	next, at := splice([]rune(line), word.start, word.end, text)
	return next, at, nil
}

func applyPartial(line string, word wordSpan, common string, cands []candidate) (string, int, []string) {
	text := quoteInsert(common, word.quote, false)
	next, at := splice([]rune(line), word.start, word.end, text)
	if common != word.prefix {
		return next, at, nil
	}
	return next, at, displayNames(cands)
}

func insertNames(cands []candidate) []string {
	names := make([]string, len(cands))
	for i, cand := range cands {
		names[i] = cand.insert
	}
	return names
}

func displayNames(cands []candidate) []string {
	names := make([]string, len(cands))
	for i, cand := range cands {
		name := cand.insert
		if slash := strings.LastIndex(name, "/"); slash >= 0 {
			name = name[slash+1:]
		}
		if cand.dir {
			name += "/"
		}
		names[i] = name
	}
	return names
}

func commonPrefix(names []string) string {
	if len(names) == 0 {
		return ""
	}
	prefix := []rune(names[0])
	for _, name := range names[1:] {
		prefix = shrinkPrefix(prefix, []rune(name))
	}
	return string(prefix)
}

func shrinkPrefix(a, b []rune) []rune {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}

func nextIsSpace(line string, end int) bool {
	rs := []rune(line)
	return end < len(rs) && (rs[end] == ' ' || rs[end] == '\t')
}

func splice(rs []rune, start, end int, text string) (string, int) {
	add := []rune(text)
	next := make([]rune, 0, len(rs)-(end-start)+len(add))
	next = append(next, rs[:start]...)
	next = append(next, add...)
	if end < len(rs) {
		next = append(next, rs[end:]...)
	}
	return string(next), start + len(add)
}

// bareEsc is the set bash would split or expand in an unquoted word.
const bareEsc = " \t\n\\'\"`$&|;()<>*?[]#{}!"

func quoteInsert(text string, quote rune, close bool) string {
	switch quote {
	case '\'':
		body := strings.ReplaceAll(text, "'", `'\''`)
		if close {
			return "'" + body + "'"
		}
		return "'" + body
	case '"':
		body := escapeDouble(text)
		if close {
			return `"` + body + `"`
		}
		return `"` + body
	default:
		return escapeBare(text)
	}
}

func escapeBare(text string) string {
	var b strings.Builder
	for _, r := range text {
		if strings.ContainsRune(bareEsc, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func escapeDouble(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r == '\\' || r == '"' || r == '$' || r == '`' || r == '\n' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
