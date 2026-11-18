package tui

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

type fakeScreen struct {
	keys   []string
	frames []string
	width  int
	height int
}

func (f *fakeScreen) size() (int, int) { return f.width, f.height }

func (f *fakeScreen) paint(frame string) { f.frames = append(f.frames, frame) }

func (f *fakeScreen) readKey() (string, error) {
	if len(f.keys) == 0 {
		return "", io.EOF
	}
	key := f.keys[0]
	f.keys = f.keys[1:]
	return key, nil
}

type sent struct {
	preset, channel, text, thread string
}

type marked struct {
	preset, channel, ts string
}

type fakeStore struct {
	convs   map[string][]Conversation
	hist    map[string][]Message
	sent    []sent
	marked  []marked
	sendErr error
	token   string
}

func (f *fakeStore) Conversations(preset string) ([]Conversation, error) {
	return f.convs[preset], nil
}

func (f *fakeStore) History(preset, channel string) ([]Message, error) {
	return f.hist[preset+"/"+channel], nil
}

func (f *fakeStore) Send(preset, channel, text, thread string) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, sent{preset, channel, text, thread})
	return nil
}

func (f *fakeStore) Mark(preset, channel, ts string) error {
	f.marked = append(f.marked, marked{preset, channel, ts})
	return nil
}

func workspaces() []Preset {
	return []Preset{{ID: "1", Name: "work"}, {ID: "2", Name: "ade"}}
}

func store() *fakeStore {
	return &fakeStore{
		token: "xoxp-secret-token",
		convs: map[string][]Conversation{
			"1": {{ID: "D1", Name: "Ada", Surface: "dm", Unread: 2, Latest: "200.2"}},
			"2": {{ID: "C9", Name: "general", Surface: "channel", Unread: 0, Latest: "300.1"}},
		},
		hist: map[string][]Message{
			"1/D1": {
				{TS: "100.1", User: "Ada", Text: "earlier"},
				{TS: "200.2", ThreadTS: "100.1", User: "Ada", Text: "later"},
			},
			"2/C9": {{TS: "300.1", User: "Bea", Text: "beta"}},
		},
	}
}

func runKeys(t *testing.T, m *model, keys ...string) *fakeScreen {
	t.Helper()
	screen := &fakeScreen{width: 80, height: 24, keys: keys}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	return screen
}

func TestWorkspaceListHidesShortcutsAndTheToken(t *testing.T) {
	kept := store()
	m := newModel(workspaces(), kept, nil)
	screen := runKeys(t, m, "q")
	if strings.Contains(screen.frames[0], "j/k move") || !strings.Contains(screen.frames[0], "> 1  work") {
		t.Fatalf("first\n%s", screen.frames[0])
	}
	if strings.Contains(screen.frames[0], kept.token) {
		t.Fatal("token on the workspace list")
	}
}

func TestQuestionMarkBlocksOpen(t *testing.T) {
	kept := store()
	m := newModel(workspaces(), kept, nil)
	screen := runKeys(t, m, "?", "j", "enter", "esc", "q")
	if !strings.Contains(screen.frames[1], "workspace") || !strings.Contains(screen.frames[1], "write") || !strings.Contains(screen.frames[1], "quit") {
		t.Fatalf("help\n%s", screen.frames[1])
	}
	if !strings.Contains(screen.frames[2], "workspace") || strings.Contains(screen.frames[2], "> 2  ade") {
		t.Fatalf("j while help\n%s", screen.frames[2])
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "> 1  work") || strings.Contains(last, "j/k move") {
		t.Fatalf("after esc\n%s", last)
	}
	if len(kept.convs["1"]) == 0 {
		t.Fatal("fixture")
	}
	if kept.hist == nil {
		t.Fatal("fixture")
	}
	if len(kept.sent) != 0 || len(kept.marked) != 0 {
		t.Fatalf("sent %v marked %v", kept.sent, kept.marked)
	}
}

func TestEnterOpensMessagesWithoutMarking(t *testing.T) {
	kept := store()
	m := newModel(workspaces(), kept, nil)
	screen := runKeys(t, m, "enter", "enter", "q")
	view := strings.Join(screen.frames, "\n")
	if !strings.Contains(view, "Ada  later") || !strings.Contains(view, "Ada  earlier") {
		t.Fatalf("messages\n%s", view)
	}
	if len(kept.marked) != 0 {
		t.Fatalf("marked on open %v", kept.marked)
	}
}

func TestHLSwitchesWorkspace(t *testing.T) {
	kept := store()
	m := newModel(workspaces(), kept, nil)
	screen := runKeys(t, m, "enter", "l", "h", "q")
	if !strings.Contains(screen.frames[2], "slack  2  ade") || !strings.Contains(screen.frames[2], "general") {
		t.Fatalf("next\n%s", screen.frames[2])
	}
	if !strings.Contains(screen.frames[3], "slack  1  work") || !strings.Contains(screen.frames[3], "Ada") {
		t.Fatalf("back\n%s", screen.frames[3])
	}
	screen = runKeys(t, newModel(workspaces(), kept, nil), "l", "h", "q")
	if !strings.Contains(screen.frames[1], "> 2  ade") || !strings.Contains(screen.frames[2], "> 1  work") {
		t.Fatalf("workspace switch\n%s", strings.Join(screen.frames, "\n"))
	}
}

func TestWriteAndReply(t *testing.T) {
	kept := store()
	m := newModel(workspaces(), kept, nil)
	runKeys(t, m, "enter", "enter", "i", "h", "i", "enter", "q")
	if len(kept.sent) != 1 || kept.sent[0].text != "hi" || kept.sent[0].thread != "" || kept.sent[0].channel != "D1" {
		t.Fatalf("write %#v", kept.sent)
	}
	kept.sent = nil
	m = newModel(workspaces(), kept, nil)
	runKeys(t, m, "enter", "enter", "k", "r", "y", "o", "enter", "q")
	if len(kept.sent) != 1 || kept.sent[0].text != "yo" || kept.sent[0].thread != "100.1" {
		t.Fatalf("reply %#v", kept.sent)
	}
}

func TestEmptyDraftDoesNotSend(t *testing.T) {
	kept := store()
	m := newModel(workspaces(), kept, nil)
	runKeys(t, m, "enter", "enter", "i", "enter", "q")
	if len(kept.sent) != 0 {
		t.Fatalf("sent %#v", kept.sent)
	}
}

func TestMarkIsExplicit(t *testing.T) {
	kept := store()
	m := newModel(workspaces(), kept, nil)
	screen := runKeys(t, m, "enter", "m", "q")
	if len(kept.marked) != 1 || kept.marked[0].ts != "200.2" || kept.marked[0].channel != "D1" {
		t.Fatalf("mark %#v", kept.marked)
	}
	if !strings.Contains(screen.frames[len(screen.frames)-1], "marked") {
		t.Fatalf("status\n%s", screen.frames[len(screen.frames)-1])
	}
}

func TestFailedSendKeepsTheDraft(t *testing.T) {
	kept := store()
	kept.sendErr = errors.New("slack refused")
	m := newModel(workspaces(), kept, nil)
	screen := runKeys(t, m, "enter", "enter", "i", "h", "i", "enter", "esc", "q")
	joined := strings.Join(screen.frames, "\n")
	if !strings.Contains(joined, "> hi") || !strings.Contains(joined, "slack refused") {
		t.Fatalf("draft\n%s", joined)
	}
	if len(kept.sent) != 0 {
		t.Fatal("sent")
	}
}

func TestEmptyAccountDoesNotCallSlack(t *testing.T) {
	kept := store()
	m := newModel(nil, kept, nil)
	screen := runKeys(t, m, "enter", "q")
	if !strings.Contains(screen.frames[len(screen.frames)-1], "no workspaces") {
		t.Fatalf("empty\n%s", screen.frames[len(screen.frames)-1])
	}
	if len(kept.sent) != 0 || len(kept.marked) != 0 {
		t.Fatal("called slack")
	}
}

func TestResumeRestoresTheMessageAndOmitsTheToken(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	first, err := tuisession.Open("slack", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Place("2", "300.1"); err != nil {
		t.Fatal(err)
	}
	if err := first.Note(tuisession.Body{Args: []string{"messages", "C9"}}); err != nil {
		t.Fatal(err)
	}
	id := first.ID()
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := tuisession.Open("slack", id)
	if err != nil {
		t.Fatal(err)
	}
	kept := store()
	m := newModel(workspaces(), kept, again)
	screen := runKeys(t, m, "q")
	if err := again.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(screen.frames[0], "general") || !strings.Contains(screen.frames[0], "Bea  beta") {
		t.Fatalf("restored\n%s", screen.frames[0])
	}
	raw, ok, err := sqlxdb.Get("tui", "session", id)
	if err != nil || !ok {
		t.Fatalf("session %v %s", err, raw)
	}
	if strings.Contains(string(raw), kept.token) || !strings.Contains(string(raw), `"screen":"slack"`) || !strings.Contains(string(raw), `"path":"2"`) {
		t.Fatalf("body %s", raw)
	}
}
