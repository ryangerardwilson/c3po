// Package config loads and updates the gmail config and resolves the
// config, data, and token paths used by the original gmail CLI.
package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/xdg"
)

// Paths resolves locations from the environment and the home directory.
type Paths struct {
	Getenv func(string) string
	Home   string
}

func (p Paths) env(key string) string {
	if p.Getenv == nil {
		return ""
	}
	return p.Getenv(key)
}

// Expand applies Python Path.expanduser to path.
func (p Paths) Expand(path string) string {
	return ExpandUser(path, p.Home)
}

// ExpandUser expands a leading ~ or ~user like Python's Path.expanduser.
func ExpandUser(path, home string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	name, rest, _ := strings.Cut(path[1:], "/")
	base := home
	if name != "" {
		u, err := user.Lookup(name)
		if err != nil {
			return path
		}
		base = u.HomeDir
	}
	if rest == "" && !strings.Contains(path, "/") {
		return base
	}
	return filepath.Join(base, rest)
}

// ConfigPath is $GMAIL_CLI_CONFIG, $XDG_CONFIG_HOME/c3po/gmail/config.json,
// or ~/.config/c3po/gmail/config.json.
func (p Paths) ConfigPath() string {
	if v := p.env("GMAIL_CLI_CONFIG"); v != "" {
		return p.Expand(v)
	}
	return filepath.Join(p.configDir(), "config.json")
}

func (p Paths) configRoot() string {
	if v := p.env("XDG_CONFIG_HOME"); v != "" {
		return p.Expand(v)
	}
	return filepath.Join(p.Home, ".config")
}

func (p Paths) configDir() string {
	base := p.configRoot()
	dest := xdg.App(base, "gmail")
	_ = xdg.Adopt(dest, filepath.Join(base, "gmail"))
	return dest
}

// DataHome is $XDG_DATA_HOME/c3po/gmail or ~/.local/share/c3po/gmail.
func (p Paths) DataHome() string {
	base := filepath.Join(p.Home, ".local", "share")
	if v := p.env("XDG_DATA_HOME"); v != "" {
		base = p.Expand(v)
	}
	dest := xdg.App(base, "gmail")
	_ = xdg.Adopt(dest, filepath.Join(base, "gmail"))
	return dest
}

// NormalizeEmail trims and lowercases an account email.
func NormalizeEmail(email string) string {
	return strings.ToLower(PyStrip(email))
}

// TokenFile is the authorized-user token path for email.
func (p Paths) TokenFile(email string) string {
	return filepath.Join(p.DataHome(), "tokens", NormalizeEmail(email)+".json")
}

// EnsureTokenDirs creates the data and token directories with mode 0700.
func (p Paths) EnsureTokenDirs() error {
	home := p.DataHome()
	tokens := filepath.Join(home, "tokens")
	if err := os.MkdirAll(tokens, 0o777); err != nil {
		return err
	}
	for _, dir := range []string{home, tokens} {
		_ = os.Chmod(dir, 0o700)
	}
	return nil
}

// DefaultSignaturePath is <config>/signatures/<email>.txt.
func (p Paths) DefaultSignaturePath(email string) string {
	return filepath.Join(p.configDir(), "signatures", email+".txt")
}

// PyStrip trims whitespace as Python's str.strip() does.
func PyStrip(s string) string {
	return strings.TrimFunc(s, IsPySpace)
}

// IsPySpace matches Python's str.isspace for one rune.
func IsPySpace(r rune) bool {
	switch {
	case r >= 0x09 && r <= 0x0d, r >= 0x1c && r <= 0x20:
		return true
	case r == 0x85 || r == 0xa0 || r == 0x1680:
		return true
	case r >= 0x2000 && r <= 0x200a:
		return true
	}
	return r == 0x2028 || r == 0x2029 || r == 0x202f || r == 0x205f || r == 0x3000
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DefaultPaths reads the process environment and the user home directory.
func DefaultPaths() Paths {
	home, _ := os.UserHomeDir()
	return Paths{Getenv: os.Getenv, Home: home}
}
