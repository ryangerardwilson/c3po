package gmail

import (
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

func (e Env) dispatch(args []string) (int, error) {
	first := strings.ToLower(args[0])
	switch first {
	case "auth":
		return e.auth(args[1:])
	case "config":
		if len(args) != 1 {
			return 0, errs.Usage("Use: c3po gmail config")
		}
		return e.editConfig("")
	case "accounts":
		return e.accounts(args[1:])
	case "setup":
		return e.setup(args[1:])
	case "spam":
		if len(args) != 2 || args[1] != "clean" {
			return 0, errs.Usage("Use: c3po gmail spam clean")
		}
		return e.spamCleanAll()
	case "tidy":
		if len(args) != 1 {
			return 0, errs.Usage("Use: c3po gmail tidy")
		}
		return e.tidy()
	case "delete":
		if len(args) != 2 || args[1] != "unread" {
			return 0, errs.Usage("Use: c3po gmail delete unread")
		}
		return e.deleteUnreadAll()
	case "timer":
		return e.timer(args[1:])
	case "conf", "sc", "ti", "td", "st":
		return 0, errs.Usage("Use declarative commands: c3po gmail config, c3po gmail spam clean, or c3po gmail timer install|disable|status")
	}
	if !config.IsDigits(first) {
		return 0, errs.Usage("Expected: c3po gmail <preset> <command>. Use c3po gmail help for examples.")
	}
	if len(args) < 2 {
		return 0, errs.Usage("Expected: c3po gmail <preset> <command>. Use c3po gmail help for examples.")
	}
	return e.preset(first, args[1], args[2:])
}

func (e Env) preset(id, command string, params []string) (int, error) {
	app, err := e.load()
	if err != nil {
		return 0, err
	}
	acct, err := app.Account(id)
	if err != nil {
		return 0, err
	}
	command = strings.ToLower(command)
	if legacy[command] {
		return 0, errs.Usage("Use declarative commands. Run: c3po gmail help")
	}
	switch command {
	case "contacts":
		return e.contacts(app, acct, params)
	case "preview":
		return e.preview(acct, params)
	case "send":
		return e.send(acct, params)
	case "list":
		return e.list(app, acct, params)
	case "reply":
		return e.reply(acct, params)
	case "open":
		return e.open(acct, params, app.TimezoneOffset)
	case "inspect":
		return e.inspect(acct, params, app.TimezoneOffset)
	case "mark":
		return e.mark(acct, params)
	case "delete":
		return e.deletePreset(acct, params)
	case "spam":
		return e.spamPreset(app, acct, params)
	case "inspect-spam":
		if len(params) != 0 {
			return 0, errs.Usage("Use: c3po gmail <preset> spam inspect")
		}
		return e.spamInspect(app, acct)
	default:
		return 0, errs.Usage("Unknown command. Use: send, list, open, reply, mark, delete, spam, or contacts.")
	}
}

var legacy = map[string]bool{
	"s": true, "-s": true, "ls": true, "r": true, "o": true, "si": true,
	"sc": true, "sa": true, "se": true, "mr": true, "mra": true, "mur": true,
	"mstr": true, "mustr": true, "d": true, "ms": true, "cn": true,
}
