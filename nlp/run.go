// Package nlp is the c3po tui nlp trainer.
package nlp

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/nlp/internal/app"
	"github.com/ryangerardwilson/c3po/nlp/internal/state"
)

// ScreenHelp is the help for c3po tui nlp.
const ScreenHelp = `c3po tui nlp
neuro-linguistic programming trainer

  c3po tui nlp
    open the course list
  c3po tui nlp help
    show this help

Courses stay in the sqlx database nlp.
On a course, part, or section menu, n creates the next item after the cursor.
,rn renames the selected item. dd deletes it.
? lists the shortcuts. q leaves a menu. Q quits.
`

var (
	runApp = app.RunSession
	termOK = terminal
)

// Run serves c3po tui nlp. args are the arguments after the screen name.
func Run(args []string, stdout, stderr io.Writer) int {
	id, rest, err := tuisession.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, ScreenHelp)
		return 2
	}
	args = rest
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, ScreenHelp)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if len(args) != 0 {
		fmt.Fprint(stderr, ScreenHelp)
		return 2
	}
	if id == "" && !termOK(stdout) {
		fmt.Fprint(stdout, ScreenHelp)
		return 0
	}
	if !termOK(stdout) {
		fmt.Fprintln(stderr, "nlp needs a terminal")
		return 1
	}
	courses, err := state.LoadCourses(version.String())
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if len(courses) == 0 {
		fmt.Fprintln(stderr, "No valid courses found.")
		return 1
	}
	sess, err := tuisession.Open("nlp", id)
	if err != nil {
		fmt.Fprintf(stderr, "nlp: %s\n", err.Error())
		return 1
	}
	defer sess.Close()
	return runApp(app.Config{Courses: courses}, sess)
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func isVersion(arg string) bool {
	return arg == "version" || arg == "-v" || arg == "--version"
}

func terminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
