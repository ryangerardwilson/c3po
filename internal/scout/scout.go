package scout

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/ast"
	"github.com/ryangerardwilson/c3po/cyclotree"
	"github.com/ryangerardwilson/c3po/deport"
	"github.com/ryangerardwilson/c3po/gcal"
	"github.com/ryangerardwilson/c3po/gdrive"
	"github.com/ryangerardwilson/c3po/gmail"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/internal/xdg"
	"github.com/ryangerardwilson/c3po/l"
	"github.com/ryangerardwilson/c3po/looptab"
	"github.com/ryangerardwilson/c3po/nlp"
	"github.com/ryangerardwilson/c3po/o"
	"github.com/ryangerardwilson/c3po/pass"
	"github.com/ryangerardwilson/c3po/slack"
	"github.com/ryangerardwilson/c3po/tunnel"
	"github.com/ryangerardwilson/c3po/xl"
)

type runFunc func([]string, io.Writer, io.Writer) int

// apps maps each app name to its entrypoint.
var apps = map[string]runFunc{
	"ast":     ast.Run,
	"deport":  deport.Run,
	"gcal":    gcal.Run,
	"gdrive":  gdrive.Run,
	"gmail":   gmail.Run,
	"looptab": looptab.Command,
	"pass":    pass.Run,
	"slack":   slack.Run,
	"tunnel":  tunnel.Run,
	"tui":     dispatchTUI,
}

var screens = map[string]runFunc{
	"cyclotree": cyclotree.Run,
	"l":         l.Run,
	"o":         o.Run,
	"tunnel":    tunnelScreen,
	"gdrive":    gdriveScreen,
	"pass":      passScreen,
	"deport":    deportScreen,
	"xl":        xl.Run,
	"looptab":   looptab.Run,
	"nlp":       nlp.Run,
}

// Main runs c3po from process arguments.
func Main(args []string) int {
	return Run(args, os.Stdout, os.Stderr)
}

// Run dispatches one app from the program name or the c3po arguments.
func Run(args []string, stdout, stderr io.Writer) int {
	if err := xdg.Migrate(); err != nil {
		fmt.Fprintf(stderr, "c3po: config: %s\n", err)
		return 1
	}
	if len(args) == 0 {
		writeHelp(stdout)
		return 0
	}
	switch toolOf(args[0]) {
	case "ast":
		return ast.Run(args[1:], stdout, stderr)
	default:
		return c3po(args[1:], stdout, stderr)
	}
}

func toolOf(argv0 string) string {
	if filepath.Base(argv0) == "rgw-ast" {
		return "ast"
	}
	return ""
}

func c3po(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (isHelp(args[0]) && len(args) == 1) {
		writeHelp(stdout)
		return 0
	}
	if args[0] == "help" {
		return helpArgs(args[1:], stdout, stderr)
	}
	if isVersion(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	return runApp(args, stdout, stderr)
}

func runApp(args []string, stdout, stderr io.Writer) int {
	run, ok := apps[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		writeHelp(stderr)
		return 2
	}
	return run(args[1:], stdout, stderr)
}

func dispatchTUI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && isHelp(args[0])) {
		writeTUI(stdout)
		return 0
	}
	run, ok := screens[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "unknown tui %q\n", args[0])
		writeTUI(stderr)
		return 2
	}
	return run(args[1:], stdout, stderr)
}

func tunnelScreen(args []string, stdout, stderr io.Writer) int {
	id, rest, err := tuisession.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, tunnel.ScreenHelp)
		return 2
	}
	args = rest
	if len(args) == 0 {
		return tunnel.ScreenResume(id, stdout, stderr)
	}
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, tunnel.ScreenHelp)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		return tunnel.Run([]string{"version"}, stdout, stderr)
	}
	fmt.Fprint(stderr, tunnel.ScreenHelp)
	return 2
}

func gdriveScreen(args []string, stdout, stderr io.Writer) int {
	id, rest, err := tuisession.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, gdrive.ScreenHelp)
		return 2
	}
	args = rest
	if len(args) == 0 {
		return gdrive.ScreenResume(id, stdout, stderr)
	}
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, gdrive.ScreenHelp)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		return gdrive.Run([]string{"version"}, stdout, stderr)
	}
	fmt.Fprint(stderr, gdrive.ScreenHelp)
	return 2
}

func deportScreen(args []string, stdout, stderr io.Writer) int {
	id, rest, err := tuisession.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, deport.ScreenHelp)
		return 2
	}
	args = rest
	if len(args) == 0 {
		return deport.ScreenResume(id, stdout, stderr)
	}
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, deport.ScreenHelp)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		return deport.Run([]string{"version"}, stdout, stderr)
	}
	fmt.Fprint(stderr, deport.ScreenHelp)
	return 2
}

func passScreen(args []string, stdout, stderr io.Writer) int {
	id, rest, err := tuisession.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, pass.ScreenHelp)
		return 2
	}
	args = rest
	if len(args) == 0 {
		return pass.ScreenResume(id, stdout, stderr)
	}
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, pass.ScreenHelp)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		return pass.Run([]string{"version"}, stdout, stderr)
	}
	fmt.Fprint(stderr, pass.ScreenHelp)
	return 2
}

func isHelp(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func isVersion(s string) bool {
	return s == "version" || s == "--version" || s == "-version"
}
