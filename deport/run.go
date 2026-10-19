// Package deport frees a local port by killing whatever is bound to it.
package deport

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// Help is the full help text for c3po deport.
const Help = `c3po deport frees a local port by killing whatever is bound to it.

Usage:
  c3po deport <port>
  c3po deport help
  c3po deport version

<port> is a number from 1 to 65535. Every process with a TCP or UDP socket
on that local port gets SIGTERM. When the port is still taken a second
later, each of those processes that is still alive gets SIGKILL. Every TCP
state counts: a listener, an accepted connection, and a socket bound to
that port all hold it. A client socket whose remote port matches is left
alone. Threads are signalled once, through their process.

Each signalled process is printed as "killed <pid> <name>", then
"<port> is free". When nothing is bound, only "<port> is free" is printed.

  c3po deport 8082
  killed 4242 python
  8082 is free

help, -h, and --help print this text. version, -v, and --version print the
c3po version.

Exit 0 when the port is free, 1 when it stays occupied or its process
cannot be seen or signalled, 2 on bad usage.
`

const (
	termTimeout = time.Second
	killTimeout = 300 * time.Millisecond
	pollEvery   = 20 * time.Millisecond
)

// usageError is a bad invocation. The command exits 2.
type usageError string

func (e usageError) Error() string { return string(e) }

// env is what Run touches outside the process: the proc tree, signals,
// and the clock.
type env struct {
	procRoot string
	killer   killer
	sleep    func(time.Duration)
}

// Run frees the port named by args. args are the arguments after
// c3po deport.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, env{procRoot: "/proc", killer: osKiller{}, sleep: time.Sleep})
}

func run(args []string, stdout, stderr io.Writer, e env) int {
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	port, err := parseArgs(args)
	if err != nil {
		return fail(stderr, err)
	}
	killed, err := free(e.procRoot, port, e.killer, e.sleep)
	for _, p := range killed {
		fmt.Fprintf(stdout, "killed %d %s\n", p.pid, p.name)
	}
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "%d is free\n", port)
	return 0
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "deport: %s\n", err.Error())
	var usage usageError
	if errors.As(err, &usage) {
		fmt.Fprint(stderr, Help)
		return 2
	}
	return 1
}

func parseArgs(args []string) (int, error) {
	if len(args) != 1 {
		return 0, usageError("expected one port")
	}
	return parsePort(args[0])
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func isVersion(arg string) bool {
	return arg == "version" || arg == "-v" || arg == "--version"
}

func parsePort(arg string) (int, error) {
	port, err := strconv.Atoi(arg)
	if err != nil || port < 1 || port > 65535 {
		return 0, usageError(fmt.Sprintf("%q is not a port", arg))
	}
	return port, nil
}
