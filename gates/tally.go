package gates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// verdict is one good sheet line: a provision judged against this build.
type verdict struct {
	section string
	fail    bool
	files   []string
}

// sheet is what a tally sheet says: the good line for each section and a
// report for each bad line.
type sheet struct {
	judged map[string]verdict
	bad    []string
}

// runTally serves c3po gates <type> <statute> tally <sheet>. It counts the
// sheet; it never decides whether the work passes.
func runTally(args []string, k kind, s statute, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "tally takes one sheet path")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "gates: cannot read sheet %s: %v\n", args[0], err)
		return 2
	}
	changed, err := changedFiles()
	if err != nil {
		fmt.Fprintf(stderr, "gates: git status failed: %v\n", err)
		return 2
	}
	return readSheet(s, string(data)).report(stdout, k, s, changed)
}

// readSheet reads one verdict per line. A blank line is skipped. Any other
// line that does not parse, or repeats a section, is reported as bad.
func readSheet(s statute, text string) sheet {
	sh := sheet{judged: map[string]verdict{}}
	seen := map[string]bool{}
	for i, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		v, err := parseVerdict(s, fields)
		if p, ok := s.find(fields[0]); ok && seen[p.section] {
			err = fmt.Errorf("duplicate %s", p.section)
		} else if ok {
			seen[p.section] = true
		}
		if err != nil {
			sh.bad = append(sh.bad, fmt.Sprintf("bad line %d: %v: %s", i+1, err, strings.TrimSpace(line)))
			continue
		}
		sh.judged[v.section] = v
	}
	return sh
}

// parseVerdict reads §1.2 <receipt> <pass|fail> <site> [files <path>...].
func parseVerdict(s statute, f []string) (verdict, error) {
	p, ok := s.find(f[0])
	switch {
	case !ok:
		return verdict{}, fmt.Errorf("unknown section %q", f[0])
	case len(f) < 2 || f[1] != receipt(p):
		return verdict{}, errors.New("bad receipt")
	case len(f) < 3 || (f[2] != "pass" && f[2] != "fail"):
		return verdict{}, errors.New("verdict is not pass or fail")
	case len(f) < 4:
		return verdict{}, errors.New("empty site")
	}
	if err := checkSite(f[3]); err != nil {
		return verdict{}, err
	}
	rest := f[4:]
	if len(rest) > 0 && (rest[0] != "files" || len(rest) == 1) {
		return verdict{}, errors.New("extra words")
	}
	v := verdict{section: p.section, fail: f[2] == "fail"}
	for _, path := range rest[min(1, len(rest)):] {
		v.files = append(v.files, filepath.Clean(path))
	}
	return v, nil
}

// checkSite accepts none or path:line, where path exists relative to the
// current directory and has that line.
func checkSite(site string) error {
	if site == "none" {
		return nil
	}
	cut := strings.LastIndex(site, ":")
	if cut < 1 {
		return fmt.Errorf("site %s is not none or path:line", site)
	}
	path := site[:cut]
	n, err := strconv.Atoi(site[cut+1:])
	if err != nil || n < 1 {
		return fmt.Errorf("site %s is not none or path:line", site)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("missing path %s", path)
	}
	if n > lineCount(data) {
		return fmt.Errorf("%s has no line %d", path, n)
	}
	return nil
}

// lineCount counts lines, including a last line with no newline.
func lineCount(data []byte) int {
	n := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n
}

// report prints the count, then the missing, unfixed, and unclaimed lines
// and the bad lines when there are any. It exits 0 only for a complete,
// valid sheet that claims every changed file.
func (sh sheet) report(w io.Writer, k kind, s statute, changed []string) int {
	fmt.Fprintf(w, "%d of %d judged\n", len(sh.judged), len(s.provisions))
	var missing []provision
	var missed, unfixed, unclaimed []string
	claimed := map[string]bool{}
	for _, p := range s.provisions {
		v, ok := sh.judged[p.section]
		if !ok {
			missing = append(missing, p)
			missed = append(missed, p.section)
			continue
		}
		if v.fail && len(v.files) == 0 {
			unfixed = append(unfixed, p.section)
		}
		for _, f := range v.files {
			claimed[f] = true
		}
	}
	for _, f := range changed {
		if !claimed[f] {
			unclaimed = append(unclaimed, f)
		}
	}
	if len(missing) > 0 {
		printList(w, "missing", missed)
		fmt.Fprintf(w, "c3po gates %s %s %s\n", k.name, s.name, bareSections(missing))
	}
	printList(w, "unfixed", unfixed)
	printList(w, "unclaimed", unclaimed)
	for _, b := range sh.bad {
		fmt.Fprintln(w, b)
	}
	if len(sh.bad) > 0 || len(missing) > 0 || len(unclaimed) > 0 || (len(changed) > 0 && len(unfixed) > 0) {
		return 2
	}
	return 0
}

func printList(w io.Writer, label string, words []string) {
	if len(words) > 0 {
		fmt.Fprintln(w, label+" "+strings.Join(words, " "))
	}
}

// changedFiles lists the tracked files git reports as changed, relative to
// the current directory. Untracked files are ignored. A renamed file is its
// new path; a deleted file still counts.
func changedFiles() ([]string, error) {
	prefix, err := gitOutput("rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	out, err := gitOutput("status", "--porcelain", "-z", "--untracked-files=no")
	if err != nil {
		return nil, err
	}
	base := filepath.Clean(strings.TrimSpace(prefix))
	var paths []string
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 4 || e[:2] == "??" || e[:2] == "!!" {
			continue
		}
		if strings.ContainsAny(e[:2], "RC") {
			i++ // the original path of a rename or copy follows
		}
		rel, err := filepath.Rel(base, filepath.FromSlash(e[3:]))
		if err != nil {
			return nil, err
		}
		paths = append(paths, rel)
	}
	return paths, nil
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(errb.String()); msg != "" {
			return "", errors.New(msg)
		}
		return "", err
	}
	return string(out), nil
}
