package scout

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/ast"
	"github.com/ryangerardwilson/c3po/cyclotree"
	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/vibetree"
)

// Main runs c3po from process arguments.
func Main(args []string) int {
	return Run(args, os.Stdout, os.Stderr)
}

// Run dispatches one scout from the program name or from c3po scout.
func Run(args []string, stdout, stderr io.Writer) int {
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
	if len(args) == 0 || isHelp(args[0]) {
		writeHelp(stdout)
		return 0
	}
	if isVersion(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if args[0] != "scout" {
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		writeHelp(stderr)
		return 2
	}
	return runScout(args[1:], stdout, stderr)
}

func runScout(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		writeScouts(stdout)
		return 0
	}
	switch args[0] {
	case "ast":
		return ast.Run(args[1:], stdout, stderr)
	case "vibetree":
		return vibetree.Run(args[1:], stdout, stderr)
	case "cyclotree":
		return cyclotree.Run(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown scout %q\n", args[0])
		writeScouts(stderr)
		return 2
	}
}

func isHelp(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func isVersion(s string) bool {
	return s == "version" || s == "--version" || s == "-version"
}
