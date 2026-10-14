package execgen

import (
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

// shellInterpreters require explicit allowlisting of the interpreter itself.
var shellInterpreters = map[string]bool{
	"bash": true, "sh": true, "zsh": true, "dash": true, "fish": true,
	"ksh": true, "csh": true, "tcsh": true,
	"bash.exe": true, "sh.exe": true,
}

// Allowed reports whether argv matches a trusted generator rule structurally.
// Patterns are whitespace-separated token prefixes matched from argv[0].
// Example pattern "npm exec -- openspec" matches
// ["npm","exec","--","openspec","archive",...].
func Allowed(cfg config.Config, argv []string) (bool, string) {
	if len(argv) == 0 {
		return false, ""
	}
	norm := normalizeArgv(argv)
	base0 := filepath.Base(norm[0])

	for _, pat := range cfg.Generators.Allow {
		tokens := tokenizePattern(pat)
		if len(tokens) == 0 {
			continue
		}
		if matchPrefix(norm, tokens) {
			// shell interpreters only when the pattern itself starts with that shell
			if shellInterpreters[base0] && !shellInterpreters[filepath.Base(tokens[0])] {
				continue
			}
			return true, strings.Join(tokens, " ")
		}
		// also allow when argv[0] is an absolute path to the same binary name
		if filepath.Base(tokens[0]) == base0 && matchPrefix(norm, tokens) {
			return true, strings.Join(tokens, " ")
		}
	}
	return false, ""
}

func tokenizePattern(pat string) []string {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return nil
	}
	fields := strings.Fields(pat)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, strings.ToLower(f))
	}
	return out
}

func normalizeArgv(argv []string) []string {
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = strings.ToLower(a)
	}
	// normalize argv[0] basename for matching while keeping full path for exec
	// Matching uses base of first token compared flexibly in matchPrefix.
	return out
}

func matchPrefix(argv, prefix []string) bool {
	if len(argv) < len(prefix) {
		return false
	}
	// first token: match full path or basename
	if !token0Equal(argv[0], prefix[0]) {
		return false
	}
	for i := 1; i < len(prefix); i++ {
		if argv[i] != prefix[i] {
			return false
		}
	}
	return true
}

func token0Equal(arg, want string) bool {
	if arg == want {
		return true
	}
	return filepath.Base(arg) == want || filepath.Base(arg) == filepath.Base(want)
}
