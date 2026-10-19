package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

const defaultBase = "https://gmail.googleapis.com/gmail/v1/users/me"

// Client talks to the Gmail REST API.
type Client struct {
	HTTP   *http.Client
	Base   string
	Access func() (string, error)
}

func (c *Client) base() string {
	if c.Base != "" {
		return strings.TrimRight(c.Base, "/")
	}
	return defaultBase
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// ProfileEmail reads users.getProfile.
func (c *Client) ProfileEmail() (string, error) {
	var out struct {
		Email string `json:"emailAddress"`
	}
	if err := c.get("/profile", nil, &out); err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(out.Email)), nil
}

// ListIDs pages users.messages.list at 500 ids per page.
func (c *Client) ListIDs(query string) ([]string, error) {
	var ids []string
	page := ""
	for {
		q := url.Values{"maxResults": {"500"}}
		if query != "" {
			q.Set("q", query)
		}
		if page != "" {
			q.Set("pageToken", page)
		}
		var out struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
			Next string `json:"nextPageToken"`
		}
		if err := c.get("/messages", q, &out); err != nil {
			return nil, errs.API("Gmail list message ids failed for query '%s': %s", query, err.Error())
		}
		for _, item := range out.Messages {
			if item.ID != "" {
				ids = append(ids, item.ID)
			}
		}
		if out.Next == "" {
			return ids, nil
		}
		page = out.Next
	}
}

// List returns up to limit full messages. limit <= 0 lists every match.
func (c *Client) List(query string, limit int) ([]Message, error) {
	ids, err := c.listLimited(query, limit)
	if err != nil {
		return nil, err
	}
	return c.hydrate(ids)
}

func (c *Client) listLimited(query string, limit int) ([]string, error) {
	if limit <= 0 {
		return c.ListIDs(query)
	}
	q := url.Values{"maxResults": {strconv.Itoa(limit)}}
	if query != "" {
		q.Set("q", query)
	}
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := c.get("/messages", q, &out); err != nil {
		return nil, errs.API("Gmail list failed for query '%s': %s", query, err.Error())
	}
	ids := make([]string, 0, len(out.Messages))
	for _, item := range out.Messages {
		if item.ID != "" {
			ids = append(ids, item.ID)
		}
	}
	return ids, nil
}

// ListPage returns one page of full messages and the next page token.
func (c *Client) ListPage(query string, limit int, page string) ([]Message, string, error) {
	if limit <= 0 {
		limit = 100
	}
	q := url.Values{"maxResults": {strconv.Itoa(limit)}}
	if query != "" {
		q.Set("q", query)
	}
	if page != "" {
		q.Set("pageToken", page)
	}
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
		Next string `json:"nextPageToken"`
	}
	if err := c.get("/messages", q, &out); err != nil {
		return nil, "", errs.API("Gmail list failed for query '%s': %s", query, err.Error())
	}
	var ids []string
	for _, item := range out.Messages {
		if item.ID != "" {
			ids = append(ids, item.ID)
		}
	}
	msgs, err := c.hydrate(ids)
	return msgs, out.Next, err
}

func (c *Client) hydrate(ids []string) ([]Message, error) {
	out := make([]Message, 0, len(ids))
	for _, id := range ids {
		msg, err := c.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

// Get loads one message in full format.
func (c *Client) Get(id string) (Message, error) {
	return c.getMessage(id, "full", nil)
}

// Meta loads From metadata for one message.
func (c *Client) Meta(id string) (Message, error) {
	return c.getMessage(id, "metadata", []string{"From"})
}

func (c *Client) getMessage(id, format string, headers []string) (Message, error) {
	q := url.Values{"format": {format}}
	for _, h := range headers {
		q.Add("metadataHeaders", h)
	}
	body, err := c.getBytes("/messages/"+url.PathEscape(id), q)
	if err != nil {
		return Message{}, errs.API("Gmail get message '%s' failed: %s", id, err.Error())
	}
	msg, err := parseMessage(body)
	if err != nil {
		return Message{}, errs.API("Gmail get message '%s' failed: %s", id, err.Error())
	}
	return msg, nil
}

// Thread loads every message in a thread.
func (c *Client) Thread(id string) ([]Message, error) {
	body, err := c.getBytes("/threads/"+url.PathEscape(id), url.Values{"format": {"full"}})
	if err != nil {
		return nil, errs.API("Gmail thread '%s' failed: %s", id, err.Error())
	}
	var raw struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, errs.API("Gmail thread '%s' failed: %s", id, err.Error())
	}
	out := make([]Message, 0, len(raw.Messages))
	for _, item := range raw.Messages {
		msg, err := parseMessage(item)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

// Send posts a raw RFC822 message. threadID is set when replying.
func (c *Client) Send(raw []byte, threadID string) (string, string, error) {
	payload := map[string]string{"raw": rawURL(raw)}
	if threadID != "" {
		payload["threadId"] = threadID
	}
	var out struct {
		ID     string `json:"id"`
		Thread string `json:"threadId"`
	}
	if err := c.post("/messages/send", payload, &out); err != nil {
		return "", "", errs.API("Gmail send failed: %s", err.Error())
	}
	return out.ID, out.Thread, nil
}

// TrashIDs adds TRASH in chunks of 1000.
func (c *Client) TrashIDs(ids []string) (int, error) {
	return c.BatchModify(ids, []string{"TRASH"}, nil)
}

// TrashOne moves one message to trash.
func (c *Client) TrashOne(id string) error {
	if err := c.post("/messages/"+url.PathEscape(id)+"/trash", map[string]string{}, nil); err != nil {
		return errs.API("Failed to trash message '%s': %s", id, err.Error())
	}
	return nil
}

// Modify changes labels on one message and returns its id and thread.
func (c *Client) Modify(id string, add, remove []string) (string, string, error) {
	body := map[string]any{}
	if len(add) > 0 {
		body["addLabelIds"] = add
	}
	if len(remove) > 0 {
		body["removeLabelIds"] = remove
	}
	var out struct {
		ID     string `json:"id"`
		Thread string `json:"threadId"`
	}
	path := "/messages/" + url.PathEscape(id) + "/modify"
	if err := c.post(path, body, &out); err != nil {
		return "", "", errs.API("Failed to modify message '%s': %s", id, err.Error())
	}
	if out.ID == "" {
		out.ID = id
	}
	return out.ID, out.Thread, nil
}

// BatchModify applies one label change to ids, in chunks of 1000.
func (c *Client) BatchModify(ids, add, remove []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	changed := 0
	for i := 0; i < len(ids); i += 1000 {
		end := i + 1000
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[i:end]
		body := map[string]any{"ids": chunk}
		if len(add) > 0 {
			body["addLabelIds"] = add
		}
		if len(remove) > 0 {
			body["removeLabelIds"] = remove
		}
		if err := c.post("/messages/batchModify", body, nil); err != nil {
			return changed, errs.API("Gmail batch modify failed: %s", err.Error())
		}
		changed += len(chunk)
	}
	return changed, nil
}

// Attachment downloads one attachment body.
func (c *Client) Attachment(msgID, attID string) ([]byte, error) {
	path := "/messages/" + url.PathEscape(msgID) + "/attachments/" + url.PathEscape(attID)
	body, err := c.getBytes(path, nil)
	if err != nil {
		return nil, errs.API("Gmail attachment failed: %s", err.Error())
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return decodeRawURL(out.Data)
}

// UnreadSenders counts From addresses on unread non-gmail mail.
func (c *Client) UnreadSenders(progress func(string, int, int)) (map[string]int, error) {
	ids, err := c.ListIDs("is:unread -from:*@gmail.com")
	if err != nil {
		return nil, errs.API("Gmail unread sender scan failed: %s", err.Error())
	}
	if progress != nil {
		progress("listed_unread_messages", len(ids), 0)
	}
	counts := map[string]int{}
	for i, id := range ids {
		msg, err := c.Meta(id)
		if err != nil {
			return nil, errs.API("Gmail unread sender scan failed: %s", err.Error())
		}
		sender := emailOf(header(msg, "from"))
		if sender != "" && !strings.HasSuffix(sender, "@gmail.com") {
			counts[sender]++
		}
		if progress != nil {
			progress("counted_sender_messages", i+1, len(ids))
		}
	}
	if progress != nil {
		progress("unique_senders", len(counts), 0)
	}
	return counts, nil
}

func (c *Client) get(path string, q url.Values, dest any) error {
	body, err := c.getBytes(path, q)
	if err != nil || dest == nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) getBytes(path string, q url.Values) ([]byte, error) {
	u := c.base() + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return c.do(http.MethodGet, u, nil)
}

func (c *Client) post(path string, payload, dest any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := c.do(http.MethodPost, c.base()+path, raw)
	if err != nil || dest == nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) do(method, u string, payload []byte) ([]byte, error) {
	if c.Access == nil {
		return nil, fmt.Errorf("missing access token")
	}
	access, err := c.Access()
	if err != nil {
		return nil, err
	}
	var r io.Reader
	if payload != nil {
		r = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, u, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%s", strings.TrimSpace(resp.Status+" "+string(body)))
	}
	return body, nil
}

// SortedIDs returns ids in lexical order, matching Python sorted(set).
func SortedIDs(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
