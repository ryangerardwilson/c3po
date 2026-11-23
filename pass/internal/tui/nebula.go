package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Nebula reaches 1Password through nebula crypt.
type Nebula struct{}

type vaultLine struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type itemLine struct {
	ID       string `json:"id"`
	VaultID  string `json:"vault_id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Username string `json:"username"`
}

// runCrypt runs nebula crypt until ctx ends. Tests replace it.
var runCrypt = func(ctx context.Context, args ...string) ([]byte, []byte, error) {
	path, err := exec.LookPath("nebula")
	if err != nil {
		return nil, nil, errors.New("nebula is not installed")
	}
	cmd := exec.CommandContext(ctx, path, append([]string{"crypt"}, args...)...)
	cmd.WaitDelay = time.Second
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, errb.Bytes(), errors.New("nebula crypt timed out")
		}
		message := strings.TrimSpace(errb.String())
		if message == "" {
			message = "nebula crypt failed"
		}
		return nil, errb.Bytes(), errors.New(message)
	}
	return out.Bytes(), errb.Bytes(), nil
}

// Vaults lists vaults.
func (Nebula) Vaults(ctx context.Context) ([]Vault, error) {
	out, _, err := runCrypt(ctx, "vaults")
	if err != nil {
		return nil, err
	}
	var vaults []Vault
	if err := eachJSON(out, func(line []byte) error {
		var item vaultLine
		if err := json.Unmarshal(line, &item); err != nil {
			return err
		}
		vaults = append(vaults, Vault{ID: item.ID, Name: item.Name})
		return nil
	}); err != nil {
		return nil, errors.New("unable to read the vault list")
	}
	return vaults, nil
}

// Items lists items in one vault. A concealed value is not in the output.
func (Nebula) Items(ctx context.Context, vault string) ([]Item, error) {
	out, _, err := runCrypt(ctx, "items", vault)
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := eachJSON(out, func(line []byte) error {
		var item itemLine
		if err := json.Unmarshal(line, &item); err != nil {
			return err
		}
		items = append(items, Item{ID: item.ID, VaultID: item.VaultID, Title: item.Title, Category: item.Category, Username: item.Username})
		return nil
	}); err != nil {
		return nil, errors.New("unable to read the item list")
	}
	return items, nil
}

// Copy asks nebula crypt to copy one item. The secret is not returned.
func (Nebula) Copy(ctx context.Context, vault, id string) error {
	_, _, err := runCrypt(ctx, "copy", vault, id)
	return err
}

// Delete asks nebula crypt to delete one item id.
func (Nebula) Delete(ctx context.Context, vault, id string) error {
	_, _, err := runCrypt(ctx, "delete", vault, "item", id)
	return err
}

func eachJSON(raw []byte, visit func([]byte) error) error {
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if err := visit(line); err != nil {
			return err
		}
	}
	return nil
}
