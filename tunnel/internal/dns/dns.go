package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const targetSuffix = ".cfargotunnel.com"

var ErrForeign = errors.New("dns points somewhere else")

type Record struct {
	Type    string
	Name    string
	Content string
}

// Decide reports whether a hostname needs a tunnel CNAME, already has one, or must be left alone.
func Decide(hostname, tunnelID string, records []Record) string {
	want := Target(tunnelID)
	if len(records) == 0 {
		return "create"
	}
	for _, rec := range records {
		if strings.EqualFold(rec.Type, "CNAME") && sameHost(rec.Content, want) {
			continue
		}
		return "refuse"
	}
	return "keep"
}

func Target(tunnelID string) string {
	return strings.ToLower(strings.TrimSpace(tunnelID)) + targetSuffix
}

func sameHost(got, want string) bool {
	got = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(got)), ".")
	want = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(want)), ".")
	return got == want
}

type Client struct {
	Token   string
	BaseURL string
	HTTP    *http.Client
	zoneID  string
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return "https://api.cloudflare.com/client/v4"
}

// Ensure creates a proxied CNAME when the name has no DNS yet.
// A record that already points somewhere else is left unchanged.
func (c *Client) Ensure(ctx context.Context, zoneName, hostname, tunnelID string) error {
	zoneID, err := c.lookupZone(ctx, zoneName)
	if err != nil {
		return err
	}
	records, err := c.records(ctx, zoneID, hostname)
	if err != nil {
		return err
	}
	switch Decide(hostname, tunnelID, records) {
	case "keep":
		return nil
	case "refuse":
		return fmt.Errorf("%s %w", hostname, ErrForeign)
	}
	if err := c.create(ctx, zoneID, hostname, Target(tunnelID)); err != nil {
		return c.recheck(ctx, zoneID, hostname, tunnelID, err)
	}
	return nil
}

// recheck reads the name again after a failed create. Another writer may
// have added the same CNAME, or a foreign record, in the meantime.
func (c *Client) recheck(ctx context.Context, zoneID, hostname, tunnelID string, createErr error) error {
	again, err := c.records(ctx, zoneID, hostname)
	if err != nil {
		return createErr
	}
	switch Decide(hostname, tunnelID, again) {
	case "keep":
		return nil
	case "refuse":
		return fmt.Errorf("%s %w", hostname, ErrForeign)
	}
	return createErr
}

func (c *Client) lookupZone(ctx context.Context, name string) (string, error) {
	if c.zoneID != "" {
		return c.zoneID, nil
	}
	var payload struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	query := url.Values{}
	query.Set("name", name)
	if err := c.do(ctx, http.MethodGet, "/zones?"+query.Encode(), nil, &payload); err != nil {
		return "", err
	}
	if !payload.Success || len(payload.Result) == 0 || payload.Result[0].ID == "" {
		return "", fmt.Errorf("zone %s was not found", name)
	}
	c.zoneID = payload.Result[0].ID
	return c.zoneID, nil
}

func (c *Client) records(ctx context.Context, zoneID, hostname string) ([]Record, error) {
	var payload struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Content string `json:"content"`
		} `json:"result"`
	}
	query := url.Values{}
	query.Set("name", hostname)
	path := "/zones/" + url.PathEscape(zoneID) + "/dns_records?" + query.Encode()
	if err := c.do(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return nil, err
	}
	if !payload.Success {
		return nil, apiError("list dns", payload.Errors)
	}
	out := make([]Record, 0, len(payload.Result))
	for _, rec := range payload.Result {
		if rec.Name != "" && !sameHost(rec.Name, hostname) {
			continue
		}
		out = append(out, Record{Type: rec.Type, Name: rec.Name, Content: rec.Content})
	}
	return out, nil
}

func (c *Client) create(ctx context.Context, zoneID, hostname, content string) error {
	body := map[string]any{
		"type":    "CNAME",
		"name":    hostname,
		"content": content,
		"proxied": true,
		"ttl":     1,
	}
	var payload struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	path := "/zones/" + url.PathEscape(zoneID) + "/dns_records"
	if err := c.do(ctx, http.MethodPost, path, body, &payload); err != nil {
		return err
	}
	if !payload.Success {
		return apiError("create dns", payload.Errors)
	}
	return nil
}

func apiError(action string, errs []struct {
	Message string `json:"message"`
}) error {
	if len(errs) == 0 || errs[0].Message == "" {
		return fmt.Errorf("%s failed", action)
	}
	return fmt.Errorf("%s: %s", action, errs[0].Message)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	reader, err := encode(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base()+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func encode(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(raw), nil
}

func decode(resp *http.Response, out any) error {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cloudflare http %d", resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("cloudflare response: %w", err)
	}
	return nil
}
