package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/slack/internal/app"
	"github.com/ryangerardwilson/c3po/slack/internal/cachedb"
	"github.com/ryangerardwilson/c3po/slack/internal/slackapi"
	"github.com/ryangerardwilson/c3po/slack/internal/tty"
)

// These tests run the composed runtime: the core, the Slack Web API client
// against an httptest server, and the sqlite cache in a temp dir.

var testCache cachedb.Store

func textOf(value any) string {
	text, _ := value.(string)
	return text
}

func listOf(value any) []any {
	list, _ := value.([]any)
	return list
}

func objectOf(value any) map[string]any {
	object, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return object
}

func TestPostSendUsesBatchUploadWithoutThread(t *testing.T) {
	var completePayload map[string]string
	var completeContentType string
	var getUploadAuth string
	var getUploadContentType string
	var getUploadForm url.Values
	var completeAuth string
	var uploadBody []byte
	var uploadContentType string
	var uploadURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/files.getUploadURLExternal":
			getUploadAuth = r.Header.Get("Authorization")
			getUploadContentType = r.Header.Get("Content-Type")
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse getUpload form: %v", err)
			}
			getUploadForm = r.Form
			_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + uploadURL + `","file_id":"F1"}`))
		case "/upload":
			uploadContentType = r.Header.Get("Content-Type")
			uploadBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`ok`))
		case "/api/files.completeUploadExternal":
			completeAuth = r.Header.Get("Authorization")
			completeContentType = r.Header.Get("Content-Type")
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse complete form: %v", err)
			}
			completePayload = map[string]string{}
			for key, values := range r.Form {
				if len(values) > 0 {
					completePayload[key] = values[0]
				}
			}
			_, _ = w.Write([]byte(`{"ok":true,"files":[{"shares":{"C123":[{"ts":"200.1"}]}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	uploadURL = server.URL + "/upload"

	attachPath := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(attachPath, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"bot":"xoxb-token","user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")

	err := rt.Run([]string{"1", "send", "to", "C123", "body", "caption", "attach", attachPath})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if getUploadAuth != "Bearer xoxp-user" || completeAuth != "Bearer xoxp-user" {
		t.Fatalf("expected user token for file upload, get=%q complete=%q", getUploadAuth, completeAuth)
	}
	if !strings.HasPrefix(getUploadContentType, "application/x-www-form-urlencoded") {
		t.Fatalf("expected getUpload form encoding, got %q", getUploadContentType)
	}
	if getUploadForm.Get("filename") != "note.txt" || getUploadForm.Get("length") != "2" {
		t.Fatalf("unexpected getUpload form: %#v", getUploadForm)
	}
	if string(uploadBody) != "hi" {
		t.Fatalf("expected raw upload bytes, got %q", string(uploadBody))
	}
	if strings.HasPrefix(uploadContentType, "multipart/") {
		t.Fatalf("expected raw upload content type, got %q", uploadContentType)
	}
	if !strings.HasPrefix(completeContentType, "application/x-www-form-urlencoded") {
		t.Fatalf("expected completeUpload form encoding, got %q", completeContentType)
	}
	if completePayload["thread_ts"] != "" {
		t.Fatalf("expected no thread_ts, got %#v", completePayload)
	}
	if completePayload["initial_comment"] != "caption" {
		t.Fatalf("expected initial_comment, got %#v", completePayload)
	}
	if !strings.Contains(stdout.String(), "posted target=C123") {
		t.Fatalf("stdout: %s", stdout.String())
	}
}

func TestInspectConversationUsesUserToken(t *testing.T) {
	longText := strings.Repeat("alpha ", 80)
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/conversations.history":
			payload := map[string]any{
				"ok": true,
				"messages": []map[string]any{{
					"ts":   "100.1",
					"user": "U2",
					"text": longText,
				}},
			}
			data, _ := json.Marshal(payload)
			_, _ = w.Write(data)
		case "/api/users.info":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U2","profile":{"real_name":"Mike Willbanks","email":"mike@willbanks.dev"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"bot":"xoxb-bot","user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")

	err := rt.Run([]string{"1", "inspect", "conversation", "C0B7V41SRST"})
	if err != nil {
		t.Fatalf("inspect conversation: %v", err)
	}
	if authHeader != "Bearer xoxp-user" {
		t.Fatalf("expected user token, got %q", authHeader)
	}
	if !strings.Contains(stdout.String(), strings.TrimSpace(longText)) {
		t.Fatalf("inspect should include full text, stdout: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "...") {
		t.Fatalf("inspect should not truncate text, stdout: %s", stdout.String())
	}

	stdout.Reset()
	if err := rt.Run([]string{"1", "inspect", "conversation", "C0B7V41SRST", "output", "json"}); err != nil {
		t.Fatalf("inspect conversation json: %v", err)
	}
	want := "{\n  \"messages\": [\n    {\n      \"text\": \"" + longText + "\",\n      \"ts\": \"100.1\",\n      \"user\": \"U2\"\n    }\n  ],\n  \"ok\": true\n}\n"
	if stdout.String() != want {
		t.Fatalf("inspect json should print Slack's reply, stdout: %s", stdout.String())
	}
}

func TestMarkAllReadUsesUserTokenAndCache(t *testing.T) {
	var calls []map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer xoxp-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/users.conversations":
			_, _ = w.Write([]byte(`{"ok":true,"channels":[
				{"id":"D1","is_im":true,"unread_count_display":1,"unread_count":1,"latest":{"ts":"100.000100"}},
				{"id":"G1","is_mpim":true,"unread_count_display":1,"unread_count":1,"latest":{"ts":"101.000100"}}
			]}`))
		case "/api/conversations.mark":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			calls = append(calls, map[string]string{"channel": r.Form.Get("channel"), "ts": r.Form.Get("ts")})
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	cachePath := filepath.Join(t.TempDir(), "events.db")
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`{"accounts":{"1":{"token":{"user":"xoxp-token"},"events_cache_db":%q}}}`, cachePath)
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := testCache.StoreEntries(cachePath, []app.MessageEntry{
		{
			SortTS:       100.0001,
			Email:        "maanas@example.com",
			ChannelID:    "D1",
			Surface:      "dm",
			Conversation: "Maanas",
			UserID:       "U2",
			Members:      "-",
			Message:      map[string]any{"ts": "100.000100", "user": "U2", "text": "cached dm"},
			Sender:       map[string]any{"id": "U2", "name": "Maanas", "email": "maanas@example.com", "label": "Maanas"},
			Unread:       true,
		},
		{
			SortTS:       101.0001,
			Email:        "-",
			ChannelID:    "G1",
			Surface:      "group_dm",
			Conversation: "A, B",
			UserID:       "U3",
			Members:      "3",
			Message:      map[string]any{"ts": "101.000100", "user": "U3", "text": "cached group dm"},
			Sender:       map[string]any{"id": "U3", "name": "A", "email": "-", "label": "A"},
			Unread:       true,
		},
		{
			SortTS:       102.0001,
			Email:        "-",
			ChannelID:    "D9",
			Surface:      "dm",
			Conversation: "stale-cache-only",
			UserID:       "U9",
			Members:      "-",
			Message:      map[string]any{"ts": "102.000100", "user": "U9", "text": "cached only"},
			Sender:       map[string]any{"id": "U9", "name": "Stale", "email": "-", "label": "Stale"},
			Unread:       true,
		},
	}, "", true)
	if err != nil {
		t.Fatalf("store cache: %v", err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")

	// Preset 1 marks through its own cache. A failed mark is an error, and
	// the summary line carries the marked and failed counts.
	err = rt.Run([]string{"1", "mark", "all", "read"})
	if err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	if !strings.Contains(stdout.String(), "Summary: marked_read=2 failed=0") {
		t.Fatalf("unexpected result: %s", stdout.String())
	}
	if len(calls) != 2 {
		t.Fatalf("unexpected mark calls: %#v", calls)
	}
	marked := map[string]bool{}
	for _, call := range calls {
		marked[call["channel"]] = true
	}
	if !marked["D1"] || !marked["G1"] || marked["D9"] {
		t.Fatalf("expected live D1/G1 only, got %#v", calls)
	}
	if !strings.Contains(stdout.String(), "Summary: marked_read=2 failed=0") {
		t.Fatalf("missing summary: %s", stdout.String())
	}
	entries, err := testCache.LoadEntries(cachePath, "U1", 10, "")
	if err != nil {
		t.Fatalf("reload cache: %v", err)
	}
	for _, entry := range entries {
		if (entry.ChannelID == "D1" || entry.ChannelID == "G1") && entry.Unread {
			t.Fatalf("entry still unread: %+v", entry)
		}
	}
}

func TestChannelPostUsesUserToken(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/chat.postMessage":
			_, _ = w.Write([]byte(`{"ok":true,"ts":"200.1"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"bot":"xoxb-bot","user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")

	err := rt.Run([]string{"1", "send", "to", "C0B7V41SRST", "body", "engineering update"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if authHeader != "Bearer xoxp-user" {
		t.Fatalf("expected user token for channel post, got %q", authHeader)
	}
	if !strings.Contains(stdout.String(), "posted target=C0B7V41SRST") {
		t.Fatalf("stdout: %s", stdout.String())
	}
}

func TestDeleteAndEditMessage(t *testing.T) {
	var deletePayload map[string]string
	var editPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/chat.delete":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &deletePayload)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/chat.update":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &editPayload)
			_, _ = w.Write([]byte(`{"ok":true,"ts":"200.1"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"bot":"xoxb-bot","user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")

	err := rt.Run([]string{"1", "delete", "message", "C0B7V41SRST:1781778512.813869"})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deletePayload["channel"] != "C0B7V41SRST" || deletePayload["ts"] != "1781778512.813869" {
		t.Fatalf("unexpected delete payload: %#v", deletePayload)
	}
	if !strings.Contains(stdout.String(), "deleted message_id=C0B7V41SRST:1781778512.813869") {
		t.Fatalf("delete stdout: %s", stdout.String())
	}

	stdout.Reset()
	err = rt.Run([]string{"1", "edit", "message", "C0B7V41SRST:1781778811.092529", "body", "corrected update"})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if editPayload["channel"] != "C0B7V41SRST" || editPayload["ts"] != "1781778811.092529" || editPayload["text"] != "corrected update" {
		t.Fatalf("unexpected edit payload: %#v", editPayload)
	}
	if !strings.Contains(stdout.String(), "edited message_id=C0B7V41SRST:1781778811.092529") {
		t.Fatalf("edit stdout: %s", stdout.String())
	}
}

func TestConfigEditFailsClosedOnRealNonTTY(t *testing.T) {
	// Regression for agent runners: ModeCharDevice can be true on non-terminals,
	// but termios ioctl correctly reports non-TTY. Do not override IsTTY.
	if tty.IsInteractive() {
		t.Skip("real interactive TTY present; cannot assert fail-closed path")
	}
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("EDITOR", "vim")
	t.Setenv("VISUAL", "vim")
	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, &http.Client{}, slackapi.DefaultBaseURL)
	// Leave OpenEditor nil so the real editor path is exercised up to the guard.
	done := make(chan error, 1)
	go func() {
		done <- rt.Run([]string{"config", "edit"})
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected config edit without real TTY to fail closed")
		}
		if !strings.Contains(err.Error(), "interactive TTY") {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(stdout.String(), "token") || strings.Contains(stdout.String(), "xox") {
			t.Fatalf("config edit leaked content: %s", stdout.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("config edit hung instead of failing closed without a TTY")
	}
}

func TestReplyExecutionPostsThread(t *testing.T) {
	var postPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/conversations.replies":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"1712764800.000100","thread_ts":"1712764800.000100","text":"root"}]}`))
		case "/api/conversations.history":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"1712764800.000100","thread_ts":"1712764800.000100","text":"root"}]}`))
		case "/api/chat.postMessage":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &postPayload)
			_, _ = w.Write([]byte(`{"ok":true,"ts":"1712764801.000200"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")
	err := rt.Run([]string{"1", "reply", "to", "C123ABC:1712764800.000100", "body", "Example"})
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if textOf(postPayload["channel"]) != "C123ABC" || textOf(postPayload["thread_ts"]) != "1712764800.000100" || textOf(postPayload["text"]) != "Example" {
		t.Fatalf("unexpected post payload: %#v", postPayload)
	}
	if !strings.Contains(stdout.String(), "replied message_id=C123ABC:1712764800.000100") {
		t.Fatalf("stdout: %s", stdout.String())
	}
}

func TestThreadCommandListsReplies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/conversations.replies":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[
				{"ts":"1712764800.000100","thread_ts":"1712764800.000100","user":"U2","text":"root","reply_count":1},
				{"ts":"1712764801.000200","thread_ts":"1712764800.000100","user":"U3","text":"reply"}
			]}`))
		case "/api/users.info":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U2","profile":{"real_name":"Ada"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")
	err := rt.Run([]string{"1", "thread", "C123ABC:1712764800.000100", "output", "json"})
	if err != nil {
		t.Fatalf("thread: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v stdout=%s", err, stdout.String())
	}
	if textOf(payload["thread_ts"]) != "1712764800.000100" {
		t.Fatalf("payload: %#v", payload)
	}
	messages := listOf(payload["messages"])
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %#v", payload)
	}
}

func TestListInChannelUsesHistoryOldest(t *testing.T) {
	now := time.Now().Unix()
	newTS := fmt.Sprintf("%d.100000", now-60)
	oldTS := fmt.Sprintf("%d.100000", now-120)
	var historyQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/conversations.history":
			historyQuery = r.URL.Query()
			payload := fmt.Sprintf(`{"ok":true,"messages":[
				{"ts":%q,"user":"U2","text":"new"},
				{"ts":%q,"user":"U2","text":"old"}
			]}`, newTS, oldTS)
			_, _ = w.Write([]byte(payload))
		case "/api/conversations.info":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C123ABC","name":"genie","is_channel":true}}`))
		case "/api/users.info":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U2","profile":{"real_name":"Ada"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")
	err := rt.Run([]string{"1", "list", "in", "C123ABC", "since", "4h", "limit", "10", "output", "json"})
	if err != nil {
		t.Fatalf("list in: %v", err)
	}
	if historyQuery.Get("channel") != "C123ABC" || historyQuery.Get("oldest") == "" {
		t.Fatalf("expected channel+oldest history query, got %#v", historyQuery)
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v stdout=%s", err, stdout.String())
	}
	if textOf(payload["order"]) != "newest_first" {
		t.Fatalf("expected newest_first, got %#v", payload)
	}
	messages := listOf(payload["messages"])
	if len(messages) == 0 {
		t.Fatalf("expected messages, got %#v", payload)
	}
	first := objectOf(messages[0])
	if textOf(first["ts"]) != newTS {
		t.Fatalf("expected newest first, got %#v", messages)
	}
}

func TestUnreadCountUsesLiveAPINotCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/users.conversations":
			_, _ = w.Write([]byte(`{"ok":true,"channels":[
				{"id":"C1","name":"secret-channel","unread_count_display":3,"unread_count":3},
				{"id":"D2","is_im":true,"unread_count_display":0,"unread_count":0,"has_unreads":true}
			]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(home, "state", "c3po", "slack", "events-1.db")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := testCache.StoreEntries(cachePath, []app.MessageEntry{{
		SortTS:       100.0001,
		Email:        "-",
		ChannelID:    "D9",
		Surface:      "dm",
		Conversation: "stale-cache-only",
		UserID:       "U9",
		Members:      "-",
		Message:      map[string]any{"ts": "100.000100", "user": "U9", "text": "cached only"},
		Sender:       map[string]any{"id": "U9", "name": "Stale", "email": "-", "label": "Stale"},
		Unread:       true,
	}}, "", true); err != nil {
		t.Fatalf("store cache: %v", err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")
	if err := rt.Run([]string{"1", "unread", "count", "output", "json"}); err != nil {
		t.Fatalf("unread count: %v stdout=%s", err, stdout.String())
	}
	out := stdout.String()
	if strings.Contains(out, "secret-channel") || strings.Contains(out, "stale-cache-only") || strings.Contains(out, "xoxp-") {
		t.Fatalf("leaked content: %s", out)
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v stdout=%s", err, out)
	}
	if int(payload["count"].(float64)) != 4 || int(payload["conversations"].(float64)) != 2 {
		t.Fatalf("unexpected totals: %#v", payload)
	}
}

func TestListUnreadIgnoresStaleCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1"}`))
		case "/api/users.conversations":
			_, _ = w.Write([]byte(`{"ok":true,"channels":[]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	configPath := filepath.Join(home, "config", "c3po", "slack", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"accounts":{"1":{"token":{"user":"xoxp-user"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(home, "state", "c3po", "slack", "events-1.db")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := testCache.StoreEntries(cachePath, []app.MessageEntry{{
		SortTS:       100.0001,
		Email:        "-",
		ChannelID:    "D9",
		Surface:      "dm",
		Conversation: "stale-cache-only",
		UserID:       "U9",
		Members:      "-",
		Message:      map[string]any{"ts": "100.000100", "user": "U9", "text": "cached only"},
		Sender:       map[string]any{"id": "U9", "name": "Stale", "email": "-", "label": "Stale"},
		Unread:       true,
	}}, "", true); err != nil {
		t.Fatalf("store cache: %v", err)
	}

	var stdout bytes.Buffer
	rt := newRuntime(&stdout, &bytes.Buffer{}, server.Client(), server.URL+"/api/")
	if err := rt.Run([]string{"1", "list", "unread", "output", "json"}); err != nil {
		t.Fatalf("list unread: %v stdout=%s", err, stdout.String())
	}
	out := stdout.String()
	if strings.Contains(out, "stale-cache-only") || strings.Contains(out, "cached only") {
		t.Fatalf("list unread used stale cache: %s", out)
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v stdout=%s", err, out)
	}
	if int(payload["count"].(float64)) != 0 {
		t.Fatalf("expected empty live unread list, got %#v", payload)
	}
}
