package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
	"github.com/ryangerardwilson/c3po/gdrive/internal/tui"
)

func presetClient(preset string) (driveapi.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	account, err := config.GetAccount(cfg, preset)
	if err != nil {
		return nil, err
	}
	if _, err := config.RequireClientSecret(account); err != nil {
		return nil, err
	}
	return newDriveClient(context.Background(), account)
}

// accountClient checks the account setup and returns its Drive client and
// backup root name.
func accountClient(account *config.Account) (driveapi.Client, string, error) {
	if _, err := config.RequireClientSecret(account); err != nil {
		return nil, "", err
	}
	root, err := config.RequireBackupRootName(account)
	if err != nil {
		return nil, "", err
	}
	client, err := newDriveClient(context.Background(), account)
	if err != nil {
		return nil, "", err
	}
	return client, root, nil
}

func runBrowse(preset string, _ []string, _ io.Writer) error {
	if _, err := ensureClientSecret(preset, true, os.Stdout); err != nil {
		return err
	}
	client, err := presetClient(preset)
	if err != nil {
		return err
	}
	_, err = tui.Browse(client)
	return err
}

// runSyncAll syncs every registration of every preset, in preset order.
// It is what the hourly timer runs.
func runSyncAll(stdout io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	didWork := false
	for _, preset := range sortedAccountKeys(cfg.Accounts) {
		account := cfg.Accounts[preset]
		if len(account.Registrations) == 0 {
			continue
		}
		if err := syncAccount(preset, account, stdout); err != nil {
			return err
		}
		didWork = true
	}
	if !didWork {
		return fmt.Errorf("no registrations")
	}
	return nil
}

func syncAccount(preset string, account *config.Account, stdout io.Writer) error {
	client, root, err := accountClient(account)
	if err != nil {
		return err
	}
	for idx := range account.Registrations {
		reg := account.Registrations[idx]
		summary, err := syncer.SyncRegistration(preset, &reg, client, root)
		if err != nil {
			return err
		}
		if err := config.UpdateRegistration(preset, reg); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "%s:%s\tcreated=%d\tupdated=%d\tmoved=%d\tdeleted=%d\n", preset, reg.ID, summary.Created, summary.Updated, summary.Moved, summary.Deleted)
	}
	return nil
}

func runRestoreAll(stdout io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	didWork := false
	for _, preset := range sortedAccountKeys(cfg.Accounts) {
		if len(cfg.Accounts[preset].Registrations) == 0 {
			continue
		}
		worked, err := restoreAccountRegistrations(preset, "", stdout)
		if err != nil {
			return err
		}
		didWork = didWork || worked
	}
	if !didWork {
		return fmt.Errorf("no registrations")
	}
	return nil
}

func runRestoreRegistrations(preset string, _ []string, stdout io.Writer) error {
	_, err := restoreAccountRegistrations(preset, "", stdout)
	return err
}

func runRestoreRegistration(preset string, params []string, stdout io.Writer) error {
	if len(params) != 3 {
		return fmt.Errorf("usage: %s <preset> restore registration <id>", CmdName)
	}
	_, err := restoreAccountRegistrations(preset, params[2], stdout)
	return err
}

func restoreAccountRegistrations(preset, registrationID string, stdout io.Writer) (bool, error) {
	cfg, err := config.Load()
	if err != nil {
		return false, err
	}
	account, err := config.GetAccount(cfg, preset)
	if err != nil {
		return false, err
	}
	regs, err := selectRegistrations(account, preset, registrationID)
	if err != nil || len(regs) == 0 {
		return false, err
	}
	client, root, err := accountClient(account)
	if err != nil {
		return false, err
	}
	return restoreRegistrations(preset, regs, client, root, stdout)
}

func selectRegistrations(account *config.Account, preset, registrationID string) ([]config.Registration, error) {
	if registrationID == "" {
		return account.Registrations, nil
	}
	reg, err := config.GetRegistration(preset, registrationID)
	if err != nil {
		return nil, err
	}
	return []config.Registration{reg}, nil
}

func restoreRegistrations(preset string, regs []config.Registration, client driveapi.Client, root string, stdout io.Writer) (bool, error) {
	didWork := false
	for _, reg := range regs {
		if !reg.Enabled {
			continue
		}
		summary, err := syncer.RestoreRegistrationFromRemote(preset, &reg, client, root)
		if err != nil {
			return false, err
		}
		if err := config.UpdateRegistration(preset, reg); err != nil {
			return false, err
		}
		didWork = true
		fmt.Fprintf(stdout, "%s:%s\tdownloaded=%d\tdirs_created=%d\tskipped_existing=%d\tstate_entries=%d\n", preset, reg.ID, summary.Downloaded, summary.DirsCreated, summary.SkippedExisting, summary.StateEntries)
	}
	return didWork, nil
}
