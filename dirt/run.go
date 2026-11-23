// Package dirt lists the dirty git repositories under a directory.
package dirt

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// Help is the text for c3po dirt and c3po help -f dirt.
const Help = `c3po dirt
list every dirty git repository under a directory

  c3po dirt
    scan the home directory
  c3po dirt <path>
    scan that directory; a relative path starts at the working directory
  c3po dirt help
    show this help
  c3po dirt version
    print the c3po version

Stdout lists only dirty repositories, sorted by absolute path, with one blank
line between them. Each is its absolute path, then git status --porcelain=v1 -b.
A repository whose status is only the ## line is clean and is not printed.
A scan path inside a repository shows that repository with only the paths
under the scan path.

A directory holding .git is a repository. Nested repositories are their own
entries. Symlinks are not followed. Directories that cannot be listed are
skipped, and so are node_modules, .venv, venv, .cache, __pycache__, .npm,
.cargo, and site-packages below the scan path.

Exit 0 when every repository was judged. A bare repository or a failed
git status is named on stderr and exits 1. A usage error exits 2.
Nothing is staged, committed, cleaned, or checked out. There is no dirt screen.
`

const usageLine = "usage: c3po dirt [<path>]"

// Run serves c3po dirt. args are the words after dirt.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && helpWord(args[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(args) == 1 && versionWord(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if len(args) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(stderr, "dirt: %s\n", err)
			return 1
		}
		args = []string{home}
	}
	root, err := scanRoot(args)
	if err != nil {
		fmt.Fprintf(stderr, "dirt: %s\n", err)
		fmt.Fprintln(stderr, usageLine)
		return 2
	}
	return report(root, stdout, stderr)
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}

// scanRoot resolves the one path argument to an absolute directory.
func scanRoot(args []string) (string, error) {
	if len(args) != 1 {
		return "", errors.New("expected at most one path")
	}
	if args[0] == "" {
		return "", errors.New("expected a path")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("no such directory %s", root)
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", root)
	}
	return root, nil
}
