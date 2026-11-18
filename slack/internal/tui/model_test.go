package tui

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

func TestMain(m *testing.M) {
	fetchBoard = func() (Board, error) { return Board{}, nil }
	openDeskItem = func(string, string) error { return errors.New("open not stubbed") }
	readDeskItem = func(string, string) error { return errors.New("read not stubbed") }
	os.Exit(m.Run())
}

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

type deskLog struct {
	opened []string
	read   []string
}

func stubDesk(t *testing.T, next Board, openErr, readErr error) *deskLog {
	t.Helper()
	log := &deskLog{}
	oldFetch, oldOpen, oldRead := fetchBoard, openDeskItem, readDeskItem
	fetchBoard = func() (Board, error) { return next, nil }
	openDeskItem = func(preset, id string) error {
		log.opened = append(log.opened, preset+"/"+id)
		return openErr
	}
	readDeskItem = func(preset, id string) error {
		log.read = append(log.read, preset+"/"+id)
		return readErr
	}
	t.Cleanup(func() {
		fetchBoard, openDeskItem, readDeskItem = oldFetch, oldOpen, oldRead
	})
	return log
}

func sampleBoard() Board {
	return Board{
		RefreshedAt: "Wed, 18 Nov 2026, 6:04 AM IST",
		Since:       "Wed, 15 Nov 2023, 3:43 AM IST",
		Workspaces: []Workspace{
			{
				Preset: "1",
				Name:   "WIOM",
				DMs: []Item{{
					MessageID: "D1:1",
					Sender:    "Ada",
					Text:      "need the numbers",
					Date:      "Wed, 18 Nov 2026, 5:40 AM IST",
				}},
				Tagged: []Item{{
					MessageID:    "C1:2",
					Conversation: "ops",
					Text:         "see this",
					Date:         "Wed, 18 Nov 2026, 5:41 AM IST",
				}},
				Aware: []Item{{
					MessageID: "C2:3",
					Sender:    "Cy",
					Text:      "production deploy failed",
					Why:       "deploy failed",
					Date:      "Wed, 18 Nov 2026, 5:42 AM IST",
				}},
			},
			{
				Preset: "2",
				Name:   "ADE",
				DMs: []Item{{
					MessageID: "D9:9",
					Sender:    "Dee",
					Text:      "ade note",
					Date:      "Wed, 18 Nov 2026, 4:00 AM IST",
				}},
			},
		},
	}
}

func runKeys(t *testing.T, m *model, keys ...string) *fakeScreen {
	t.Helper()
	screen := &fakeScreen{width: 100, height: 24, keys: keys}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	return screen
}

func selected(frame string) string {
	const on = "\x1b[1;97m> "
	i := strings.Index(frame, on)
	if i < 0 {
		return ""
	}
	rest := frame[i+len(on):]
	j := strings.Index(rest, "\x1b[0m")
	if j < 0 {
		return rest
	}
	return rest[:j]
}

func TestDeskTableShowsTheThreeSections(t *testing.T) {
	screen := runKeys(t, newModel(sampleBoard(), nil), "q")
	frame := screen.frames[0]
	for _, want := range []string{"DMS  1", "TAGGED  1", "SHOULD KNOW  1", "WHEN", "WHO", "WHAT", "Ada", "need the numbers", "ops", "deploy failed"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "production deploy failed") || strings.Contains(frame, "j/k move") || strings.Contains(frame, "xoxp-") {
		t.Fatalf("frame\n%s", frame)
	}
	if !strings.Contains(selected(frame), "Ada") {
		t.Fatalf("selected %q", selected(frame))
	}
}

func TestQuestionMarkBlocksOpenAndRead(t *testing.T) {
	log := stubDesk(t, sampleBoard(), nil, nil)
	screen := runKeys(t, newModel(sampleBoard(), nil), "?", "j", "enter", "r", "esc", "q")
	if !strings.Contains(screen.frames[1], "workspace") || !strings.Contains(screen.frames[1], "open") || !strings.Contains(screen.frames[1], "read") {
		t.Fatalf("help\n%s", screen.frames[1])
	}
	if !strings.Contains(selected(screen.frames[0]), "Ada") || !strings.Contains(selected(screen.frames[len(screen.frames)-1]), "Ada") {
		t.Fatalf("cursor moved\n%s", selected(screen.frames[len(screen.frames)-1]))
	}
	if len(log.opened) != 0 || len(log.read) != 0 {
		t.Fatalf("opened %v read %v", log.opened, log.read)
	}
}

func TestEnterOpensWithoutMarking(t *testing.T) {
	log := stubDesk(t, sampleBoard(), nil, nil)
	runKeys(t, newModel(sampleBoard(), nil), "enter")
	if len(log.opened) != 1 || log.opened[0] != "1/D1:1" || len(log.read) != 0 {
		t.Fatalf("opened %v read %v", log.opened, log.read)
	}
}

func TestFailedOpenStaysOnTheRow(t *testing.T) {
	log := stubDesk(t, sampleBoard(), errors.New("browser refused"), nil)
	screen := runKeys(t, newModel(sampleBoard(), nil), "enter", "q")
	if len(log.opened) != 1 || len(log.read) != 0 {
		t.Fatalf("opened %v read %v", log.opened, log.read)
	}
	if !strings.Contains(screen.frames[1], "browser refused") || !strings.Contains(selected(screen.frames[1]), "Ada") {
		t.Fatalf("frame\n%s", screen.frames[1])
	}
}

func TestMarkDropsTheRow(t *testing.T) {
	next := sampleBoard()
	next.Workspaces[0].DMs = nil
	log := stubDesk(t, next, nil, nil)
	screen := runKeys(t, newModel(sampleBoard(), nil), "r", "q")
	if len(log.read) != 1 || log.read[0] != "1/D1:1" || len(log.opened) != 0 {
		t.Fatalf("read %v opened %v", log.read, log.opened)
	}
	frame := screen.frames[1]
	if strings.Contains(frame, "need the numbers") || !strings.Contains(frame, "marked") || !strings.Contains(frame, "DMS  0") {
		t.Fatalf("after mark\n%s", frame)
	}
}

func TestFailedMarkLeavesTheRow(t *testing.T) {
	log := stubDesk(t, sampleBoard(), nil, errors.New("desk busy"))
	screen := runKeys(t, newModel(sampleBoard(), nil), "r", "q")
	if len(log.read) != 1 || !strings.Contains(screen.frames[1], "desk busy") || !strings.Contains(screen.frames[1], "need the numbers") {
		t.Fatalf("frame\n%s", screen.frames[1])
	}
}

func TestHLWrapsWorkspaces(t *testing.T) {
	screen := runKeys(t, newModel(sampleBoard(), nil), "l", "l", "h", "q")
	if !strings.Contains(screen.frames[1], "ade note") || strings.Contains(screen.frames[1], "need the numbers") {
		t.Fatalf("next\n%s", screen.frames[1])
	}
	if !strings.Contains(screen.frames[2], "need the numbers") {
		t.Fatalf("wrap\n%s", screen.frames[2])
	}
	if !strings.Contains(screen.frames[3], "ade note") {
		t.Fatalf("back\n%s", screen.frames[3])
	}
}

func TestEmptyWorkspaceNamesTheIntake(t *testing.T) {
	board := Board{
		Since:      "Wed, 15 Nov 2023, 3:43 AM IST",
		Workspaces: []Workspace{{Preset: "1", Name: "WIOM"}},
	}
	log := stubDesk(t, board, nil, nil)
	screen := runKeys(t, newModel(board, nil), "enter", "q")
	frame := screen.frames[0]
	for _, want := range []string{"DMS  0", "TAGGED  0", "SHOULD KNOW  0", "watching after  Wed, 15 Nov 2023, 3:43 AM IST"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	if len(log.opened) != 0 {
		t.Fatal("opened an empty desk")
	}
}

func TestMissingDeskDoesNotOpen(t *testing.T) {
	log := stubDesk(t, Board{}, nil, nil)
	screen := runKeys(t, newModel(Board{Error: "desk unavailable"}, nil), "enter", "q")
	if !strings.Contains(screen.frames[0], "DESK UNAVAILABLE") || len(log.opened) != 0 {
		t.Fatalf("frame\n%s", screen.frames[0])
	}
}

func TestNoBriefYet(t *testing.T) {
	screen := runKeys(t, newModel(Board{}, nil), "q")
	if !strings.Contains(screen.frames[0], "no brief yet") {
		t.Fatalf("frame\n%s", screen.frames[0])
	}
}

func TestDecodeDropsAToken(t *testing.T) {
	raw := []byte(`{"refreshed_at":"Wed, 18 Nov 2026, 6:04 AM IST","workspaces":[{"preset":"1","name":"WIOM","token":"xoxp-secret","dms":[{"message_id":"D1:1","sender":"Ada","text":"hi","date":"Wed, 18 Nov 2026, 5:40 AM IST"}],"tagged":[],"aware":[]}]}`)
	board, err := decodeBoard(raw)
	if err != nil {
		t.Fatal(err)
	}
	screen := runKeys(t, newModel(board, nil), "q")
	if strings.Contains(screen.frames[0], "xoxp-secret") || !strings.Contains(screen.frames[0], "Ada") {
		t.Fatalf("frame\n%s", screen.frames[0])
	}
}

func TestResumeRestoresTheRowAndOmitsTheToken(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	first, err := tuisession.Open("slack", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Place("2", "D9:9"); err != nil {
		t.Fatal(err)
	}
	if err := first.Note(tuisession.Body{}); err != nil {
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
	screen := runKeys(t, newModel(sampleBoard(), again), "q")
	if err := again.Close(); err != nil {
		t.Fatal(err)
	}
	frame := screen.frames[0]
	if !strings.Contains(selected(frame), "Dee") || !strings.Contains(frame, "ade note") || strings.Contains(frame, "need the numbers") {
		t.Fatalf("restored\n%s", frame)
	}
	raw, ok, err := sqlxdb.Get("tui", "session", id)
	if err != nil || !ok {
		t.Fatalf("session %v %s", err, raw)
	}
	if strings.Contains(string(raw), "xoxp-") || !strings.Contains(string(raw), `"screen":"slack"`) || !strings.Contains(string(raw), `"path":"2"`) {
		t.Fatalf("body %s", raw)
	}
}
