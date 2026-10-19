// Package paths resolves the gcal config and data directories.
//
// Config lives at $XDG_CONFIG_HOME/c3po/gcal or ~/.config/c3po/gcal.
// Tokens and transcripts live under $XDG_DATA_HOME/c3po/gcal or
// ~/.local/share/c3po/gcal.
package paths

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/xdg"
)

// Dirs is the resolved set of gcal directories.
type Dirs struct {
	Config string
	Data   string
	home   string
}

// Resolve reads XDG_CONFIG_HOME, XDG_DATA_HOME, and HOME through getenv.
func Resolve(getenv func(string) string) Dirs {
	home := getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	d := Dirs{home: home}
	d.Config = appDir(getenv("XDG_CONFIG_HOME"), filepath.Join(home, ".config"), home, "gcal")
	d.Data = appDir(getenv("XDG_DATA_HOME"), filepath.Join(home, ".local", "share"), home, "gcal")
	return d
}

func appDir(env, fallback, home, name string) string {
	base := fallback
	if env != "" {
		base = expand(env, home)
	}
	dest := xdg.App(base, name)
	_ = xdg.Adopt(dest, filepath.Join(base, name))
	return dest
}

// ConfigFile is the gcal config.json path.
func (d Dirs) ConfigFile() string { return filepath.Join(d.Config, "config.json") }

// Tokens is the directory of per-email token files.
func (d Dirs) Tokens() string { return filepath.Join(d.Data, "tokens") }

// Transcripts is the directory exported transcripts are saved in.
func (d Dirs) Transcripts() string { return filepath.Join(d.Data, "transcripts") }

// TokenFile is the token path for one account email.
func (d Dirs) TokenFile(email string) string {
	return filepath.Join(d.Tokens(), NormalizeEmail(email)+".json")
}

// NormalizeEmail trims and lowercases an account email.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Ensure creates every gcal directory with mode 0700.
func (d Dirs) Ensure() error {
	for _, dir := range []string{d.Config, d.Data, d.Tokens(), d.Transcripts()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		_ = os.Chmod(dir, 0o700)
	}
	return nil
}

// Expand replaces a leading ~ or ~user like Python's expanduser.
func (d Dirs) Expand(p string) string { return expand(p, d.home) }

func expand(p, home string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	name, rest, _ := strings.Cut(p[1:], "/")
	base := home
	if name != "" {
		u, err := user.Lookup(name)
		if err != nil {
			return p
		}
		base = u.HomeDir
	}
	if rest == "" && !strings.Contains(p, "/") {
		return base
	}
	return strings.TrimSuffix(base, "/") + "/" + rest
}
