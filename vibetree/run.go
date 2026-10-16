// Package vibetree prints a size tree of files and directories.
package vibetree

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	modeLoc   = "loc"
	modeCyclo = "cyclo"
)

const helpText = `c3po scout vibetree prints a size tree for a directory.

Usage:
  c3po scout vibetree <dir> [-p N] [-m loc|cyclo] [--json]
  c3po scout vibetree help

<dir> is the directory to read. It may come before or after the flags. The
default is the working directory. -p is an integer from 1 to 99 and defaults
to 95. -m is loc or cyclo and defaults to loc.

A module is a file or a directory that contains one. It is not a language
package, crate, or TypeScript module. A directory's size is the sum of the
files under it.

loc counts lines in any text file. Binary files are omitted.

cyclo sums the cyclomatic complexity of the functions in a file. Go, Python,
Rust, TypeScript, and JavaScript are scored. JavaScript uses the TypeScript
reader. A file with no scored function is omitted.

In a git work tree the file list is git ls-files, tracked and untracked,
honoring gitignore. Otherwise the walk skips VCS and dependency directories.

The report names the mode, the module count, and the chosen percentile, then
a tree of the modules at or above that percentile. When that percentile is
the largest module, the report gives the rank and asks for a lower -p.
--json prints that same cutoff as one JSON object, with the same note. Tree lines use a yellow scale: brighter means
a larger size. In loc mode a module of 200 lines or fewer stays dim when a
larger one is shown. In cyclo mode a size of 1 stays dim the same way.
Color is auto on a terminal. --color=always and --no-color override it.
Exit 0 when the tree can be read, 1 when the path cannot, 2 on bad flags.
`

// Run executes c3po scout vibetree from the working directory.
func Run(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "vibetree: %s\n", err.Error())
		return 1
	}
	return runAt(args, cwd, stdout, stderr)
}

func runAt(args []string, cwd string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelp(args[0]) {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	opt, err := parse(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		fmt.Fprint(stderr, helpText)
		return 2
	}
	root, err := resolve(cwd, opt.path)
	if err != nil {
		fmt.Fprintf(stderr, "vibetree: %s\n", err.Error())
		return 1
	}
	files, err := collect(root, measureOf(opt.mode))
	if err != nil {
		fmt.Fprintf(stderr, "vibetree: %s\n", err.Error())
		return 1
	}
	built := build(files)
	if opt.json {
		writeJSONReport(stdout, built, opt.p, opt.mode)
		return 0
	}
	writeReport(stdout, built, opt.p, opt.mode, shouldColor(stdout, opt.color))
	return 0
}

func measureOf(mode string) measure {
	if mode == modeCyclo {
		return cycloOf
	}
	return fileLines
}

func isHelp(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

type options struct {
	p     int
	path  string
	color string
	mode  string
	json  bool
}

func parse(args []string) (options, error) {
	opt := options{p: 95, color: colorAuto, mode: modeLoc}
	for i := 0; i < len(args); i++ {
		next, err := one(args, i, &opt)
		if err != nil {
			return options{}, err
		}
		i = next
	}
	return opt, nil
}

func one(args []string, i int, opt *options) (int, error) {
	a := args[i]
	switch {
	case isPercentile(a):
		return takePercentile(args, i, opt)
	case isColor(a):
		return takeColor(args, i, opt)
	case isMode(a):
		return takeMode(args, i, opt)
	case isJSON(a):
		opt.json = true
		return i, nil
	case strings.HasPrefix(a, "-"):
		return i, fmt.Errorf("vibetree: unknown flag %q", a)
	case opt.path != "":
		return i, fmt.Errorf("vibetree: one path only")
	default:
		opt.path = a
		return i, nil
	}
}

func isPercentile(a string) bool {
	return a == "-p" || a == "--percentile" || strings.HasPrefix(a, "-p")
}

func isColor(a string) bool {
	return a == "--no-color" || a == "--color" || strings.HasPrefix(a, "--color=")
}

func isMode(a string) bool {
	if a == "-m" || a == "--mode" || strings.HasPrefix(a, "--mode=") {
		return true
	}
	return strings.HasPrefix(a, "-m") && !strings.HasPrefix(a, "--")
}

func isJSON(a string) bool {
	return a == "--json"
}

func takePercentile(args []string, i int, opt *options) (int, error) {
	raw, i, err := flagValue(args, i, "-p", "--percentile")
	if err != nil {
		return i, fmt.Errorf("vibetree: -p needs a percentile from 1 to 99")
	}
	n, err := percentileArg(raw)
	if err != nil {
		return i, err
	}
	opt.p = n
	return i, nil
}

func takeColor(args []string, i int, opt *options) (int, error) {
	if args[i] == "--no-color" {
		opt.color = colorNever
		return i, nil
	}
	raw, i, err := flagValue(args, i, "--color", "")
	if err != nil {
		return i, fmt.Errorf("vibetree: --color needs auto, always, or never")
	}
	if err = setColor(opt, raw); err != nil {
		return i, err
	}
	return i, nil
}

func takeMode(args []string, i int, opt *options) (int, error) {
	raw, i, err := flagValue(args, i, "-m", "--mode")
	if err != nil || raw != modeLoc && raw != modeCyclo {
		return i, fmt.Errorf("vibetree: -m needs loc or cyclo")
	}
	opt.mode = raw
	return i, nil
}

// flagValue reads --name value, or the suffix of --name=value or -p90.
func flagValue(args []string, i int, name, also string) (string, int, error) {
	a := args[i]
	if a == name || (also != "" && a == also) {
		if i+1 >= len(args) {
			return "", i, fmt.Errorf("missing")
		}
		return args[i+1], i + 1, nil
	}
	if strings.HasPrefix(a, name+"=") {
		return strings.TrimPrefix(a, name+"="), i, nil
	}
	if also != "" && strings.HasPrefix(a, also+"=") {
		return strings.TrimPrefix(a, also+"="), i, nil
	}
	if strings.HasPrefix(a, name) {
		return strings.TrimPrefix(a, name), i, nil
	}
	return "", i, fmt.Errorf("missing")
}

func percentileArg(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 99 {
		return 0, fmt.Errorf("vibetree: -p needs a percentile from 1 to 99")
	}
	return n, nil
}
