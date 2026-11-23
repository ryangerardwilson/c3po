package tui

import (
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

func TestMain(m *testing.M) {
	fetchBoard = func() (Board, error) { return Board{}, nil }
	openDeskItem = func(string, string) error { return errors.New("open not stubbed") }
	readDeskItem = func(string, string) (Board, error) { return Board{}, errors.New("read not stubbed") }
	syncDesk = func() error { return errors.New("sync not stubbed") }
	os.Exit(m.Run())
}

type fakeScreen struct {
	mu     sync.Mutex
	keys   []string
	frames []string
	width  int
	height int
}

func (f *fakeScreen) size() (int, int) { return f.width, f.height }

func (f *fakeScreen) paint(frame string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.frames = append(f.frames, frame)
}

// lastFrame is safe to call from a desk stub on the screen's worker.
func (f *fakeScreen) lastFrame() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.frames) == 0 {
		return ""
	}
	return f.frames[len(f.frames)-1]
}

// waitFrame waits until a painted frame satisfies ok.
func (f *fakeScreen) waitFrame(t *testing.T, what string, ok func(string) bool) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if frame := f.lastFrame(); ok(frame) {
			return frame
		}
		if time.Now().After(deadline) {
			t.Fatalf("no frame %s; last\n%s", what, f.lastFrame())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

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
	readDeskItem = func(preset, id string) (Board, error) {
		log.read = append(log.read, preset+"/"+id)
		if readErr != nil {
			return Board{}, readErr
		}
		return next, nil
	}
	t.Cleanup(func() {
		fetchBoard, openDeskItem, readDeskItem = oldFetch, oldOpen, oldRead
	})
	return log
}

func stubSync(t *testing.T, err error) *int {
	t.Helper()
	n := 0
	old := syncDesk
	syncDesk = func() error {
		n++
		return err
	}
	t.Cleanup(func() { syncDesk = old })
	return &n
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
	for _, want := range []string{"DMS  1", "TAGGED  1", "SHOULD KNOW  1", "WHEN", "WHO", "WHAT", "Ada", "need the numbers", "ops", "deploy failed", "last synced at Wed, 18 Nov 2026, 6:04 AM IST"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "production deploy failed") || strings.Contains(frame, "j/k move") || strings.Contains(frame, "xoxp-") || strings.Contains(frame, "watching after") {
		t.Fatalf("frame\n%s", frame)
	}
	if !strings.Contains(selected(frame), "Ada") {
		t.Fatalf("selected %q", selected(frame))
	}
}

func TestQuestionMarkBlocksOpenAndRead(t *testing.T) {
	log := stubDesk(t, sampleBoard(), nil, nil)
	syncs := stubSync(t, nil)
	screen := runKeys(t, newModel(sampleBoard(), nil), "?", "j", "enter", "r", "s", "esc", "q")
	help := screen.frames[1]
	for _, want := range []string{"workspace", "open", "read", "sync"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q\n%s", want, help)
		}
	}
	if !strings.Contains(selected(screen.frames[0]), "Ada") || !strings.Contains(selected(screen.frames[len(screen.frames)-1]), "Ada") {
		t.Fatalf("cursor moved\n%s", selected(screen.frames[len(screen.frames)-1]))
	}
	if len(log.opened) != 0 || len(log.read) != 0 || *syncs != 0 {
		t.Fatalf("opened %v read %v syncs %d", log.opened, log.read, *syncs)
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
	if !strings.Contains(screen.frames[1], "opening") {
		t.Fatalf("frame\n%s", screen.frames[1])
	}
	if !strings.Contains(screen.frames[2], "browser refused") || !strings.Contains(selected(screen.frames[2]), "Ada") {
		t.Fatalf("frame\n%s", screen.frames[2])
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
	marking := screen.frames[1]
	if strings.Contains(marking, "need the numbers") || !strings.Contains(marking, "marking") || !strings.Contains(marking, "DMS  0") {
		t.Fatalf("while marking\n%s", marking)
	}
	frame := screen.frames[2]
	if strings.Contains(frame, "need the numbers") || !strings.Contains(frame, "marked") || !strings.Contains(frame, "DMS  0") {
		t.Fatalf("after mark\n%s", frame)
	}
}

func TestFailedMarkLeavesTheRow(t *testing.T) {
	log := stubDesk(t, sampleBoard(), nil, errors.New("desk busy"))
	screen := runKeys(t, newModel(sampleBoard(), nil), "r", "q")
	if len(log.read) != 1 || strings.Contains(screen.frames[1], "need the numbers") {
		t.Fatalf("while marking\n%s", screen.frames[1])
	}
	if !strings.Contains(screen.frames[2], "desk busy") || !strings.Contains(screen.frames[2], "need the numbers") || !strings.Contains(selected(screen.frames[2]), "Ada") {
		t.Fatalf("frame\n%s", screen.frames[2])
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

func TestJKWrapsTheRows(t *testing.T) {
	screen := runKeys(t, newModel(sampleBoard(), nil), "j", "j", "j", "k", "q")
	if !strings.Contains(selected(screen.frames[2]), "deploy failed") {
		t.Fatalf("last\n%s", selected(screen.frames[2]))
	}
	if !strings.Contains(selected(screen.frames[3]), "Ada") {
		t.Fatalf("wrap down\n%s", selected(screen.frames[3]))
	}
	if !strings.Contains(selected(screen.frames[4]), "deploy failed") {
		t.Fatalf("wrap up\n%s", selected(screen.frames[4]))
	}
}

func TestEmptyRowsDoNotMove(t *testing.T) {
	board := Board{Workspaces: []Workspace{{Preset: "1", Name: "WIOM"}}}
	m := newModel(board, nil)
	runKeys(t, m, "j", "k", "q")
	if m.cursor != 0 {
		t.Fatalf("empty %d", m.cursor)
	}
	one := newModel(sampleBoard(), nil)
	runKeys(t, one, "l", "j", "k", "q")
	if one.cursor != 0 {
		t.Fatalf("one %d", one.cursor)
	}
}

func TestEmptyWorkspaceSaysNoItems(t *testing.T) {
	board := Board{
		Since:      "Wed, 15 Nov 2023, 3:43 AM IST",
		Workspaces: []Workspace{{Preset: "1", Name: "WIOM"}},
	}
	log := stubDesk(t, board, nil, nil)
	screen := runKeys(t, newModel(board, nil), "enter", "q")
	frame := screen.frames[0]
	for _, want := range []string{"DMS  0", "TAGGED  0", "SHOULD KNOW  0", "no items"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "watching after") || strings.Contains(frame, "Wed, 15 Nov 2023, 3:43 AM IST") {
		t.Fatalf("intake painted\n%s", frame)
	}
	if len(log.opened) != 0 {
		t.Fatal("opened an empty desk")
	}
}

func TestSyncReloadsTheDesk(t *testing.T) {
	next := sampleBoard()
	next.RefreshedAt = "2026-11-18T03:16:09Z"
	next.Workspaces[0].DMs = append([]Item{{
		MessageID: "D2:2",
		Sender:    "Bea",
		Text:      "fresh note",
		Date:      "Wed, 18 Nov 2026, 8:40 AM IST",
	}}, next.Workspaces[0].DMs...)
	stubDesk(t, next, nil, nil)
	syncs := stubSync(t, nil)
	screen := runKeys(t, newModel(sampleBoard(), nil), "s", "q")
	if *syncs != 1 || len(screen.frames) < 3 {
		t.Fatalf("syncs %d frames %d", *syncs, len(screen.frames))
	}
	if !strings.Contains(screen.frames[1], "syncing") || !strings.Contains(screen.frames[1], "need the numbers") {
		t.Fatalf("syncing\n%s", screen.frames[1])
	}
	frame := screen.frames[2]
	if !strings.Contains(frame, "fresh note") || !strings.Contains(frame, "last synced at Wed, 18 Nov 2026, 8:46 AM IST") || strings.Contains(frame, "syncing") {
		t.Fatalf("after\n%s", frame)
	}
}

func TestFailedSyncKeepsTheTable(t *testing.T) {
	syncs := stubSync(t, errors.New("slack refused"))
	screen := runKeys(t, newModel(sampleBoard(), nil), "s", "q")
	frame := screen.frames[2]
	if *syncs != 1 || !strings.Contains(frame, "need the numbers") || !strings.Contains(frame, "slack refused") {
		t.Fatalf("syncs %d\n%s", *syncs, frame)
	}
}

func TestErrorAndLastSyncedBothShow(t *testing.T) {
	board := sampleBoard()
	board.RefreshedAt = "2026-11-18T03:16:09Z"
	board.Workspaces[0].Error = "search failed"
	screen := runKeys(t, newModel(board, nil), "q")
	frame := screen.frames[0]
	if !strings.Contains(frame, "last synced at Wed, 18 Nov 2026, 8:46 AM IST") || !strings.Contains(frame, "SEARCH FAILED") || !strings.Contains(frame, "need the numbers") {
		t.Fatalf("frame\n%s", frame)
	}
}

func TestSyncedAtUsesIST(t *testing.T) {
	if got := syncedAt("2026-11-18T03:16:09Z"); got != "Wed, 18 Nov 2026, 8:46 AM IST" {
		t.Fatalf("got %q", got)
	}
	stored := "Wed, 18 Nov 2026, 6:04 AM IST"
	if syncedAt(stored) != stored || syncedAt("  ") != "" {
		t.Fatalf("stored %q blank %q", syncedAt(stored), syncedAt("  "))
	}
}

func TestRefreshAlreadyRunningIsAFailure(t *testing.T) {
	err := syncResult("refreshed: now error: refresh already running", nil)
	if err == nil || err.Error() != "refresh already running" {
		t.Fatalf("err %v", err)
	}
	if err := syncResult("refreshed: now", nil); err != nil {
		t.Fatal(err)
	}
	if err := syncResult("", errors.New("desk unavailable")); err == nil || err.Error() != "desk unavailable" {
		t.Fatalf("err %v", err)
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
	if err := first.Note(tuisession.Place{}); err != nil {
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
	stored, err := tuisession.Open("slack", id)
	if err != nil {
		t.Fatal(err)
	}
	defer stored.Close()
	body, ok := stored.Restored()
	raw := strings.Join(append([]string{body.CWD, body.Path, body.Cursor}, body.Args...), " ")
	if !ok || strings.Contains(raw, "xoxp-") || body.Screen != "slack" || body.Path != "2" {
		t.Fatalf("body %+v", body)
	}
}

func freshBoard() Board {
	next := sampleBoard()
	next.RefreshedAt = "2026-11-18T03:16:09Z"
	next.Workspaces[0].DMs = append([]Item{{
		MessageID: "D2:2",
		Sender:    "Bea",
		Text:      "fresh note",
		Date:      "Wed, 18 Nov 2026, 8:40 AM IST",
	}}, next.Workspaces[0].DMs...)
	return next
}

type timedKey struct {
	at  time.Time
	key string
}

type timedScreen struct {
	fakeScreen
	at       time.Time
	keys     []timedKey
	timeouts int
	settle   func()
}

// readUntil lets queued desk work finish first, so a fake clock does not
// race the worker.
func (s *timedScreen) readUntil(deadline time.Time) (string, bool, error) {
	if s.settle != nil {
		s.settle()
	}
	if len(s.keys) > 0 && !s.keys[0].at.After(deadline) {
		if s.keys[0].at.After(s.at) {
			s.at = s.keys[0].at
		}
		key := s.keys[0].key
		s.keys = s.keys[1:]
		return key, true, nil
	}
	s.timeouts++
	if s.timeouts > 4 {
		return "", false, errors.New("too many timeouts")
	}
	if len(s.keys) == 0 {
		return "", false, io.EOF
	}
	if deadline.After(s.at) {
		s.at = deadline
	}
	return "", false, nil
}

func bindClock(t *testing.T, s *timedScreen) {
	t.Helper()
	old := now
	now = func() time.Time { return s.at }
	t.Cleanup(func() { now = old })
}

func runTimed(t *testing.T, m *model, start time.Time, keys ...timedKey) *timedScreen {
	t.Helper()
	screen := &timedScreen{fakeScreen: fakeScreen{width: 100, height: 24}, at: start}
	screen.keys = append([]timedKey(nil), keys...)
	screen.settle = func() { m.jobs.Wait(time.Second) }
	bindClock(t, screen)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	return screen
}

func clockStart() time.Time {
	return time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)
}

func TestAutoSyncRefreshesTheDesk(t *testing.T) {
	stubDesk(t, freshBoard(), nil, nil)
	start := clockStart()
	screen := &timedScreen{fakeScreen: fakeScreen{width: 100, height: 24}, at: start}
	screen.keys = []timedKey{{at: start.Add(syncEvery + time.Minute), key: "q"}}
	bindClock(t, screen)
	m := newModel(sampleBoard(), nil)
	screen.settle = func() { m.jobs.Wait(time.Second) }
	n := 0
	old := syncDesk
	syncDesk = func() error {
		n++
		screen.waitFrame(t, "saying syncing over the old rows", func(frame string) bool {
			return strings.Contains(frame, "syncing") && strings.Contains(frame, "need the numbers")
		})
		return nil
	}
	t.Cleanup(func() { syncDesk = old })
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("syncs %d", n)
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "fresh note") || !strings.Contains(last, "last synced at Wed, 18 Nov 2026, 8:46 AM IST") || strings.Contains(last, "syncing") {
		t.Fatalf("after\n%s", last)
	}
}

func TestOpenDoesNotSync(t *testing.T) {
	syncs := stubSync(t, nil)
	start := clockStart()
	runTimed(t, newModel(sampleBoard(), nil), start, timedKey{at: start.Add(time.Minute), key: "q"})
	if *syncs != 0 {
		t.Fatalf("syncs %d", *syncs)
	}
}

func TestHelpHoldsTheRefresh(t *testing.T) {
	syncs := stubSync(t, nil)
	start := clockStart()
	runTimed(t, newModel(sampleBoard(), nil), start,
		timedKey{at: start, key: "?"},
		timedKey{at: start.Add(15 * time.Minute), key: "q"},
	)
	if *syncs != 0 {
		t.Fatalf("syncs %d", *syncs)
	}
}

func TestHelpCloseSyncsWhenDue(t *testing.T) {
	stubDesk(t, freshBoard(), nil, nil)
	syncs := stubSync(t, nil)
	start := clockStart()
	screen := runTimed(t, newModel(sampleBoard(), nil), start,
		timedKey{at: start, key: "?"},
		timedKey{at: start.Add(15 * time.Minute), key: "esc"},
		timedKey{at: start.Add(16 * time.Minute), key: "q"},
	)
	if *syncs != 1 {
		t.Fatalf("syncs %d", *syncs)
	}
	var syncing string
	for _, frame := range screen.frames {
		if strings.Contains(frame, "syncing") {
			syncing = frame
		}
	}
	if syncing == "" || strings.Contains(syncing, "j/k    move") || !strings.Contains(syncing, "need the numbers") {
		t.Fatalf("syncing\n%s", syncing)
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "fresh note") || !strings.Contains(last, "last synced at Wed, 18 Nov 2026, 8:46 AM IST") {
		t.Fatalf("after\n%s", last)
	}
}

func TestFailedAutoSyncKeepsTheTable(t *testing.T) {
	syncs := stubSync(t, errors.New("slack refused"))
	start := clockStart()
	screen := runTimed(t, newModel(sampleBoard(), nil), start,
		timedKey{at: start.Add(syncEvery + 30*time.Second), key: "q"},
	)
	if *syncs != 1 {
		t.Fatalf("syncs %d", *syncs)
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "need the numbers") || !strings.Contains(last, "slack refused") {
		t.Fatalf("frame\n%s", last)
	}
}

func TestManualSyncRestartsTheWait(t *testing.T) {
	syncs := stubSync(t, nil)
	start := clockStart()
	runTimed(t, newModel(sampleBoard(), nil), start,
		timedKey{at: start.Add(9 * time.Minute), key: "s"},
		timedKey{at: start.Add(10*time.Minute + 30*time.Second), key: "q"},
	)
	if *syncs != 1 {
		t.Fatalf("syncs %d", *syncs)
	}
}
