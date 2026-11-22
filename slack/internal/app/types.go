package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/slack/internal/anyval"
)

const (
	appName                    = "c3po/slack"
	legacyAppName              = "sluser"
	defaultListLimit           = 10
	defaultListHTTPTimeout     = 45 * time.Second
	eventSyncConversationLimit = 20
	conversationTypesDMs       = "im,mpim"
	conversationTypesMember    = "im,mpim,public_channel,private_channel"
	defaultUserTokenFile       = "~/.openclaw/credentials/slack-user-token"
	configBootstrapText        = "{\n  \"accounts\": {}\n}\n"
)

var (
	userIDRE         = regexp.MustCompile(`^[UW][A-Z0-9]+$`)
	conversationIDRE = regexp.MustCompile(`^[CDG][A-Z0-9]+$`)
	messageIDRE      = regexp.MustCompile(`^([CDG][A-Z0-9]+):([0-9]+\.[0-9]+)$`)
	relativeTimeRE   = regexp.MustCompile(`(?i)^(\d+)([hdwmy])$`)
	isoDateRE        = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	isoMonthRE       = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	namedMonthRE     = regexp.MustCompile(`(?i)^([A-Za-z]+)[ -]+(\d{4})$`)
)

type Config map[string]any
type Account map[string]any
type Contacts map[string]string

type Args struct {
	Command       string
	Preset        string
	Label         string
	Email         string
	Recipient     string
	Message       string
	FileID        string
	OutputPath    string
	Paths         []string
	OpenMode      bool
	ListLabel     string
	ListRegistry  bool
	ListFilter    string
	ListLimit     int
	ListFrom      string
	ListContains  string
	ListTimeLimit string
	ListIn        string
	Query         string
	AuthPreset    string
	AuthUserToken string
	AuthName      string
	AuthImport    bool
	AuthList      bool
	EventsAction  string
	OutputJSON    bool
	Verbose       bool
}

// Runtime runs c3po slack. Slack and Cache are the ports to the outside;
// the composition root fills them.
type Runtime struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Slack      Connector
	Cache      Cache
	Now        func() time.Time
	OpenEditor func(path string, bootstrap string) error
	// RunEditor runs the editor command on path, attached to the process
	// terminal, never to Stdout.
	RunEditor func(editor, path string) error
	// IsTTY decides whether config edit may run. Nil uses StdioIsTerminal.
	IsTTY func() bool
	// StdioIsTerminal reports whether the process stdin and stdout are real
	// terminals. Nil means they are not.
	StdioIsTerminal func() bool
}

type UsageError struct {
	Message string
}

func (e UsageError) Error() string {
	return e.Message
}

// NewRuntime writes to the process stdio. Slack, Cache, and the terminal
// checks are left for the caller.
func NewRuntime() *Runtime {
	return &Runtime{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Now: func() time.Time {
			return time.Now()
		},
	}
}

func getenv(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func expandPath(value string) string {
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if value == "~" {
				return home
			}
			if strings.HasPrefix(value, "~/") {
				return filepath.Join(home, value[2:])
			}
		}
	}
	return os.ExpandEnv(value)
}

func configPath(override string) string {
	if override != "" {
		return expandPath(override)
	}
	base := getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appName, "config.json")
}

func stateBaseDir() string {
	base := getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, appName)
}

func xdgBase(kind string) string {
	switch kind {
	case "config":
		base := getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}
		return base
	default:
		base := getenv("XDG_STATE_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".local", "state")
		}
		return base
	}
}

func migrateLegacyXDG() {
	if testing.Testing() {
		return
	}
	oldCfg := filepath.Join(xdgBase("config"), legacyAppName, "config.json")
	newCfg := filepath.Join(xdgBase("config"), appName, "config.json")
	if _, err := os.Stat(newCfg); err == nil {
		// already migrated
	} else if _, err := os.Stat(oldCfg); err == nil {
		_ = os.MkdirAll(filepath.Dir(newCfg), 0o700)
		if err := os.Rename(oldCfg, newCfg); err == nil {
			_ = os.Chmod(newCfg, 0o600)
		}
	}
	oldState := filepath.Join(xdgBase("state"), legacyAppName)
	newState := filepath.Join(xdgBase("state"), appName)
	if _, err := os.Stat(newState); err != nil {
		if info, err := os.Stat(oldState); err == nil && info.IsDir() {
			_ = os.Rename(oldState, newState)
		}
	}
}

func loadConfig(path string) (Config, error) {
	migrateLegacyXDG()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return Config{}, nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config JSON at %s: %w", path, err)
	}
	if cfg == nil {
		cfg = Config{}
	}
	return cfg, nil
}

func saveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func accounts(cfg Config) map[string]Account {
	raw, ok := cfg["accounts"].(map[string]any)
	if !ok {
		return map[string]Account{}
	}
	out := map[string]Account{}
	for key, value := range raw {
		if account, ok := value.(map[string]any); ok {
			out[key] = Account(account)
		}
	}
	return out
}

func ensureAccounts(cfg Config) map[string]any {
	raw, ok := cfg["accounts"].(map[string]any)
	if !ok {
		raw = map[string]any{}
		cfg["accounts"] = raw
	}
	return raw
}

func sortedPresetKeys(accts map[string]Account) []string {
	keys := make([]string, 0, len(accts))
	for key := range accts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		leftInt, leftErr := strconv.Atoi(keys[i])
		rightInt, rightErr := strconv.Atoi(keys[j])
		if leftErr == nil && rightErr == nil {
			return leftInt < rightInt
		}
		return keys[i] < keys[j]
	})
	return keys
}

func selectAccount(cfg Config, preset string) (string, Account, error) {
	accts := accounts(cfg)
	if len(accts) == 0 {
		if preset == "" {
			return "default", Account(cfg), nil
		}
		return "", nil, UsageError{Message: "No account presets configured. Run: c3po slack auth <preset> ..."}
	}
	if preset == "" {
		return "", nil, UsageError{Message: fmt.Sprintf("Missing preset. Use: c3po slack <preset> <command>. Available presets: %s", strings.Join(sortedPresetKeys(accts), ","))}
	}
	account, ok := accts[preset]
	if !ok {
		return "", nil, UsageError{Message: fmt.Sprintf("Unknown preset: %s. Available presets: %s", preset, strings.Join(sortedPresetKeys(accts), ","))}
	}
	return preset, account, nil
}

func contactsForAccount(cfg Config, account Account) Contacts {
	contacts := Contacts{}
	copyContacts := func(raw any) {
		switch typed := raw.(type) {
		case map[string]any:
			for key, value := range typed {
				if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
					contacts[key] = strings.TrimSpace(text)
				}
			}
		case map[string]string:
			for key, value := range typed {
				if strings.TrimSpace(value) != "" {
					contacts[key] = strings.TrimSpace(value)
				}
			}
		}
	}
	copyContacts(account["contacts"])
	if len(accounts(cfg)) == 0 {
		copyContacts(cfg["contacts"])
	}
	return contacts
}

func saveContact(cfg Config, preset, label, target string) error {
	accts := ensureAccounts(cfg)
	raw, _ := accts[preset].(map[string]any)
	if raw == nil {
		raw = map[string]any{}
		accts[preset] = raw
	}
	contacts, _ := raw["contacts"].(map[string]any)
	if contacts == nil {
		contacts = map[string]any{}
		raw["contacts"] = contacts
	}
	contacts[label] = target
	return nil
}

func asMap(value any) map[string]any {
	return anyval.Map(value)
}

func asList(value any) []any {
	return anyval.List(value)
}

func str(value any) string {
	return anyval.Str(value)
}

func intValue(value any, fallback int) int {
	return anyval.Int(value, fallback)
}

func boolValue(value any) bool {
	return anyval.Bool(value)
}

func firstNonEmpty(values ...string) string {
	return anyval.FirstNonEmpty(values...)
}

func tsFloat(value string) float64 {
	return anyval.TSFloat(value)
}

func maxFloat(values ...float64) float64 {
	return anyval.MaxFloat(values...)
}

func maxInt(a, b int) int {
	return anyval.MaxInt(a, b)
}
