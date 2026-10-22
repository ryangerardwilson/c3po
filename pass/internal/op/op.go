// Package op runs the 1Password CLI for c3po pass.
package op

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Vault is one entry of op vault list --format=json.
type Vault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Item is one entry of op item list --long --format=json.
type Item struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Vault    struct {
		ID string `json:"id"`
	} `json:"vault"`
	AdditionalInformation string  `json:"additional_information"`
	Fields                []Field `json:"fields"`
}

// Field is one field of a listed item.
type Field struct {
	Label string `json:"label"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Runner starts the op binary. Tests replace it with a fake.
type Runner interface {
	// LookPath reports whether op can be started.
	LookPath() error
	// Run starts op with args and returns its stdout. stdin is nil when
	// op reads nothing.
	Run(env []string, stdin []byte, stderr io.Writer, args ...string) ([]byte, error)
}

// Exec starts the op binary found on PATH.
type Exec struct{}

// LookPath reports whether op is on PATH.
func (Exec) LookPath() error {
	_, err := exec.LookPath("op")
	return err
}

// Run starts op with env, sends stdin, and returns stdout.
func (Exec) Run(env []string, stdin []byte, stderr io.Writer, args ...string) ([]byte, error) {
	cmd := exec.Command("op", args...)
	cmd.Env = env
	cmd.Stderr = stderr
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	return cmd.Output()
}

// Client runs op with the service account environment.
type Client struct {
	Runner Runner
	Env    []string
	Err    io.Writer
}

// List returns the items in vault.
func (c Client) List(vault string) ([]Item, error) {
	out, err := c.Runner.Run(c.Env, nil, c.Err, "item", "list", "--vault", vault, "--long", "--format=json")
	if err != nil {
		return nil, fmt.Errorf("Unable to list 1Password items in vault %s.", vault)
	}
	var items []Item
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, errors.New("Unable to process the 1Password item list.")
	}
	return items, nil
}

// Vaults returns the vaults this account can see.
func (c Client) Vaults() ([]Vault, error) {
	out, err := c.Runner.Run(c.Env, nil, c.Err, "vault", "list", "--format=json")
	if err != nil {
		return nil, errors.New("Unable to list 1Password vaults.")
	}
	var vaults []Vault
	if err := json.Unmarshal(out, &vaults); err != nil {
		return nil, errors.New("Unable to process the 1Password vault list.")
	}
	return vaults, nil
}

// Read returns the password, or the credential of an API credential.
func (c Client) Read(vaultID, id, category string) (string, error) {
	if vaultID == "" || id == "" {
		return "", errors.New("1Password did not provide the IDs needed to read the secret.")
	}
	field := "password"
	if strings.EqualFold(category, "API_CREDENTIAL") {
		field = "credential"
	}
	ref := fmt.Sprintf("op://%s/%s/%s", vaultID, id, field)
	secret, err := c.Runner.Run(c.Env, nil, c.Err, "read", "--no-newline", ref)
	if err != nil || len(bytes.TrimSpace(secret)) == 0 {
		return "", errors.New("Match found, but no password is available.")
	}
	return string(secret), nil
}

// Get returns the full item JSON.
func (c Client) Get(vault, id string) (map[string]any, error) {
	out, err := c.Runner.Run(c.Env, nil, c.Err, "item", "get", id, "--vault", vault, "--format", "json")
	if err != nil {
		return nil, errors.New("Unable to read the 1Password item.")
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil || doc == nil {
		return nil, errors.New("Unable to read the 1Password item.")
	}
	return doc, nil
}

// Do runs op with stdin and discards its stdout.
func (c Client) Do(stdin []byte, args ...string) error {
	_, err := c.Runner.Run(c.Env, stdin, c.Err, args...)
	return err
}
