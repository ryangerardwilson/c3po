package op

import (
	"errors"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/xdg"
)

const tokenVar = "OP_SERVICE_ACCOUNT_TOKEN"

var errNoToken = errors.New("1Password service account token is missing.")

// Env returns the environment for op. OP_SERVICE_ACCOUNT_TOKEN is used
// when it is set. Otherwise the token is read from the file named by
// OP_SERVICE_ACCOUNT_TOKEN_FILE, or from
// ~/.config/c3po/pass/service-account-token.
func Env() ([]string, error) {
	if strings.TrimSpace(os.Getenv(tokenVar)) != "" {
		return os.Environ(), nil
	}
	path, err := TokenPath()
	if err != nil {
		return nil, errNoToken
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errNoToken
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return nil, errNoToken
	}
	return withToken(os.Environ(), token), nil
}

// TokenPath is the token file read when OP_SERVICE_ACCOUNT_TOKEN is empty.
func TokenPath() (string, error) {
	return xdg.PassToken()
}

func withToken(environ []string, token string) []string {
	env := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		if strings.HasPrefix(entry, tokenVar+"=") {
			continue
		}
		env = append(env, entry)
	}
	return append(env, tokenVar+"="+token)
}
