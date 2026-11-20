// Package notify sends a toast through omarchy-shell.
package notify

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// Help is the text for c3po notify and c3po help -f notify.
const Help = `c3po notify
send a toast through omarchy-shell (ryan.notify overlay)

  c3po notify heading "<title>" body "<message>"
    show a toast
  c3po notify --urgency critical heading "<title>" body "<message>"
    urgency is low, normal, or critical; -u is the short flag
  c3po notify <title> [body...]
    the legacy positional form
  c3po notify help
    show this help
  c3po notify version
    print the c3po version

examples:
  c3po notify heading "gdrive" body "backup finished"
  c3po notify --urgency critical heading "gdrive" body "backup failed"

Delivery tries omarchy-shell notify show, then the quickshell bar.
Success prints nothing. There is no notify screen.
`

type sender func(title, body, urgency string) error

// Run serves c3po notify. args are the words after notify.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, deliver)
}

func run(args []string, stdout, stderr io.Writer, send sender) int {
	if len(args) == 1 && versionWord(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	urgency, rest, help, err := parseFlags(args)
	if help {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if err != nil {
		return usageError(err, stderr)
	}
	title, body, err := parseMessage(rest)
	if err != nil {
		return usageError(err, stderr)
	}
	if err := send(title, body, urgency); err != nil {
		fmt.Fprintf(stderr, "notify: %s\n", err)
		return 1
	}
	return 0
}

func usageError(err error, stderr io.Writer) int {
	fmt.Fprintf(stderr, "notify: %s\n", err)
	fmt.Fprintln(stderr, `usage: c3po notify [--urgency low|normal|critical] heading "<title>" body "<message>"`)
	return 2
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}

// parseFlags reads the leading flags. help is true when a help word comes
// before the message.
func parseFlags(args []string) (urgency string, rest []string, help bool, err error) {
	urgency = "normal"
	if len(args) > 0 && helpWord(args[0]) {
		return urgency, nil, true, nil
	}
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "--urgency", "-u":
			if len(args) < 2 {
				return urgency, nil, false, errors.New("expected value after --urgency")
			}
			urgency = normalizeUrgency(args[1])
			args = args[2:]
		case "--help", "-h":
			return urgency, nil, true, nil
		default:
			return urgency, nil, false, fmt.Errorf("unknown flag %q", args[0])
		}
	}
	return urgency, args, false, nil
}

func parseMessage(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", errors.New(`expected heading "..." body "..."`)
	}
	heading, body, labeled, err := parseLabeledMessage(args)
	if err != nil {
		return "", "", err
	}
	if labeled {
		if strings.TrimSpace(heading) == "" {
			return "", "", errors.New("heading must not be empty")
		}
		return heading, body, nil
	}
	if strings.TrimSpace(args[0]) == "" {
		return "", "", errors.New("title must not be empty")
	}
	return args[0], strings.Join(args[1:], " "), nil
}

func parseLabeledMessage(args []string) (heading, body string, labeled bool, err error) {
	hasHeading := false
	for i := 0; i < len(args); i++ {
		switch strings.ToLower(args[i]) {
		case "heading":
			if i+1 >= len(args) {
				return "", "", false, errors.New("expected value after heading")
			}
			heading = args[i+1]
			hasHeading = true
			labeled = true
			i++
		case "body":
			if i+1 >= len(args) {
				return "", "", false, errors.New("expected value after body")
			}
			body = args[i+1]
			labeled = true
			i++
		default:
			if labeled {
				return "", "", false, fmt.Errorf("unexpected argument %q after labeled notify syntax", args[i])
			}
		}
	}
	if labeled && !hasHeading {
		return "", "", false, errors.New(`notify requires heading "..."`)
	}
	return heading, body, labeled, nil
}

func normalizeUrgency(value string) string {
	switch v := strings.ToLower(strings.TrimSpace(value)); v {
	case "low", "normal", "critical":
		return v
	default:
		return "normal"
	}
}
