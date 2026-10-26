// Package l is the c3po tui l: a compact colored tree listing.
package l

import (
	"fmt"
	"io"
	"os"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/l/internal/tree"
)

// Help is the full help for c3po tui l.
const Help = `c3po tui l shows a compact tree view for seeing refactoring damage.

Usage:
  c3po tui l [path] [flags]
  c3po tui l help

Flags:
  -a, --all              include hidden files and directories
  -f, --files            show text files only; hide binaries and empty dirs
  -d, --depth <n|*>      max tree depth from root, or * for unlimited (default 2)
      --level <n|*>      same as --depth
      --ignore <pattern> add a basename or glob ignore; repeat or comma-separate
      --no-ignore        disable default dependency/cache/build ignores
      --color <mode>     auto, always, or never (default auto)
      --no-color         disable color
      --animate <mode>   auto, always, or never (default auto)
      --no-animate       disable tree animation
  -h, --help             show this help
      --version          show the c3po version

Value flags also take the --flag=value form. The path may come before or
after the flags. Use -- to end flags; every argument after it is a path.

Each file shows its size and line count. Each binary shows its size.
Directories sort before files, then names sort without regard to case.

Directories render white, text files yellow, and binaries cyan. Tree
glyphs render gray. Text-file brightness scales with line count on a log
scale, and binary brightness scales with size the same way. Files under
200L and binaries under 100KB share one low-intensity color. When no
entry of a kind is above its threshold, that kind uses one middle
intensity.

Color is auto by default: it is emitted only when stdout is a terminal.
Animation is auto by default: on a terminal that is not TERM=dumb, each
line renders as a quick branch-then-label cascade. Redirected or piped
output is plain and deterministic.

Hidden entries (names that start with a dot) are skipped unless -a is
given. Default ignored directories include node_modules, __pycache__,
.git, bin, dist, build, target, coverage, env, venv, .venv, .next,
.turbo, and common cache dirs. .DS_Store files are ignored too. A
directory named internal is listed like any other directory.

help, -h, --help, version, and --version are read as commands wherever
they appear. To list a directory named help or version, pass ./help or
put it after --.

Exit 0 when the tree is printed, 1 when the path cannot be read, 2 on bad
usage.
`

// Run runs c3po tui l. args are the arguments after "l".
func Run(args []string, stdout, stderr io.Writer) int {
	id, rest, rerr := tuisession.TakeResume(args)
	if rerr != nil {
		fmt.Fprintf(stderr, "l: %s\n\n", rerr)
		fmt.Fprint(stderr, Help)
		return 2
	}
	args = rest
	var sess *tuisession.Session
	if id != "" && len(args) == 0 {
		var err error
		sess, err = tuisession.Open("l", id)
		if err != nil {
			fmt.Fprintf(stderr, "l: %s\n", err.Error())
			return 1
		}
		defer sess.Close()
		if body, ok := sess.Restored(); ok && body.Args != nil {
			args = body.Args
		}
	}
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "l: %s\n\n", err)
		fmt.Fprint(stderr, Help)
		return 2
	}
	if opts.help {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if opts.version {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if sess == nil && (id != "" || tuisession.Interactive(stdout)) {
		sess, err = tuisession.Open("l", id)
		if err != nil {
			fmt.Fprintf(stderr, "l: %s\n", err.Error())
			return 1
		}
		defer sess.Close()
	}
	if sess != nil {
		_ = sess.Note(tuisession.Body{Path: opts.config.Root, Args: append([]string(nil), args...)})
	}
	return list(opts, stdout, stderr)
}

func list(opts options, stdout, stderr io.Writer) int {
	root, stats, err := tree.Build(opts.config)
	if err != nil {
		fmt.Fprintf(stderr, "l: %s\n", err)
		return 1
	}
	cfg := tree.RenderConfig{
		Color:          shouldColor(stdout, opts.colorMode),
		Animate:        shouldAnimate(stdout, opts.animationMode),
		AnimationDelay: tree.DefaultAnimationDelay,
	}
	if err := tree.Render(stdout, root, stats, cfg); err != nil {
		fmt.Fprintf(stderr, "l: %s\n", err)
		return 1
	}
	return 0
}

func shouldColor(out io.Writer, mode tree.ColorMode) bool {
	switch mode {
	case tree.ColorAlways:
		return true
	case tree.ColorNever:
		return false
	default:
		return isTerminal(out)
	}
}

func shouldAnimate(out io.Writer, mode tree.AnimationMode) bool {
	switch mode {
	case tree.AnimationAlways:
		return true
	case tree.AnimationNever:
		return false
	default:
		return isTerminal(out) && os.Getenv("TERM") != "dumb"
	}
}

// isTerminal reports whether out is an *os.File on a character device.
func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
