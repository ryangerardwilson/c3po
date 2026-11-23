package tui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/internal/tuijob"
)

// press applies one key and lets what it queued finish, as the loop does
// before it reads a key from a terminal without a deadline.
func press(t *testing.T, m *model, key string) bool {
	t.Helper()
	if m.jobs == nil {
		m.jobs = tuijob.New()
		t.Cleanup(m.jobs.Close)
	}
	quit := m.handleKey(key)
	if !m.jobs.Wait(time.Second) {
		t.Fatal("Drive calls did not finish")
	}
	m.jobs.Apply()
	return quit
}

// liveScreen is a terminal on the real clock. The test types into keys
// while the loop runs on its own goroutine.
type liveScreen struct {
	mu     sync.Mutex
	frames []string
	typed  chan string
}

func newLiveScreen() *liveScreen {
	return &liveScreen{typed: make(chan string, 8)}
}

func (s *liveScreen) size() (int, int) { return 80, 24 }

func (s *liveScreen) paint(frame string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames = append(s.frames, frame)
}

func (s *liveScreen) readKey() (string, error) { return <-s.typed, nil }

func (s *liveScreen) readUntil(deadline time.Time) (string, bool, error) {
	select {
	case key := <-s.typed:
		return key, true, nil
	case <-time.After(time.Until(deadline)):
		return "", false, nil
	}
}

func (s *liveScreen) lastFrame() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.frames) == 0 {
		return ""
	}
	return s.frames[len(s.frames)-1]
}

func (s *liveScreen) sawFrame(ok func(string) bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, frame := range s.frames {
		if ok(frame) {
			return true
		}
	}
	return false
}

// waitFrame waits until the last painted frame satisfies ok.
func (s *liveScreen) waitFrame(t *testing.T, what string, ok func(string) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ok(s.lastFrame()) {
		if time.Now().After(deadline) {
			t.Fatalf("no frame %s; last\n%s", what, s.lastFrame())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func runLive(m *model, screen *liveScreen) <-chan error {
	done := make(chan error, 1)
	go func() { done <- m.loop(screen) }()
	return done
}

func waitDone(t *testing.T, done <-chan error, within time.Duration) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(within):
		t.Fatal("the screen did not leave")
	}
	return nil
}

// liveDrive answers through the test's functions, on the screen's worker.
type liveDrive struct {
	list  func(parentID string) ([]driveapi.NavEntry, error)
	trash func(fileID string) error
}

func (d *liveDrive) ListChildren(_ context.Context, parentID string) ([]driveapi.NavEntry, error) {
	return d.list(parentID)
}

func (d *liveDrive) FindAvailableName(_ context.Context, _, name string) (string, error) {
	return name, nil
}

func (d *liveDrive) CreateFolder(context.Context, string, string) (string, error) {
	return "", errors.New("no folders here")
}

func (d *liveDrive) UploadFile(context.Context, string, string, string) (string, error) {
	return "", errors.New("no uploads here")
}

func (d *liveDrive) TrashEntry(_ context.Context, fileID string) error { return d.trash(fileID) }

func myDrive(parentID string) ([]driveapi.NavEntry, error) {
	if parentID != "root" {
		return nil, nil
	}
	return []driveapi.NavEntry{
		{ID: "dir", Name: "Album", MimeType: driveapi.FolderMime},
		{ID: "a", Name: "a.txt", MimeType: "text/plain"},
		{ID: "b", Name: "b.txt", MimeType: "text/plain"},
	}, nil
}

func connected(drive Drive) Connect {
	return func(context.Context) (Drive, error) { return drive, nil }
}

func TestTheFirstFrameSaysLoadingBeforeDriveAnswers(t *testing.T) {
	answer := make(chan error)
	connect := func(context.Context) (Drive, error) {
		if err := <-answer; err != nil {
			return nil, err
		}
		return &liveDrive{list: myDrive}, nil
	}
	screen := newLiveScreen()
	done := runLive(newModel(connect), screen)
	screen.waitFrame(t, "saying loading before the token refresh", func(f string) bool {
		return strings.Contains(f, "gdrive browse - My Drive") && strings.Contains(f, "loading")
	})
	screen.typed <- "?"
	screen.waitFrame(t, "with the shortcuts while Drive is still connecting", func(f string) bool {
		return strings.Contains(f, "dd     delete")
	})
	screen.typed <- "?"
	answer <- nil
	screen.waitFrame(t, "with the folder", func(f string) bool {
		return strings.Contains(f, "> dir   Album") && strings.Contains(f, "b.txt") && !strings.Contains(f, "loading")
	})
	screen.typed <- "q"
	if err := waitDone(t, done, time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestAFailedConnectIsShownAndReturned(t *testing.T) {
	connect := func(context.Context) (Drive, error) { return nil, errors.New("oauth refresh failed: offline") }
	screen := newLiveScreen()
	done := runLive(newModel(connect), screen)
	screen.waitFrame(t, "with the failure", func(f string) bool {
		return strings.Contains(f, "oauth refresh failed") && strings.Contains(f, "No files.")
	})
	screen.typed <- "q"
	if err := waitDone(t, done, time.Second); err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("leave error %v", err)
	}
}

func TestOpeningAFolderThatFailsKeepsTheFolder(t *testing.T) {
	answer := make(chan error)
	list := func(parentID string) ([]driveapi.NavEntry, error) {
		if parentID == "dir" {
			return nil, <-answer
		}
		return myDrive(parentID)
	}
	screen := newLiveScreen()
	done := runLive(newModel(connected(&liveDrive{list: list})), screen)
	screen.waitFrame(t, "with the folder", func(f string) bool { return strings.Contains(f, "> dir   Album") })
	screen.typed <- "l"
	screen.waitFrame(t, "naming the folder while it loads", func(f string) bool {
		return strings.Contains(f, "gdrive browse - My Drive/Album") && strings.Contains(f, "loading")
	})
	answer <- errors.New("Drive timed out")
	screen.waitFrame(t, "back in My Drive with the failure", func(f string) bool {
		return strings.Contains(f, "gdrive browse - My Drive"+styleReset) && strings.Contains(f, "> dir   Album") && strings.Contains(f, "Drive timed out")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestDeleteHidesTheFileBeforeDriveAnswers(t *testing.T) {
	answer := make(chan error)
	drive := &liveDrive{list: myDrive, trash: func(string) error { return <-answer }}
	screen := newLiveScreen()
	done := runLive(newModel(connected(drive)), screen)
	screen.waitFrame(t, "with the folder", func(f string) bool { return strings.Contains(f, "b.txt") })
	screen.typed <- "G"
	screen.typed <- "d"
	screen.typed <- "d"
	screen.waitFrame(t, "with the file gone while Drive trashes it", func(f string) bool {
		return !strings.Contains(f, "file  b.txt") && strings.Contains(f, "deleting b.txt") && strings.Contains(f, "> file  a.txt")
	})
	screen.typed <- "k"
	screen.waitFrame(t, "with the cursor moved during the delete", func(f string) bool {
		return strings.Contains(f, "> dir   Album") && !strings.Contains(f, "file  b.txt")
	})
	answer <- errors.New("googleapi: Error 403: forbidden")
	screen.waitFrame(t, "with the file back and the failure shown", func(f string) bool {
		return strings.Contains(f, "file  b.txt") && strings.Contains(f, "forbidden")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestLeavingDrainsTheDelete(t *testing.T) {
	var trashed atomic.Int32
	drive := &liveDrive{list: myDrive, trash: func(string) error {
		time.Sleep(100 * time.Millisecond)
		trashed.Add(1)
		return nil
	}}
	screen := newLiveScreen()
	done := runLive(newModel(connected(drive)), screen)
	screen.waitFrame(t, "with the folder", func(f string) bool { return strings.Contains(f, "b.txt") })
	screen.typed <- "G"
	screen.typed <- "d"
	screen.typed <- "d"
	screen.typed <- "q"
	if err := waitDone(t, done, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if trashed.Load() != 1 {
		t.Fatalf("the delete was dropped: %d", trashed.Load())
	}
	if !screen.sawFrame(func(f string) bool { return strings.Contains(f, "saving 1") }) {
		t.Fatal("leaving did not say it was saving the delete")
	}
}
