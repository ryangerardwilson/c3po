package scout

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/ast"
	"github.com/ryangerardwilson/c3po/clippy"
	"github.com/ryangerardwilson/c3po/deport"
	"github.com/ryangerardwilson/c3po/dirt"
	"github.com/ryangerardwilson/c3po/gates"
	"github.com/ryangerardwilson/c3po/gcal"
	"github.com/ryangerardwilson/c3po/gdrive"
	"github.com/ryangerardwilson/c3po/gmail"
	"github.com/ryangerardwilson/c3po/idealake"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/internal/usage"
	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/internal/xdg"
	"github.com/ryangerardwilson/c3po/looptab"
	"github.com/ryangerardwilson/c3po/mess"
	"github.com/ryangerardwilson/c3po/notify"
	"github.com/ryangerardwilson/c3po/pass"
	"github.com/ryangerardwilson/c3po/slack"
	"github.com/ryangerardwilson/c3po/tunnel"
)

type runFunc func([]string, io.Writer, io.Writer) int

// apps maps each app name to its entrypoint.
var apps = map[string]runFunc{
	"ast":      ast.Run,
	"clippy":   clippy.Run,
	"deport":   deport.Run,
	"dirt":     dirt.Run,
	"gates":    gates.Run,
	"gcal":     gcal.Run,
	"gdrive":   gdrive.Run,
	"gmail":    gmail.Run,
	"idealake": idealake.Run,
	"looptab":  looptab.Command,
	"mess":     mess.Run,
	"notify":   notify.Run,
	"pass":     pass.Run,
	"slack":    slack.Run,
	"tunnel":   tunnel.Run,
}

// Main runs c3po from process arguments.
func Main(args []string) int {
	if len(args) >= 2 {
		switch args[1] {
		case "__usage-backup":
			usage.FlushBackup()
			return 0
		case "__usage-restore":
			usage.RestoreRuntime()
			return 0
		case "__session":
			return tuisession.Command(args[2:], os.Stdout, os.Stderr)
		case "__snapshot":
			if err := sqlxdb.MarkChanged(); err != nil {
				fmt.Fprintf(os.Stderr, "c3po: snapshot: %s\n", err)
				return 1
			}
			return 0
		}
	}
	return Run(args, os.Stdout, os.Stderr)
}

// Run dispatches one app from the program name or the c3po arguments.
// The usage note is recorded after the command returns. A failed note does
// not change the exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	code := dispatch(args, stdout, stderr)
	usage.Note(code, args)
	return code
}

func dispatch(args []string, stdout, stderr io.Writer) int {
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
	if args[0] == "usage" {
		return usageCommand(args[1:], stdout, stderr)
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

func usageCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: c3po usage [prefix]")
		return 2
	}
	prefix := ""
	if len(args) == 1 {
		prefix = args[0]
	}
	text, err := usage.Report(prefix)
	if err != nil {
		fmt.Fprintf(stderr, "c3po: usage: %s\n", err)
		return 1
	}
	fmt.Fprint(stdout, text)
	return 0
}

func isHelp(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func isVersion(s string) bool {
	return s == "version" || s == "--version" || s == "-version"
}
