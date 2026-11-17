// Package clippy copies a multiline text to the Wayland clipboard.
package clippy

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// Help is the text for c3po clippy and c3po help -f clippy.
const Help = `c3po clippy
copy a multiline text to the clipboard

  c3po clippy
    read stdin until EOF and copy it
  c3po clippy <text...>
    copy those words as one string
  c3po clippy help
    show this help
  c3po clippy version
    print the c3po version

The text is not printed. Success prints copied.
On a terminal with no text, this help is printed instead.
`

// Run serves c3po clippy. args are the words after clippy.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, os.Stdin, WLCopy{})
}

func run(args []string, stdout, stderr io.Writer, stdin io.Reader, board Clipboard) int {
	if len(args) == 1 && helpWord(args[0]) {
		return showHelp(stdout)
	}
	if len(args) == 1 && versionWord(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if len(args) == 0 && terminal(stdin) {
		return showHelp(stdout)
	}
	text, err := payload(args, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "clippy: %s\n", err)
		return 1
	}
	if text == "" {
		fmt.Fprintln(stderr, "usage: c3po clippy [text...]")
		fmt.Fprint(stderr, Help)
		return 2
	}
	return copyText(text, stdout, stderr, board)
}

func showHelp(stdout io.Writer) int {
	fmt.Fprint(stdout, Help)
	return 0
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}

func terminal(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func payload(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func copyText(text string, stdout, stderr io.Writer, board Clipboard) int {
	if err := board.Copy(text, stderr); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fmt.Fprintln(stdout, "copied")
	return 0
}
