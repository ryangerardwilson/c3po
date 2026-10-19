package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
)

// Tunnel is the named connector identity. Credential is the cloudflared JSON.
type Tunnel struct {
	ID         string
	Zone       string
	Credential []byte
	APIToken   string
}

func Load(tokenFile string) (Tunnel, error) {
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return Tunnel{}, fmt.Errorf("read service account token: %w", err)
	}
	trimmed := strings.TrimSpace(string(token))
	if trimmed == "" {
		return Tunnel{}, fmt.Errorf("service account token is empty")
	}
	tunnelItem, err := opItem(trimmed, "CLOUDFLARE_TUNNEL_CREDENTIALS")
	if err != nil {
		return Tunnel{}, err
	}
	apiItem, err := opItem(trimmed, "CLOUDFLARE_API_TOKEN")
	if err != nil {
		return Tunnel{}, err
	}
	return Parse(tunnelItem, apiItem)
}

func opItem(accountToken, title string) ([]byte, error) {
	cmd := exec.Command("op", "item", "get", title, "--vault", "Root", "--format", "json")
	cmd.Env = append(os.Environ(), "OP_SERVICE_ACCOUNT_TOKEN="+accountToken)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read %s from 1Password", title)
	}
	return out, nil
}

func Parse(tunnelItem, apiItem []byte) (Tunnel, error) {
	tunnelFields, err := fields(tunnelItem)
	if err != nil {
		return Tunnel{}, fmt.Errorf("tunnel item: %w", err)
	}
	apiFields, err := fields(apiItem)
	if err != nil {
		return Tunnel{}, fmt.Errorf("api item: %w", err)
	}
	cred := strings.TrimSpace(tunnelFields["credential"])
	if cred == "" {
		return Tunnel{}, fmt.Errorf("tunnel item is missing credential")
	}
	if !json.Valid([]byte(cred)) {
		return Tunnel{}, fmt.Errorf("tunnel credential is not json")
	}
	id := strings.TrimSpace(tunnelFields["tunnel_id"])
	if id == "" {
		return Tunnel{}, fmt.Errorf("tunnel item is missing tunnel_id")
	}
	zone := strings.TrimSpace(tunnelFields["hostname"])
	if zone == "" {
		zone = routes.Zone
	}
	api := strings.TrimSpace(apiFields["credential"])
	if api == "" {
		return Tunnel{}, fmt.Errorf("api item is missing credential")
	}
	return Tunnel{ID: id, Zone: zone, Credential: []byte(cred), APIToken: api}, nil
}

func fields(raw []byte) (map[string]string, error) {
	var item struct {
		Fields []struct {
			Label string `json:"label"`
			ID    string `json:"id"`
			Value string `json:"value"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, field := range item.Fields {
		label := field.Label
		if label == "" {
			label = field.ID
		}
		if label != "" && field.Value != "" {
			out[label] = field.Value
		}
	}
	return out, nil
}
