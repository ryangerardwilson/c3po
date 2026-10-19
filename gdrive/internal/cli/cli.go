// Package cli dispatches the gdrive commands after the global
// help and version actions.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

// CmdName is how the app is invoked. Usage errors name it.
const CmdName = "c3po gdrive"

// stdin is the prompt input. Tests replace it.
var stdin io.Reader = os.Stdin

// newDriveClient builds the Drive client for an account. Tests replace it
// so no command reaches the network.
var newDriveClient = func(ctx context.Context, account *config.Account) (driveapi.Client, error) {
	client, err := driveapi.New(ctx, account)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// Dispatch runs one non-global command. Every error is a usage or runtime
// failure that the caller prints and turns into exit status 2.
func Dispatch(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing command")
	}
	command := args[0]
	params := args[1:]
	switch {
	case command == "auth":
		if len(params) != 1 {
			return fmt.Errorf("usage: %s auth <client_secret_path>", CmdName)
		}
		return authAccount(params[0], stdout, stderr)
	case command == "config":
		if len(params) != 0 {
			return fmt.Errorf("usage: %s config", CmdName)
		}
		return openConfig()
	case command == "sync":
		return dispatchSync(params, stdout)
	case command == "timer":
		return dispatchTimer(params, stdout)
	case isNumeric(command):
		return dispatchPreset(command, params, stdout)
	default:
		return fmt.Errorf("unknown command `%s`", command)
	}
}

func dispatchSync(params []string, stdout io.Writer) error {
	if sameArgs(params, "run") {
		return runSyncAll(stdout)
	}
	if sameArgs(params, "restore") {
		return runRestoreAll(stdout)
	}
	return fmt.Errorf("usage: %s sync run | %s sync restore", CmdName, CmdName)
}

func dispatchTimer(params []string, stdout io.Writer) error {
	usage := fmt.Errorf("usage: %s timer install|disable|status", CmdName)
	if len(params) != 1 {
		return usage
	}
	switch params[0] {
	case "install":
		return installTimer(stdout)
	case "disable":
		return disableTimer(stdout)
	case "status":
		return timerStatus(stdout)
	default:
		return usage
	}
}

type presetCommand struct {
	match func(params []string) bool
	run   func(preset string, params []string, stdout io.Writer) error
}

// presetCommands are tried in order; the first match runs.
var presetCommands = []presetCommand{
	{func(p []string) bool { return p[0] == "register" }, runRegister},
	{func(p []string) bool { return sameArgs(p, "list", "registrations") }, runListRegistrations},
	{func(p []string) bool { return hasPrefix(p, "remove", "registration") }, runRemoveRegistration},
	{func(p []string) bool { return sameArgs(p, "browse") }, runBrowse},
	{func(p []string) bool { return p[0] == "upload" }, runUploadPicker},
	{func(p []string) bool { return sameArgs(p, "restore", "registrations") }, runRestoreRegistrations},
	{func(p []string) bool { return hasPrefix(p, "restore", "registration") }, runRestoreRegistration},
}

func dispatchPreset(preset string, params []string, stdout io.Writer) error {
	if len(params) == 0 {
		return fmt.Errorf("missing command: use `%s %s list registrations`", CmdName, preset)
	}
	for _, command := range presetCommands {
		if command.match(params) {
			return command.run(preset, params, stdout)
		}
	}
	return fmt.Errorf("unknown command `%s`", params[0])
}

func sameArgs(args []string, want ...string) bool {
	if len(args) != len(want) {
		return false
	}
	for idx := range args {
		if args[idx] != want[idx] {
			return false
		}
	}
	return true
}

func hasPrefix(args []string, want ...string) bool {
	return len(args) >= len(want) && sameArgs(args[:len(want)], want...)
}

func isNumeric(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

func sortedAccountKeys(accounts map[string]*config.Account) []string {
	keys := make([]string, 0, len(accounts))
	for key := range accounts {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		left, leftErr := strconv.Atoi(keys[i])
		right, rightErr := strconv.Atoi(keys[j])
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return keys[i] < keys[j]
	})
	return keys
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
