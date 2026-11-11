package usage

import (
	"path/filepath"
	"strings"
)

// Feature names the catalog feature for one invocation.
// argv[0] is the program name. Help and version stay help and version, so
// reading help does not count as using the feature the help describes.
// An unknown word is "unknown", or the app name when that name is itself a
// catalog feature. The typed word is never stored as a new feature.
func Feature(argv []string) string {
	if len(argv) == 0 {
		return "c3po"
	}
	args := argv[1:]
	if filepath.Base(argv[0]) == "rgw-ast" {
		return astFeature(args)
	}
	return c3poFeature(args)
}

func c3poFeature(args []string) string {
	if len(args) == 0 {
		return "c3po"
	}
	if args[0] == "help" || (helpWord(args[0]) && len(args) == 1) {
		return "help"
	}
	if topVersion(args[0]) {
		return "version"
	}
	if args[0] == "usage" {
		return "usage"
	}
	switch args[0] {
	case "ast":
		return astFeature(args[1:])
	case "tui":
		return tuiFeature(args[1:])
	case "deport":
		return deportFeature(args[1:])
	case "gcal":
		return gcalFeature(args[1:])
	case "gdrive":
		return gdriveFeature(args[1:])
	case "gmail":
		return gmailFeature(args[1:])
	case "looptab":
		return looptabFeature(args[1:])
	case "pass":
		return passFeature(args[1:])
	case "slack":
		return slackFeature(args[1:])
	case "tunnel":
		return tunnelFeature(args[1:])
	default:
		return "unknown"
	}
}

func astFeature(args []string) string {
	args, ok := dropRoot(args)
	if !ok {
		return "unknown"
	}
	args = dropExact(args, "--refresh")
	if len(args) == 0 {
		return "help"
	}
	cmd, rest := args[0], args[1:]
	if helpWord(cmd) {
		return "help"
	}
	if cmd == "version" || cmd == "--version" || cmd == "-v" {
		return "version"
	}
	if !astVerb[cmd] {
		return "unknown"
	}
	if cmd != "search" && len(rest) == 1 && (rest[0] == "--help" || rest[0] == "-h") {
		return "help"
	}
	return "ast " + cmd
}

func tuiFeature(args []string) string {
	if len(args) == 0 {
		return "tui"
	}
	if len(args) == 1 && helpWord(args[0]) {
		return "help"
	}
	if !tuiScreen[args[0]] {
		return "unknown"
	}
	rest := dropResume(args[1:])
	if len(rest) > 0 && helpWord(rest[0]) {
		return "help"
	}
	if len(rest) > 0 && (topVersion(rest[0]) || rest[0] == "-v") {
		return "version"
	}
	return "tui " + args[0]
}

func deportFeature(args []string) string {
	if len(args) == 1 && helpWord(args[0]) {
		return "help"
	}
	if len(args) == 1 && (args[0] == "version" || args[0] == "-v" || args[0] == "--version") {
		return "version"
	}
	if len(args) == 1 && args[0] == "list" {
		return "deport list"
	}
	return "deport"
}

func tunnelFeature(args []string) string {
	if len(args) == 0 || helpWord(args[0]) {
		return "help"
	}
	if args[0] == "version" {
		return "version"
	}
	switch args[0] {
	case "list", "add", "remove", "start", "stop", "status", "logs", "run":
		return "tunnel " + args[0]
	default:
		return "unknown"
	}
}

func looptabFeature(args []string) string {
	if len(args) == 0 {
		return "looptab"
	}
	switch args[0] {
	case "help", "-h", "--help":
		return "help"
	case "version", "--version", "-v":
		return "version"
	case "list", "register", "remove", "show", "edit":
		return "looptab " + args[0]
	default:
		return "looptab"
	}
}

func passFeature(args []string) string {
	if len(args) == 0 {
		return "help"
	}
	switch args[0] {
	case "help", "-h", "--help":
		return "help"
	case "version":
		return "version"
	case "upgrade":
		return "pass upgrade"
	}
	for _, arg := range args {
		switch arg {
		case "--create":
			return "pass create"
		case "--edit":
			return "pass edit"
		case "--show":
			return "pass show"
		case "--delete":
			return "pass delete"
		}
	}
	return "pass"
}

func gmailFeature(args []string) string {
	if len(args) == 0 {
		return "help"
	}
	if helpWord(args[0]) {
		return "help"
	}
	if args[0] == "version" || args[0] == "--version" || args[0] == "-v" {
		return "version"
	}
	if args[0] == "upgrade" {
		return "gmail"
	}
	first := strings.ToLower(args[0])
	switch first {
	case "auth":
		return "gmail auth"
	case "config":
		return "gmail config"
	case "accounts":
		return "gmail accounts"
	case "setup":
		return "gmail setup"
	case "spam":
		return "gmail spam clean"
	case "tidy":
		if len(args) >= 2 && strings.ToLower(args[1]) == "audit" {
			return "gmail tidy audit"
		}
		return "gmail tidy"
	case "delete":
		return "gmail delete unread"
	case "timer":
		return gmailTimer(args[1:])
	case "conf", "sc", "ti", "td", "st":
		return "gmail"
	}
	if !digits(first) || len(args) < 2 {
		return "gmail"
	}
	switch strings.ToLower(args[1]) {
	case "contacts", "preview", "send", "list", "reply", "open", "inspect", "mark", "delete", "spam":
		return "gmail " + strings.ToLower(args[1])
	case "inspect-spam":
		return "gmail inspect-spam"
	default:
		return "gmail"
	}
}

func gmailTimer(params []string) string {
	if len(params) >= 1 {
		switch strings.ToLower(params[0]) {
		case "install":
			return "gmail timer install"
		case "disable":
			return "gmail timer disable"
		case "status":
			return "gmail timer status"
		}
	}
	return "gmail timer"
}

func gcalFeature(args []string) string {
	if len(args) == 0 || (len(args) == 1 && helpWord(args[0])) {
		return "help"
	}
	if len(args) == 1 && args[0] == "version" {
		return "version"
	}
	if args[0] == "upgrade" {
		return "gcal"
	}
	if helpWord(args[0]) {
		return "help"
	}
	if args[0] == "version" {
		return "version"
	}
	switch args[0] {
	case "auth":
		return "gcal auth"
	case "accounts":
		return "gcal accounts list"
	case "setup":
		return "gcal setup check"
	case "config":
		return "gcal config"
	}
	if !digits(args[0]) || len(args) < 2 {
		return "gcal"
	}
	switch args[1] {
	case "preview", "list", "export", "delete", "reschedule", "create":
		return "gcal " + args[1]
	default:
		return "gcal"
	}
}

func gdriveFeature(args []string) string {
	if len(args) == 0 || (len(args) == 1 && helpWord(args[0])) {
		return "help"
	}
	if len(args) == 1 && (args[0] == "version" || args[0] == "-v" || args[0] == "--version") {
		return "version"
	}
	switch args[0] {
	case "auth":
		return "gdrive auth"
	case "config":
		return "gdrive config"
	case "sync":
		if len(args) >= 2 && args[1] == "run" {
			return "gdrive sync run"
		}
		if len(args) >= 2 && args[1] == "restore" {
			return "gdrive sync restore"
		}
		return "gdrive"
	case "timer":
		if len(args) >= 2 {
			switch args[1] {
			case "install", "disable", "status":
				return "gdrive timer " + args[1]
			}
		}
		return "gdrive"
	}
	if !digits(args[0]) {
		return "gdrive"
	}
	return gdrivePreset(args[1:])
}

func gdrivePreset(params []string) string {
	if len(params) == 0 {
		return "gdrive"
	}
	if params[0] == "register" {
		return "gdrive register"
	}
	if len(params) >= 2 && params[0] == "list" && params[1] == "registrations" {
		return "gdrive list registrations"
	}
	if len(params) >= 2 && params[0] == "remove" && params[1] == "registration" {
		return "gdrive remove registration"
	}
	if len(params) >= 2 && params[0] == "restore" && params[1] == "registrations" {
		return "gdrive restore registrations"
	}
	if len(params) >= 2 && params[0] == "restore" && params[1] == "registration" {
		return "gdrive restore registration"
	}
	return "gdrive"
}

func slackFeature(args []string) string {
	if len(args) == 0 {
		return "help"
	}
	if helpWord(args[0]) {
		return "help"
	}
	if args[0] == "version" || args[0] == "--version" || args[0] == "-v" {
		return "version"
	}
	switch args[0] {
	case "accounts":
		return "slack accounts list"
	case "setup":
		return "slack setup check"
	case "config":
		return "slack config"
	case "auth":
		return "slack auth"
	case "mark":
		return "slack mark"
	}
	if !digits(args[0]) || len(args) < 2 {
		return "slack"
	}
	switch args[1] {
	case "contacts", "users", "events", "preview", "inspect", "send", "reply", "delete", "edit", "files", "open", "list", "unread", "thread", "conversations", "mark":
		return "slack " + args[1]
	default:
		return "slack"
	}
}

func helpWord(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func topVersion(s string) bool {
	return s == "version" || s == "--version" || s == "-version"
}

func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func dropExact(args []string, name string) []string {
	var rest []string
	for _, arg := range args {
		if arg == name {
			continue
		}
		rest = append(rest, arg)
	}
	return rest
}

func dropRoot(args []string) ([]string, bool) {
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--root" {
			if i+1 >= len(args) {
				return nil, false
			}
			i++
			continue
		}
		if strings.HasPrefix(arg, "--root=") {
			continue
		}
		rest = append(rest, arg)
	}
	return rest, true
}

func dropResume(args []string) []string {
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--resume" {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(args[i], "--resume=") {
			continue
		}
		rest = append(rest, args[i])
	}
	return rest
}
