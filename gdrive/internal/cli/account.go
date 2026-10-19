package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/auth"
	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/paths"
	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
)

func authAccount(clientSecretPath string, stdout, stderr io.Writer) error {
	path := paths.ExpandHome(clientSecretPath)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return fmt.Errorf("missing client secret file: %s", path)
	}
	root, err := prompt("Drive backup root dir name: ", stdout)
	if err != nil {
		return err
	}
	for strings.TrimSpace(root) == "" {
		fmt.Fprintln(stderr, "enter a folder name like `Backups` or `ComputerBackups`")
		root, err = prompt("Drive backup root dir name: ", stdout)
		if err != nil {
			return err
		}
	}
	_, email, err := auth.AuthorizeAccount(path, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	account, err := config.UpsertAuthenticatedAccount(path, email, root)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "authorized\t%s\t%s\t%s\n", account.Preset, account.Email, account.BackupRootName)
	return nil
}

func ensureClientSecret(preset string, interactive bool, stdout io.Writer) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	account, err := config.EnsureAccount(cfg, preset)
	if err != nil {
		return "", err
	}
	if account.ClientSecretFile != "" {
		return config.RequireClientSecret(account)
	}
	if !interactive {
		return "", fmt.Errorf("missing client secret in config for preset `%s`: run `%s auth <client_secret_path>` first", preset, CmdName)
	}
	value, err := prompt(fmt.Sprintf("Preset %s Google client secret file path: ", preset), stdout)
	if err != nil {
		return "", err
	}
	return config.SetClientSecret(preset, value)
}

func ensureBackupRootName(preset string, interactive bool, stdout io.Writer) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	account, err := config.EnsureAccount(cfg, preset)
	if err != nil {
		return "", err
	}
	if account.BackupRootName != "" {
		return config.RequireBackupRootName(account)
	}
	if !interactive {
		return "", fmt.Errorf("missing backup root in config for preset `%s`: run `%s %s list registrations` interactively first", preset, CmdName, preset)
	}
	value, err := prompt(fmt.Sprintf("Preset %s Drive backup root dir name: ", preset), stdout)
	if err != nil {
		return "", err
	}
	return config.SetBackupRootName(preset, value)
}

func ensureSetup(preset string, interactive bool, stdout io.Writer) error {
	if _, err := ensureClientSecret(preset, interactive, stdout); err != nil {
		return err
	}
	_, err := ensureBackupRootName(preset, interactive, stdout)
	return err
}

func runRegister(preset string, params []string, stdout io.Writer) error {
	if len(params) != 4 || params[2] != "as" {
		return fmt.Errorf("usage: %s <preset> register <local_dir> as <drive_path>", CmdName)
	}
	if err := ensureSetup(preset, true, stdout); err != nil {
		return err
	}
	reg, err := config.AddRegistration(preset, params[1], params[3])
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "registered\t%s\t%s\t%s\t%s\n", preset, reg.ID, reg.LocalDir, reg.DrivePath)
	return nil
}

func runListRegistrations(preset string, _ []string, stdout io.Writer) error {
	if err := ensureSetup(preset, true, stdout); err != nil {
		return err
	}
	return printRegistrations(preset, stdout)
}

func runRemoveRegistration(preset string, params []string, stdout io.Writer) error {
	if len(params) != 3 {
		return fmt.Errorf("usage: %s <preset> remove registration <id>", CmdName)
	}
	if err := ensureSetup(preset, true, stdout); err != nil {
		return err
	}
	reg, err := config.RemoveRegistration(preset, params[2])
	if err != nil {
		return err
	}
	_ = syncer.DeleteState(preset, reg.ID)
	fmt.Fprintf(stdout, "removed\t%s\t%s\t%s\n", preset, reg.ID, reg.LocalDir)
	return nil
}

func printRegistrations(preset string, stdout io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	account, err := config.GetAccount(cfg, preset)
	if err != nil {
		return err
	}
	root, err := config.RequireBackupRootName(account)
	if err != nil {
		return err
	}
	if len(account.Registrations) == 0 {
		fmt.Fprintln(stdout, "no registrations")
		return nil
	}
	for idx, reg := range account.Registrations {
		url := "-"
		if reg.RemoteRootID != "" {
			url = "https://drive.google.com/drive/folders/" + reg.RemoteRootID
		}
		prefix := fmt.Sprintf("[%d]", idx+1)
		fmt.Fprintln(stdout, prefix+strings.Repeat("-", max(1, 79-len(prefix))))
		fmt.Fprintf(stdout, "edit_id : %s\n", reg.ID)
		fmt.Fprintf(stdout, "local   : %s\n", reg.LocalDir)
		fmt.Fprintf(stdout, "drive   : %s/%s\n", root, reg.DrivePath)
		fmt.Fprintln(stdout, url)
	}
	return nil
}

func openConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		parts = []string{"vim"}
	}
	cmd := exec.Command(parts[0], append(parts[1:], cfg.Path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func prompt(label string, stdout io.Writer) (string, error) {
	fmt.Fprint(stdout, label)
	reader := bufio.NewReader(stdin)
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	return strings.TrimSpace(value), nil
}
