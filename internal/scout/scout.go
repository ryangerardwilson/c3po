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
	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/internal/xdg"
	"github.com/ryangerardwilson/c3po/l"
	"github.com/ryangerardwilson/c3po/o"
	"github.com/ryangerardwilson/c3po/pass"
	"github.com/ryangerardwilson/c3po/tunnel"
)

type runFunc func([]string, io.Writer, io.Writer) int

// apps maps each app name to its entrypoint.
var apps = map[string]runFunc{
	"ast":       ast.Run,
	"cyclotree": cyclotree.Run,
	"deport":    deport.Run,
	"gcal":      gcal.Run,
	"gdrive":    gdrive.Run,
	"gmail":     gmail.Run,
	"l":         l.Run,
	"o":         o.Run,
	"pass":      pass.Run,
	"tunnel":    tunnel.Run,
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

func isHelp(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func isVersion(s string) bool {
	return s == "version" || s == "--version" || s == "-version"
}
