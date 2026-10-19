package cyclotree

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const helpText = `c3po cyclotree shows calls from one function, or from the application entrypoint.

Usage:
  c3po cyclotree [symbol] [dir]
  c3po cyclotree help

With no symbol, the tree starts at the program entrypoint and includes
every call reached from it. A Go entrypoint is func main in package main.
A Python, JavaScript, or TypeScript entrypoint is a function named main.
A Rust fn main can be the root and shows its complexity. Its calls are not
indexed. When more than one entrypoint exists, the command lists each
path:line name. When none exists, the command exits 1.

<symbol> is a function name. When more than one function has that name,
pass path:name, using the file path the command prints. [dir] is the
directory to read and defaults to the working directory. It may come
before or after the symbol.

Each row is a function, its cyclomatic complexity, and how many levels
of calls can be opened under it. A call back to a function already on the
path does not add a level. The root row shows its path from the
directory being read. A nested row shows its path relative to the caller.
The bottom of the screen shows the call chain from the root down to
the row under the cursor, then the full path of that row. A named
root's calls start open. From an entrypoint, every reached call
starts open. A function already opened on
another branch stays closed. j and k move among the visible rows. l opens
the calls of a closed row, and moves into the first call of an open row.
h moves to the caller of the row under the cursor. c collapses that row
and everything open under it. A call back to a function already on the
path is marked cycle and stays closed. A row that can be opened is bright.
A row with nothing under it is dim. The screen is yellow. While the
screen is open, the tree updates as source files change. Open and closed
rows stay as they were, and the cursor stays on the same function.

Go, Python, TypeScript, and JavaScript calls are followed. A call into a
directory named internal is listed like any other call.

enter opens the file under the cursor in vim at that line. When vim
exits, the tree returns. ? toggles the shortcut list. The screen does
not show it until then. q quits the screen. When stdout is not a
terminal, the open tree is printed and the command exits.

Exit 0 when the tree is shown, 1 when the entrypoint, symbol, or directory
cannot be read, 2 on bad usage.
`

func Run(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "cyclotree: %s\n", err.Error())
		return 1
	}
	return runAt(args, cwd, stdout, stderr)
}

func runAt(args []string, cwd string, stdout, stderr io.Writer) int {
	opt, err := parseArgs(args, cwd)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		var usage usageError
		if errors.As(err, &usage) {
			fmt.Fprint(stderr, helpText)
			return 2
		}
		return 1
	}
	if opt.help {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	root, err := resolveDir(cwd, opt.dir)
	if err != nil {
		fmt.Fprintf(stderr, "cyclotree: %s\n", err.Error())
		return 1
	}
	screen, err := openTree(root, opt.symbol)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if f, ok := interactive(stdout); ok {
		if err := runTUI(screen, root, opt.symbol, os.Stdin, f); err != nil {
			fmt.Fprintf(stderr, "cyclotree: %s\n", err.Error())
			return 1
		}
		return 0
	}
	fmt.Fprint(stdout, screen.Text(false))
	return 0
}

type options struct {
	dir, symbol string
	help        bool
}

type usageError string

func (e usageError) Error() string { return string(e) }

func parseArgs(args []string, cwd string) (options, error) {
	opt := options{dir: cwd}
	for _, a := range args {
		if err := takeArg(cwd, a, &opt); err != nil {
			return options{}, err
		}
	}
	return opt, nil
}

func takeArg(cwd, a string, opt *options) error {
	if a == "help" || a == "-h" || a == "--help" {
		opt.help = true
		return nil
	}
	if strings.HasPrefix(a, "-") {
		return usageError(fmt.Sprintf("cyclotree: unknown flag %s", a))
	}
	switch kindOf(cwd, a) {
	case "dir":
		if opt.dir != cwd {
			return usageError("cyclotree: two directories")
		}
		opt.dir = a
	case "badpath":
		return fmt.Errorf("cyclotree: %s is not a directory", a)
	default:
		if opt.symbol != "" {
			return usageError("cyclotree: two symbols")
		}
		opt.symbol = a
	}
	return nil
}

func kindOf(cwd, a string) string {
	if isDir(cwd, a) {
		return "dir"
	}
	if strings.Contains(a, ":") {
		return "symbol"
	}
	if a == "." || a == ".." || strings.ContainsAny(a, `/\`) {
		return "badpath"
	}
	return "symbol"
}

func isDir(cwd, a string) bool {
	path := a
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func resolveDir(cwd, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}
	return abs, nil
}

func interactive(stdout io.Writer) (*os.File, bool) {
	f, ok := stdout.(*os.File)
	if !ok || !isCharDevice(f) || !isCharDevice(os.Stdin) {
		return nil, false
	}
	return f, true
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

const refreshWait = 250 * time.Millisecond

func runTUI(s *screen, dir, query string, in, out *os.File) error {
	restore, err := enterRaw(int(in.Fd()))
	if err != nil {
		fmt.Fprint(out, s.Text(false))
		return nil
	}
	defer func() {
		if restore != nil {
			restore()
		}
	}()
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[0m\x1b[?25h\x1b[?1049l")
	return tuiLoop(s, dir, query, in, out, &restore)
}

func tuiLoop(s *screen, dir, query string, in, out *os.File, restore *func()) error {
	stamp := sourceStamp(dir)
	var pending string
	for {
		paint(out, s.Frame(termRows(int(out.Fd()))))
		k, ok, err := readKeyWait(in, refreshWait)
		if err != nil {
			return err
		}
		if ok {
			switch applyKeys(s, burst(in, k)) {
			case actQuit:
				return nil
			case actEdit:
				if err := editRow(dir, s, in, out, restore); err != nil {
					return err
				}
			}
			continue
		}
		s, stamp, pending = settle(s, dir, query, stamp, pending)
	}
}

func burst(in *os.File, first rune) []rune {
	keys := []rune{first}
	for len(keys) < 256 {
		k, ok := nextKey(in)
		if !ok {
			return keys
		}
		keys = append(keys, k)
	}
	return keys
}

type keyAction int

const (
	actNone keyAction = iota
	actQuit
	actEdit
)

func applyKeys(s *screen, keys []rune) keyAction {
	for _, k := range keys {
		switch k {
		case 'q', 'Q', 3:
			return actQuit
		case '\r', '\n':
			if s != nil && s.shortcuts {
				break
			}
			if _, _, ok := cursorFile(s); ok {
				return actEdit
			}
		case 0:
		default:
			s.Key(k)
		}
	}
	return actNone
}
