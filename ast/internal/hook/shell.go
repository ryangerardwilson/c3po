package hook

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/execgen"
)

// The shell analysis answers one question: which paths could this command
// text write? It is a conservative lexer, not a shell. Anything it cannot
// place is reported as a write to the current directory.

type shWord struct {
	text    string
	quoted  bool // any part was quoted or escaped
	dynamic bool // contains an expansion whose value is unknown
}

type shRedir struct {
	op     string
	target shWord
}

type shCommand struct {
	words  []shWord
	redirs []shRedir
}

type shScript struct {
	cmds   []shCommand
	subs   []string // substitution bodies, analysed as nested scripts
	opaque bool     // an unquoted heredoc body runs substitutions
}

type heredoc struct {
	delim     string
	stripTabs bool
	quoted    bool
}

var errShellSyntax = errors.New("unparseable shell text")

type lexer struct {
	src     string
	i       int
	vars    map[string]string // known values of plain $NAME expansions
	out     shScript
	cur     shCommand
	buf     strings.Builder
	inWord  bool
	quoted  bool
	dynamic bool
	pending string // redirection operator awaiting its target word
	docs    []heredoc
}

func parseShell(src string, vars map[string]string) (shScript, error) {
	lx := &lexer{src: src, vars: vars}
	if err := lx.run(); err != nil {
		return shScript{}, err
	}
	return lx.out, nil
}

func (lx *lexer) run() error {
	src := lx.src
	for lx.i < len(src) {
		c := src[lx.i]
		switch {
		case c == '\\':
			if lx.i+1 < len(src) {
				if src[lx.i+1] == '\n' {
					lx.i += 2
					continue
				}
				lx.buf.WriteByte(src[lx.i+1])
				lx.inWord, lx.quoted = true, true
				lx.i += 2
			} else {
				lx.i++
			}
		case c == '\'':
			end := strings.IndexByte(src[lx.i+1:], '\'')
			if end < 0 {
				return errShellSyntax
			}
			lx.buf.WriteString(src[lx.i+1 : lx.i+1+end])
			lx.inWord, lx.quoted = true, true
			lx.i += end + 2
		case c == '"':
			if err := lx.doubleQuoted(); err != nil {
				return err
			}
		case c == '$':
			if err := lx.dollar(); err != nil {
				return err
			}
		case c == '`':
			if err := lx.backquote(); err != nil {
				return err
			}
		case c == '#' && !lx.inWord:
			for lx.i < len(src) && src[lx.i] != '\n' {
				lx.i++
			}
		case c == '~' && !lx.inWord && (lx.i+1 == len(src) || strings.IndexByte("/ \t\n;|&)", src[lx.i+1]) >= 0):
			lx.buf.WriteString(lx.vars["HOME"])
			lx.inWord = true
			lx.i++
		case c == ' ' || c == '\t' || c == '\r':
			lx.finishWord()
			lx.i++
		case c == '\n':
			if err := lx.endCommand(); err != nil {
				return err
			}
			lx.i++
			if err := lx.heredocBodies(); err != nil {
				return err
			}
		case c == ';' || c == '(' || c == ')':
			if err := lx.endCommand(); err != nil {
				return err
			}
			lx.i++
		case c == '&':
			if lx.i+1 < len(src) && src[lx.i+1] == '>' {
				lx.finishWord()
				op := "&>"
				lx.i += 2
				if lx.i < len(src) && src[lx.i] == '>' {
					op = "&>>"
					lx.i++
				}
				if err := lx.setPending(op); err != nil {
					return err
				}
				continue
			}
			if err := lx.endCommand(); err != nil {
				return err
			}
			lx.i++
			if lx.i < len(src) && src[lx.i] == '&' {
				lx.i++
			}
		case c == '|':
			if err := lx.endCommand(); err != nil {
				return err
			}
			lx.i++
			if lx.i < len(src) && (src[lx.i] == '|' || src[lx.i] == '&') {
				lx.i++
			}
		case c == '<' || c == '>':
			if err := lx.redirect(); err != nil {
				return err
			}
		default:
			lx.buf.WriteByte(c)
			lx.inWord = true
			lx.i++
		}
	}
	if err := lx.endCommand(); err != nil {
		return err
	}
	if len(lx.docs) > 0 {
		// A heredoc with no body line: bash reads to end of input.
		lx.docs = nil
	}
	return nil
}

func (lx *lexer) doubleQuoted() error {
	src := lx.src
	lx.i++
	lx.inWord, lx.quoted = true, true
	for lx.i < len(src) && src[lx.i] != '"' {
		switch c := src[lx.i]; c {
		case '\\':
			if lx.i+1 < len(src) && strings.IndexByte("$`\"\\\n", src[lx.i+1]) >= 0 {
				lx.buf.WriteByte(src[lx.i+1])
				lx.i += 2
				continue
			}
			lx.buf.WriteByte(c)
			lx.i++
		case '$':
			if err := lx.dollar(); err != nil {
				return err
			}
		case '`':
			if err := lx.backquote(); err != nil {
				return err
			}
		default:
			lx.buf.WriteByte(c)
			lx.i++
		}
	}
	if lx.i >= len(src) {
		return errShellSyntax
	}
	lx.i++
	return nil
}

func (lx *lexer) dollar() error {
	src := lx.src
	lx.inWord = true
	if lx.i+1 >= len(src) {
		lx.buf.WriteByte('$')
		lx.i++
		return nil
	}
	switch next := src[lx.i+1]; {
	case next == '(':
		end, err := balancedParen(src, lx.i+1)
		if err != nil {
			return err
		}
		lx.out.subs = append(lx.out.subs, src[lx.i+2:end-1])
		lx.dynamic = true
		lx.i = end
	case next == '{':
		end := strings.IndexByte(src[lx.i:], '}')
		if end < 0 {
			return errShellSyntax
		}
		if value, ok := lx.vars[src[lx.i+2:lx.i+end]]; ok {
			lx.buf.WriteString(value)
		} else {
			lx.dynamic = true
		}
		lx.i += end + 1
	case next == '\'':
		end := strings.IndexByte(src[lx.i+2:], '\'')
		if end < 0 {
			return errShellSyntax
		}
		lx.buf.WriteString(src[lx.i+2 : lx.i+2+end])
		lx.quoted = true
		lx.i += end + 3
	case next == '"':
		lx.i++ // $"..." is a translated double-quoted string
	case isNameByte(next):
		j := lx.i + 1
		for j < len(src) && isNameByte(src[j]) {
			j++
		}
		if value, ok := lx.vars[src[lx.i+1:j]]; ok {
			lx.buf.WriteString(value)
		} else {
			lx.dynamic = true
		}
		lx.i = j
	case strings.IndexByte("@*#?$!-", next) >= 0:
		lx.dynamic = true
		lx.i += 2
	default:
		lx.buf.WriteByte('$')
		lx.i++
	}
	return nil
}

func (lx *lexer) backquote() error {
	src := lx.src
	var body strings.Builder
	j := lx.i + 1
	for ; j < len(src) && src[j] != '`'; j++ {
		if src[j] == '\\' && j+1 < len(src) {
			j++
		}
		body.WriteByte(src[j])
	}
	if j >= len(src) {
		return errShellSyntax
	}
	lx.out.subs = append(lx.out.subs, body.String())
	lx.inWord, lx.dynamic = true, true
	lx.i = j + 1
	return nil
}

func (lx *lexer) redirect() error {
	src := lx.src
	c := src[lx.i]
	if lx.i+1 < len(src) && src[lx.i+1] == '(' {
		// Process substitution runs its body as a command.
		end, err := balancedParen(src, lx.i+1)
		if err != nil {
			return err
		}
		lx.out.subs = append(lx.out.subs, src[lx.i+2:end-1])
		lx.inWord, lx.dynamic = true, true
		lx.i = end
		return nil
	}
	if lx.inWord && !lx.quoted && !lx.dynamic && isDigits(lx.buf.String()) {
		lx.resetWord() // a file-descriptor prefix such as 2>
	} else {
		lx.finishWord()
	}
	rest := src[lx.i:]
	var op string
	if c == '>' {
		switch {
		case strings.HasPrefix(rest, ">>"):
			op = ">>"
		case strings.HasPrefix(rest, ">|"):
			op = ">|"
		case strings.HasPrefix(rest, ">&"):
			op = ">&"
		default:
			op = ">"
		}
	} else {
		switch {
		case strings.HasPrefix(rest, "<<<"):
			op = "<<<"
		case strings.HasPrefix(rest, "<<-"):
			op = "<<-"
		case strings.HasPrefix(rest, "<<"):
			op = "<<"
		case strings.HasPrefix(rest, "<&"):
			op = "<&"
		case strings.HasPrefix(rest, "<>"):
			op = "<>"
		default:
			op = "<"
		}
	}
	lx.i += len(op)
	return lx.setPending(op)
}

func (lx *lexer) setPending(op string) error {
	if lx.pending != "" {
		return errShellSyntax
	}
	lx.pending = op
	return nil
}

func (lx *lexer) resetWord() {
	lx.buf.Reset()
	lx.inWord, lx.quoted, lx.dynamic = false, false, false
}

func (lx *lexer) finishWord() {
	if !lx.inWord {
		return
	}
	w := shWord{text: lx.buf.String(), quoted: lx.quoted, dynamic: lx.dynamic}
	lx.resetWord()
	if lx.pending == "" {
		lx.cur.words = append(lx.cur.words, w)
		return
	}
	op := lx.pending
	lx.pending = ""
	lx.cur.redirs = append(lx.cur.redirs, shRedir{op: op, target: w})
	if op == "<<" || op == "<<-" {
		lx.docs = append(lx.docs, heredoc{delim: w.text, stripTabs: op == "<<-", quoted: w.quoted})
	}
}

func (lx *lexer) endCommand() error {
	lx.finishWord()
	if lx.pending != "" {
		return errShellSyntax
	}
	if len(lx.cur.words) > 0 || len(lx.cur.redirs) > 0 {
		lx.out.cmds = append(lx.out.cmds, lx.cur)
	}
	lx.cur = shCommand{}
	return nil
}

// heredocBodies consumes the bodies of heredocs opened on the line just ended.
func (lx *lexer) heredocBodies() error {
	docs := lx.docs
	lx.docs = nil
	for _, doc := range docs {
		for lx.i < len(lx.src) {
			end := strings.IndexByte(lx.src[lx.i:], '\n')
			line := lx.src[lx.i:]
			if end >= 0 {
				line = lx.src[lx.i : lx.i+end]
				lx.i += end + 1
			} else {
				lx.i = len(lx.src)
			}
			check := line
			if doc.stripTabs {
				check = strings.TrimLeft(check, "\t")
			}
			if check == doc.delim {
				break
			}
			if !doc.quoted && (strings.Contains(line, "$(") || strings.Contains(line, "`")) {
				lx.out.opaque = true
			}
		}
	}
	return nil
}

// balancedParen returns the index just past the ')' matching src[open].
// Heredoc bodies are skipped, so their quotes and parentheses do not count.
func balancedParen(src string, open int) (int, error) {
	depth := 0
	var delims []string
	for i := open; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++
		case '<':
			if strings.HasPrefix(src[i:], "<<") && !strings.HasPrefix(src[i:], "<<<") {
				delim, next := heredocDelimiter(src, i+2)
				if delim != "" {
					delims = append(delims, delim)
					i = next - 1
				}
			}
		case '\n':
			for _, delim := range delims {
				i = skipHeredocBody(src, i+1, delim) - 1
			}
			delims = nil
		case '\'':
			end := strings.IndexByte(src[i+1:], '\'')
			if end < 0 {
				return 0, errShellSyntax
			}
			i += end + 1
		case '"':
			j := i + 1
			for ; j < len(src) && src[j] != '"'; j++ {
				if src[j] == '\\' {
					j++
				}
			}
			if j >= len(src) {
				return 0, errShellSyntax
			}
			i = j
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, errShellSyntax
}

// heredocDelimiter reads the delimiter word after "<<" at src[i], without
// quotes, and returns it with the index just past it.
func heredocDelimiter(src string, i int) (string, int) {
	if i < len(src) && src[i] == '-' {
		i++
	}
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	var b strings.Builder
	for i < len(src) && strings.IndexByte(" \t\n;|&<>()", src[i]) < 0 {
		if src[i] != '\'' && src[i] != '"' && src[i] != '\\' {
			b.WriteByte(src[i])
		}
		i++
	}
	return b.String(), i
}

// skipHeredocBody returns the index just past the line that closes a heredoc
// whose body starts at src[i].
func skipHeredocBody(src string, i int, delim string) int {
	for i < len(src) {
		end := strings.IndexByte(src[i:], '\n')
		if end < 0 {
			return len(src)
		}
		line := src[i : i+end]
		i += end + 1
		if strings.TrimLeft(line, "\t") == delim {
			return i
		}
	}
	return i
}

func isNameByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// shellWrites returns the absolute paths the command text could write,
// resolving relative paths against cwd.
func shellWrites(cmd, cwd string) []string {
	var out []string
	for _, t := range analyzeShell(cmd, cwd, nil).targets {
		out = append(out, t.path)
	}
	return out
}

// analyzeShell finds the write targets of the command text and, when cfg is
// given, the generator commands it runs outside rgw-ast exec.
func analyzeShell(cmd, cwd string, cfg *config.Config) *writeAnalysis {
	home, _ := os.UserHomeDir()
	a := &writeAnalysis{home: home, vars: map[string]string{"HOME": home}, cfg: cfg}
	a.script(cmd, cwd, 0)
	return a
}

const maxShellDepth = 8

type writeAnalysis struct {
	home       string
	vars       map[string]string
	cfg        *config.Config
	hint       string // advice for targets added by the current command
	targets    []shTarget
	generators []genCall
}

// shTarget is one written path and the rgw-ast command that would do it.
type shTarget struct {
	path, hint string
}

// genCall is a generator command run outside rgw-ast exec.
type genCall struct {
	cmd, cwd string
}

const (
	hintMove   = "use rgw-ast move <src> <dst> --expect-hash <sha> (rgw-ast hash <src> prints the hash)"
	hintDelete = "use rgw-ast delete <file> --expect-hash <sha> (rgw-ast hash <file> prints the hash)"
	hintPatch  = "use rgw-ast hash <file>, then rgw-ast patch <file> --expect-hash <sha> --old <text> --new <text> [--all]"
	hintInline = "inline interpreter code writes files; use rgw-ast patch/create/move/delete instead"
)

func (a *writeAnalysis) add(path, cwd string) {
	a.targets = append(a.targets, shTarget{path: absPath(path, cwd), hint: a.hint})
}

func (a *writeAnalysis) addWord(w shWord, cwd string) {
	if w.dynamic || w.text == "" {
		a.add(".", cwd) // unknown location: judge the current directory
		return
	}
	a.add(w.text, cwd)
}

func (a *writeAnalysis) script(src, cwd string, depth int) {
	if depth > maxShellDepth {
		a.add(".", cwd)
		return
	}
	sc, err := parseShell(src, a.vars)
	if err == nil && a.learn(sc) {
		sc, err = parseShell(src, a.vars) // expand the variables just learned
	}
	if err != nil || sc.opaque {
		a.add(".", cwd)
		if err != nil {
			return
		}
	}
	for _, sub := range sc.subs {
		a.script(sub, cwd, depth+1)
	}
	for _, c := range sc.cmds {
		cwd = a.command(c, cwd, depth)
	}
}

// learn records NAME=value commands whose value is static, so later $NAME
// targets resolve to a path. It reports whether anything was learned. A
// dynamic value makes the variable unknown again.
func (a *writeAnalysis) learn(sc shScript) bool {
	learned := false
	for _, c := range sc.cmds {
		words := c.words
		if len(words) > 0 && (words[0].text == "export" || words[0].text == "readonly" || words[0].text == "local") {
			words = words[1:]
		}
		if len(words) == 0 || len(c.redirs) > 0 {
			continue
		}
		pure := true
		for _, w := range words {
			pure = pure && isAssignment(w.text)
		}
		if !pure {
			continue
		}
		for _, w := range words {
			name, value, _ := strings.Cut(w.text, "=")
			if w.dynamic {
				delete(a.vars, name)
				continue
			}
			if old, ok := a.vars[name]; !ok || old != value {
				a.vars[name] = value
				learned = true
			}
		}
	}
	return learned
}

// command records the writes of one simple command and returns the
// directory later commands start from.
func (a *writeAnalysis) command(c shCommand, cwd string, depth int) string {
	prevHint := a.hint
	a.hint = ""
	defer func() { a.hint = prevHint }()
	for _, r := range c.redirs {
		switch r.op {
		case ">", ">>", ">|", "&>", "&>>", "<>":
		case ">&":
			if r.target.text == "-" || isDigits(r.target.text) {
				continue // descriptor duplication
			}
		default:
			continue // input, here-string, heredoc
		}
		if !r.target.dynamic && strings.HasPrefix(r.target.text, "/dev/") {
			continue
		}
		a.addWord(r.target, cwd)
	}
	words := stripPrefixes(c.words)
	if len(words) == 0 {
		return cwd
	}
	name := filepath.Base(words[0].text)
	args := words[1:]
	a.generator(words, cwd)
	if strings.HasPrefix(name, "python") {
		name = "python"
	}
	switch name {
	case "mv":
		a.hint = hintMove
	case "rm", "rmdir", "unlink":
		a.hint = hintDelete
	case "sed", "perl", "ruby":
		a.hint = hintPatch
	}
	switch name {
	case "cd":
		ops := operands(args)
		switch {
		case len(ops) == 0:
			return a.home
		case ops[0].dynamic || ops[0].text == "-":
			return cwd
		default:
			return absPath(ops[0].text, cwd)
		}
	case "rm", "rmdir", "unlink", "touch", "shred", "mv", "tee":
		for _, w := range operands(args) {
			a.addWord(w, cwd)
		}
	case "truncate":
		for _, w := range operandsSkipping(args, "-s", "-r", "--size", "--reference") {
			a.addWord(w, cwd)
		}
	case "cp", "ln", "install", "rsync":
		if dir, ok := optionValue(args, "-t", "--target-directory"); ok {
			a.addWord(dir, cwd)
		} else if ops := operandsSkipping(args, copyValuedOptions[name]...); len(ops) > 0 {
			a.addWord(ops[len(ops)-1], cwd)
		}
	case "chmod", "chown", "chgrp":
		ops := operands(args)
		if len(ops) > 1 {
			for _, w := range ops[1:] {
				a.addWord(w, cwd)
			}
		}
	case "sed", "perl", "ruby":
		for _, w := range inPlaceFiles(args) {
			a.addWord(w, cwd)
		}
		if name != "sed" && inlineWrites(name, args) {
			a.hint = hintInline
			a.add(".", cwd)
		}
	case "python", "node", "deno", "bun", "php":
		if inlineWrites(name, args) {
			a.hint = hintInline
			a.add(".", cwd)
		}
	case "dd":
		for _, w := range args {
			if strings.HasPrefix(w.text, "of=") {
				a.addWord(shWord{text: strings.TrimPrefix(w.text, "of="), dynamic: w.dynamic}, cwd)
			}
		}
	case "find":
		a.find(args, cwd, depth)
	case "xargs":
		sub := xargsCommand(args)
		if len(sub) > 0 {
			// Operands come from stdin, so their location is unknown.
			sub = append(sub, shWord{dynamic: true})
			a.command(shCommand{words: sub}, cwd, depth)
		}
	case "git":
		a.git(args, cwd)
	case "apply_patch":
		a.add(".", cwd)
	case "patch":
		if dir, ok := optionValue(args, "-d", "--directory"); ok {
			a.addWord(dir, cwd)
		} else {
			a.add(".", cwd)
		}
	case "sh", "bash", "zsh", "dash", "ksh":
		for i, w := range args {
			if strings.HasPrefix(w.text, "-") && !strings.HasPrefix(w.text, "--") && strings.Contains(w.text, "c") {
				if i+1 < len(args) {
					if args[i+1].dynamic {
						a.add(".", cwd)
					} else {
						a.script(args[i+1].text, cwd, depth+1)
					}
				}
				break
			}
		}
	case "eval":
		parts := make([]string, 0, len(args))
		for _, w := range args {
			if w.dynamic {
				a.add(".", cwd)
			}
			parts = append(parts, w.text)
		}
		a.script(strings.Join(parts, " "), cwd, depth+1)
	}
	return cwd
}

func (a *writeAnalysis) find(args []shWord, cwd string, depth int) {
	var roots []shWord
	i := 0
	for ; i < len(args); i++ {
		t := args[i].text
		if strings.HasPrefix(t, "-") || t == "(" || t == "!" {
			break
		}
		roots = append(roots, args[i])
	}
	if len(roots) == 0 {
		roots = []shWord{{text: "."}}
	}
	for ; i < len(args); i++ {
		switch args[i].text {
		case "-delete", "-fprint", "-fprint0", "-fprintf", "-fls":
			for _, r := range roots {
				a.addWord(r, cwd)
			}
		case "-exec", "-execdir", "-ok", "-okdir":
			var sub []shWord
			j := i + 1
			for ; j < len(args) && args[j].text != ";" && args[j].text != "+"; j++ {
				sub = append(sub, args[j])
			}
			i = j
			for _, r := range roots {
				bound := make([]shWord, len(sub))
				for k, w := range sub {
					bound[k] = w
					if w.text == "{}" {
						bound[k] = r
					}
				}
				if len(bound) > 0 {
					a.command(shCommand{words: bound}, cwd, depth)
				}
			}
		}
	}
}

// copyValuedOptions are the options of copy-like commands whose value is the
// next word, so that value is not taken for the destination.
var copyValuedOptions = map[string][]string{
	"cp":      {"-S", "--suffix"},
	"ln":      {"-S", "--suffix"},
	"install": {"-m", "--mode", "-o", "--owner", "-g", "--group", "-S", "--suffix"},
	"rsync": {"--exclude", "--include", "--filter", "-f", "--exclude-from", "--include-from",
		"--files-from", "-e", "--rsh", "--chmod", "--chown", "--usermap", "--groupmap",
		"--log-file", "--backup-dir", "--suffix", "--partial-dir", "-T", "--temp-dir",
		"--compare-dest", "--copy-dest", "--link-dest", "--max-size", "--min-size",
		"--bwlimit", "--timeout", "--port", "--password-file", "--rsync-path"},
}

var gitWritingSubcommands = map[string]bool{
	"apply": true, "am": true, "restore": true, "rm": true, "mv": true, "clean": true,
}

func (a *writeAnalysis) git(args []shWord, cwd string) {
	dir := cwd
	i := 0
	for ; i < len(args); i++ {
		t := args[i].text
		switch {
		case t == "-C" && i+1 < len(args):
			i++
			if args[i].dynamic {
				a.add(".", cwd)
				return
			}
			dir = absPath(args[i].text, dir)
		case t == "-c" && i+1 < len(args):
			i++
		case strings.HasPrefix(t, "-"):
		default:
			goto sub
		}
	}
	return
sub:
	name := args[i].text
	rest := args[i+1:]
	if gitWritingSubcommands[name] {
		if name == "mv" {
			a.hint = hintMove
		}
		a.add(".", dir)
		return
	}
	if name == "checkout" {
		for _, w := range rest {
			if w.text == "--" {
				a.add(".", dir)
				return
			}
		}
	}
}

// generator records a command that matches generators.allow and so must run
// through rgw-ast exec. openspec counts only for its mutating subcommands.
func (a *writeAnalysis) generator(words []shWord, cwd string) {
	if a.cfg == nil {
		return
	}
	argv := make([]string, len(words))
	for i, w := range words {
		if w.dynamic {
			return
		}
		argv[i] = w.text
	}
	ok, pattern := execgen.Allowed(*a.cfg, argv)
	if !ok {
		return
	}
	tokens := strings.Fields(pattern)
	if filepath.Base(tokens[len(tokens)-1]) == "openspec" {
		sub := ""
		if len(argv) > len(tokens) {
			sub = argv[len(tokens)]
		}
		if !openspecMutating[sub] {
			return
		}
	}
	a.generators = append(a.generators, genCall{cmd: strings.Join(argv, " "), cwd: cwd})
}

var openspecMutating = map[string]bool{"new": true, "archive": true, "init": true, "update": true}

// inlineCodeFlags introduce code on the command line, per interpreter.
var inlineCodeFlags = map[string][]string{
	"python": {"-c"},
	"node":   {"-e", "--eval", "-p", "--print"},
	"bun":    {"-e", "--eval", "-p", "--print"},
	"deno":   {"eval"},
	"ruby":   {"-e"},
	"perl":   {"-e", "-E"},
	"php":    {"-r"},
}

// writeCall matches file-writing calls in inline interpreter code.
var writeCall = regexp.MustCompile(`open\([^)]*,\s*(mode\s*=\s*)?['"][rbt]*[wax+]|open\s*\(?\s*\w+\s*,\s*['"]\+?[>|]|` +
	`write_text|write_bytes|os\.(remove|unlink|rename|replace|rmdir|truncate)\b|shutil\.|\bunlink\b|\.rename\(|\.rmdir\(|\.touch\(|` +
	`writeFile|appendFile|createWriteStream|rmSync|unlinkSync|renameSync|rmdirSync|copyFile|Deno\.(write|remove|rename|create)|Bun\.write|` +
	`File\.(write|open|delete|rename)|FileUtils\.|file_put_contents`)

// inlineWrites reports whether the interpreter's inline code writes files.
// Code whose text is unknown counts as writing.
func inlineWrites(name string, args []shWord) bool {
	for i, w := range args {
		for _, flag := range inlineCodeFlags[name] {
			if w.text == flag && i+1 < len(args) {
				return args[i+1].dynamic || writeCall.MatchString(args[i+1].text)
			}
		}
	}
	return false
}

// stripPrefixes drops assignments, reserved words, and wrappers that run the
// command that follows them.
func stripPrefixes(words []shWord) []shWord {
	for len(words) > 0 {
		w := words[0]
		t := w.text
		switch {
		case isAssignment(t):
			words = words[1:]
		case !w.quoted && isReservedWord(t):
			words = words[1:]
		case t == "sudo" || t == "env" || t == "command" || t == "builtin" || t == "exec" ||
			t == "nohup" || t == "time" || t == "stdbuf" || t == "nice" || t == "ionice":
			words = words[1:]
			for len(words) > 0 && (strings.HasPrefix(words[0].text, "-") || isAssignment(words[0].text)) {
				flag := words[0].text
				words = words[1:]
				if (flag == "-n" || flag == "-u" || flag == "-g") && len(words) > 0 {
					words = words[1:]
				}
			}
		case t == "timeout":
			words = words[1:]
			for len(words) > 0 && strings.HasPrefix(words[0].text, "-") {
				flag := words[0].text
				words = words[1:]
				if (flag == "-s" || flag == "-k" || flag == "--signal" || flag == "--kill-after") && len(words) > 0 {
					words = words[1:]
				}
			}
			if len(words) > 0 {
				words = words[1:] // duration
			}
		default:
			return words
		}
	}
	return words
}

func isAssignment(t string) bool {
	eq := strings.IndexByte(t, '=')
	if eq <= 0 {
		return false
	}
	for i := 0; i < eq; i++ {
		if !isNameByte(t[i]) {
			return false
		}
	}
	return !(t[0] >= '0' && t[0] <= '9')
}

func isReservedWord(t string) bool {
	switch t {
	case "!", "{", "}", "if", "then", "else", "elif", "fi", "do", "done", "while", "until", "time":
		return true
	}
	return false
}

// operands returns the non-option words, honouring "--".
func operands(args []shWord) []shWord {
	return operandsSkipping(args)
}

// operandsSkipping is operands where each listed option consumes the next word.
func operandsSkipping(args []shWord, valued ...string) []shWord {
	var out []shWord
	ended := false
	for i := 0; i < len(args); i++ {
		w := args[i]
		if ended || w.dynamic || !strings.HasPrefix(w.text, "-") || w.text == "-" {
			out = append(out, w)
			continue
		}
		if w.text == "--" {
			ended = true
			continue
		}
		for _, v := range valued {
			if w.text == v {
				i++
				break
			}
		}
	}
	return out
}

func optionValue(args []shWord, short, long string) (shWord, bool) {
	for i, w := range args {
		switch {
		case w.text == short && i+1 < len(args):
			return args[i+1], true
		case strings.HasPrefix(w.text, long+"="):
			return shWord{text: strings.TrimPrefix(w.text, long+"="), dynamic: w.dynamic}, true
		case w.text == long && i+1 < len(args):
			return args[i+1], true
		}
	}
	return shWord{}, false
}

// inPlaceFiles returns the files a sed/perl/ruby call edits in place, or
// nothing when it does not edit in place.
func inPlaceFiles(args []shWord) []shWord {
	inPlace := false
	script := false
	var ops []shWord
	for i := 0; i < len(args); i++ {
		t := args[i].text
		if args[i].dynamic || !strings.HasPrefix(t, "-") || t == "-" {
			ops = append(ops, args[i])
			continue
		}
		if t == "--in-place" || strings.HasPrefix(t, "--in-place=") {
			inPlace = true
			continue
		}
		if t == "--expression" || t == "--file" {
			script = true
			i++
			continue
		}
		if strings.HasPrefix(t, "--") {
			continue
		}
		cluster := t[1:]
		if k := strings.IndexByte(cluster, 'i'); k >= 0 {
			inPlace = true
			cluster = cluster[:k] // anything after i is a backup suffix
		}
		if strings.ContainsAny(cluster, "ef") {
			script = true
			if strings.HasSuffix(cluster, "e") || strings.HasSuffix(cluster, "f") {
				i++ // the option value is the next word
			}
		}
	}
	if !inPlace {
		return nil
	}
	if !script && len(ops) > 0 {
		ops = ops[1:] // the first operand is the script
	}
	return ops
}

// xargsCommand returns the command xargs runs, without xargs options.
func xargsCommand(args []shWord) []shWord {
	valued := map[string]bool{"-I": true, "-L": true, "-n": true, "-P": true, "-s": true, "-d": true, "-E": true, "-a": true}
	for i := 0; i < len(args); i++ {
		t := args[i].text
		if !strings.HasPrefix(t, "-") {
			return args[i:]
		}
		if valued[t] {
			i++
		}
	}
	return nil
}
