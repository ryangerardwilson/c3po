// Package slackapi is the Slack Web API over HTTP behind the core's Gateway
// port. Requests, replies, retries, and uploads stop here; the core sees
// Slack's objects and *app.APIError.
package slackapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/slack/internal/anyval"
	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

// DefaultBaseURL is the Slack Web API.
const DefaultBaseURL = "https://slack.com/api/"

const maxRateLimitRetries = 3

// Connector opens a Client per token over one HTTP client.
type Connector struct {
	HTTPClient *http.Client
	// BaseURL ends in a slash. Empty is DefaultBaseURL.
	BaseURL string
	// Sleep is used for rate-limit backoff; nil means time.Sleep.
	Sleep func(time.Duration)
}

var _ app.Connector = Connector{}

// Connect returns a Client for opts.Token. A timeout keeps the HTTP client's
// transport and replaces its bound.
func (c Connector) Connect(opts app.ConnectOptions) app.Gateway {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if opts.Timeout > 0 {
		httpClient = &http.Client{Timeout: opts.Timeout, Transport: httpClient.Transport}
	}
	return Client{Token: opts.Token, HTTPClient: httpClient, BaseURL: c.BaseURL, Verbose: opts.Verbose, Sleep: c.Sleep}
}

// Client calls the Slack Web API with one user token.
type Client struct {
	Token      string
	HTTPClient *http.Client
	BaseURL    string
	// Verbose receives progress lines (pagination, rate limits). Typically stderr.
	Verbose io.Writer
	// Sleep is used for rate-limit backoff; nil means time.Sleep.
	Sleep func(time.Duration)
}

var _ app.Gateway = Client{}

func (c Client) logf(format string, args ...any) {
	if c.Verbose == nil {
		return
	}
	fmt.Fprintf(c.Verbose, "slack: "+format+"\n", args...)
}

func (c Client) sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	if c.Sleep != nil {
		c.Sleep(d)
		return
	}
	time.Sleep(d)
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient == nil {
		return http.DefaultClient
	}
	return c.HTTPClient
}

func (c Client) baseURL() string {
	if c.BaseURL == "" {
		return DefaultBaseURL
	}
	return c.BaseURL
}

func shortCursor(cursor string) string {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return "-"
	}
	if len(cursor) <= 12 {
		return cursor
	}
	return cursor[:6] + "…" + cursor[len(cursor)-4:]
}

func retryAfterDuration(resp *http.Response, decoded map[string]any) time.Duration {
	if resp != nil {
		if raw := strings.TrimSpace(resp.Header.Get("Retry-After")); raw != "" {
			if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
			if when, err := http.ParseTime(raw); err == nil {
				if wait := time.Until(when); wait > 0 {
					return wait
				}
			}
		}
	}
	if decoded != nil {
		if seconds := anyval.Int(decoded["retry_after"], 0); seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return 2 * time.Second
}

// encodeRequest builds the URL, body, and content type for one call. GET
// sends the payload as a query, useForm as a form, and otherwise as JSON.
func (c Client) encodeRequest(method string, payload map[string]string, useForm bool, httpMethod string) (string, []byte, string, error) {
	reqURL := c.baseURL() + method
	contentType := "application/json; charset=utf-8"
	if strings.EqualFold(httpMethod, http.MethodGet) {
		values := url.Values{}
		for key, value := range payload {
			values.Set(key, value)
		}
		if encoded := values.Encode(); encoded != "" {
			reqURL += "?" + encoded
		}
		return reqURL, nil, contentType, nil
	}
	if useForm {
		values := url.Values{}
		for key, value := range payload {
			values.Set(key, value)
		}
		return reqURL, []byte(values.Encode()), "application/x-www-form-urlencoded", nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", nil, "", err
	}
	return reqURL, data, contentType, nil
}

// do sends one attempt and decodes Slack's reply.
func (c Client) do(method string, attempt int, httpMethod, reqURL, contentType string, bodyBytes []byte) (*http.Response, map[string]any, error) {
	var body io.Reader
	if bodyBytes != nil {
		body = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequest(httpMethod, reqURL, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	c.logf("api method=%s attempt=%d", method, attempt+1)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, nil, fmt.Errorf("Slack API %s returned invalid JSON: %w", method, err)
	}
	return resp, decoded, nil
}

// Request calls one Web API method, retrying rate limits. A reply that is
// not ok returns the reply and an *app.APIError.
func (c Client) Request(method string, payload map[string]string, useForm bool, httpMethod string) (map[string]any, error) {
	if httpMethod == "" {
		httpMethod = http.MethodPost
	}
	reqURL, bodyBytes, contentType, err := c.encodeRequest(method, payload, useForm, httpMethod)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt <= maxRateLimitRetries; attempt++ {
		resp, decoded, err := c.do(method, attempt, httpMethod, reqURL, contentType, bodyBytes)
		if err != nil {
			return nil, err
		}
		rateLimited := resp.StatusCode == http.StatusTooManyRequests || anyval.Str(decoded["error"]) == "ratelimited"
		if rateLimited {
			wait := retryAfterDuration(resp, decoded)
			if attempt >= maxRateLimitRetries {
				c.logf("rate_limited method=%s giving_up after=%d wait_would_be=%s", method, attempt+1, wait)
				return decoded, fmt.Errorf("Slack API %s rate limited after %d retries", method, attempt+1)
			}
			c.logf("rate_limited method=%s attempt=%d retry_after=%s", method, attempt+1, wait)
			c.sleep(wait)
			continue
		}
		if decoded["ok"] != true {
			return decoded, &app.APIError{Method: method, Code: anyval.FirstNonEmpty(anyval.Str(decoded["error"]), "unknown_error")}
		}
		return decoded, nil
	}
	return nil, fmt.Errorf("Slack API %s failed: rate_limited", method)
}

func (c Client) get(method string, payload map[string]string) (map[string]any, error) {
	return c.Request(method, payload, false, http.MethodGet)
}

func (c Client) postJSON(method string, payload map[string]string) (map[string]any, error) {
	return c.Request(method, payload, false, http.MethodPost)
}

func (c Client) postForm(method string, payload map[string]string) (map[string]any, error) {
	return c.Request(method, payload, true, http.MethodPost)
}

func listAPI(client Client, method string, params map[string]string, key string) ([]map[string]any, error) {
	var rows []map[string]any
	cursor := ""
	page := 0
	for {
		page++
		payload := map[string]string{}
		for k, v := range params {
			payload[k] = v
		}
		if cursor != "" {
			payload["cursor"] = cursor
		}
		client.logf("paginate method=%s page=%d cursor=%s rows_so_far=%d", method, page, shortCursor(cursor), len(rows))
		data, err := client.get(method, payload)
		if err != nil {
			return nil, err
		}
		before := len(rows)
		for _, raw := range anyval.List(data[key]) {
			if item := anyval.Map(raw); len(item) > 0 {
				rows = append(rows, item)
			}
		}
		cursor = strings.TrimSpace(anyval.Str(anyval.Map(data["response_metadata"])["next_cursor"]))
		client.logf("paginate method=%s page=%d fetched=%d total=%d next_cursor=%s", method, page, len(rows)-before, len(rows), shortCursor(cursor))
		if cursor == "" {
			break
		}
	}
	return rows, nil
}

// objects returns each item of a reply list as an object, keeping the count.
func objects(value any) []map[string]any {
	raw := anyval.List(value)
	if raw == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		out = append(out, anyval.Map(item))
	}
	return out
}
