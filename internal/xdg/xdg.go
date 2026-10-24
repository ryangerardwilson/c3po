// Package xdg places each c3po app under the XDG config, data, and cache
// directories and renames a legacy path into place on first use.
package xdg

import (
	"os"
	"path/filepath"
	"testing"
)

// App is base/c3po/name.
func App(base, name string) string {
	return filepath.Join(base, "c3po", name)
}

// ConfigBase is XDG_CONFIG_HOME when home is this process's home, and
// home/.config otherwise. A temporary home passed by tests stays inside itself.
func ConfigBase(home string) string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && (home == "" || home == os.Getenv("HOME")) {
		return xdg
	}
	return filepath.Join(home, ".config")
}

// Adopt renames legacy onto dest when dest is absent. It does nothing under
// go test, so a test cannot rename the operator's live config.
func Adopt(dest, legacy string) error {
	if testing.Testing() {
		return nil
	}
	return move(dest, legacy)
}

func move(dest, legacy string) error {
	if dest == "" || legacy == "" || dest == legacy {
		return nil
	}
	there, err := present(dest)
	if err != nil || there {
		return err
	}
	there, err = present(legacy)
	if err != nil || !there {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	return os.Rename(legacy, dest)
}

func present(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// PassToken is the 1Password service-account token file.
// OP_SERVICE_ACCOUNT_TOKEN_FILE wins after the default file is adopted, so an
// override path is not renamed.
func PassToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := ConfigBase(home)
	dest := filepath.Join(App(base, "pass"), "service-account-token")
	legacy := filepath.Join(base, "op", "service-account-token")
	if err = Adopt(dest, legacy); err != nil {
		return "", err
	}
	if path := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN_FILE"); path != "" {
		return path, nil
	}
	return dest, nil
}

// Migrate moves every known legacy directory into the c3po XDG layout.
func Migrate() error {
	if testing.Testing() {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configBase := ConfigBase(home)
	dataBase := envOr(os.Getenv("XDG_DATA_HOME"), filepath.Join(home, ".local", "share"))
	cacheBase := envOr(os.Getenv("XDG_CACHE_HOME"), filepath.Join(home, ".cache"))
	stateBase := envOr(os.Getenv("XDG_STATE_HOME"), filepath.Join(home, ".local", "state"))
	pairs := legacyPairs(configBase, dataBase, cacheBase, stateBase)
	for _, pair := range pairs {
		if err = move(pair[0], pair[1]); err != nil {
			return err
		}
	}
	return nil
}

func legacyPairs(configBase, dataBase, cacheBase, stateBase string) [][2]string {
	return [][2]string{
		{App(configBase, "ast"), filepath.Join(configBase, "rgw-ast")},
		{App(configBase, "gcal"), filepath.Join(configBase, "gcal")},
		{App(configBase, "gdrive"), filepath.Join(configBase, "gdrive")},
		{App(configBase, "gmail"), filepath.Join(configBase, "gmail")},
		{App(configBase, "o"), filepath.Join(configBase, "o")},
		{filepath.Join(App(configBase, "pass"), "service-account-token"), filepath.Join(configBase, "op", "service-account-token")},
		{App(configBase, "tunnel"), filepath.Join(configBase, "tunnel")},
		{App(configBase, "slack"), filepath.Join(configBase, "sluser")},
		{App(dataBase, "gcal"), filepath.Join(dataBase, "gcal")},
		{App(dataBase, "gdrive"), filepath.Join(dataBase, "gdrive")},
		{App(dataBase, "gmail"), filepath.Join(dataBase, "gmail")},
		{App(cacheBase, "ast"), filepath.Join(cacheBase, "rgw-ast")},
		{filepath.Join(App(cacheBase, "o"), "picker-selection.txt"), filepath.Join(cacheBase, "o", "picker-selection.txt")},
		{App(stateBase, "slack"), filepath.Join(stateBase, "sluser")},
	}
}

func envOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
