// Package screen opens one terminal screen by name.
// c3po does not dispatch these names. r2d2 calls Run.
package screen

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/cyclotree"
	"github.com/ryangerardwilson/c3po/deport"
	"github.com/ryangerardwilson/c3po/gdrive"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/l"
	"github.com/ryangerardwilson/c3po/looptab"
	"github.com/ryangerardwilson/c3po/o"
	"github.com/ryangerardwilson/c3po/pass"
	"github.com/ryangerardwilson/c3po/tunnel"
	"github.com/ryangerardwilson/c3po/xl"
)

type runFunc func([]string, io.Writer, io.Writer) int

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
}

// Run opens the named screen. An unknown name exits 2.
func Run(name string, args []string, stdout, stderr io.Writer) int {
	run, ok := screens[name]
	if !ok {
		fmt.Fprintf(stderr, "unknown screen %q\n", name)
		return 2
	}
	return run(args, stdout, stderr)
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
