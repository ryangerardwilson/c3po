package paths

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/xdg"
)

func ConfigHome() string {
	return appHome("XDG_CONFIG_HOME", filepath.Join(homeDir(), ".config"), "gdrive")
}

func DataHome() string {
	return appHome("XDG_DATA_HOME", filepath.Join(homeDir(), ".local", "share"), "gdrive")
}

func appHome(key, fallback, name string) string {
	base := os.Getenv(key)
	if base == "" {
		base = fallback
	}
	base = expandHome(base)
	dest := xdg.App(base, name)
	_ = xdg.Adopt(dest, filepath.Join(base, name))
	return dest
}

func ConfigFile() string {
	if override := os.Getenv("GDRIVE_CONFIG"); override != "" {
		return expandHome(override)
	}
	return filepath.Join(ConfigHome(), "config.json")
}

func TokenDir() string {
	return filepath.Join(DataHome(), "tokens")
}

func TokenFileForEmail(email string) string {
	return filepath.Join(TokenDir(), strings.ToLower(strings.TrimSpace(email))+".json")
}

func StateDir() string {
	return filepath.Join(DataHome(), "state")
}

func EnsureDirs() error {
	for _, dir := range []string{ConfigHome(), DataHome(), TokenDir(), StateDir()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func ExpandHome(path string) string {
	return expandHome(path)
}

func expandHome(path string) string {
	if path == "~" {
		return homeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir(), strings.TrimPrefix(path, "~/"))
	}
	return path
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}
