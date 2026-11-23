package tui

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// liveScreen is a terminal on the real clock. The test types into keys while
// the loop runs on its own goroutine.
type liveScreen struct {
	fakeScreen
	typed chan string
}

func newLiveScreen() *liveScreen {
	return &liveScreen{fakeScreen: fakeScreen{width: 100, height: 24}, typed: make(chan string, 8)}
}

func (s *liveScreen) readUntil(deadline time.Time) (string, bool, error) {
	select {
	case key := <-s.typed:
		return key, true, nil
	case <-time.After(time.Until(deadline)):
		return "", false, nil
	}
}

func runLive(t *testing.T, m *model, screen *liveScreen) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- m.loop(screen) }()
	return done
}

func waitDone(t *testing.T, done <-chan error, within time.Duration) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(within):
		t.Fatal("the screen did not leave")
	}
}

func TestMarkPaintsTheRowGoneBeforeTheDeskAnswers(t *testing.T) {
	release := make(chan struct{})
	old := readDeskItem
	readDeskItem = func(preset, id string) (Board, error) {
		<-release
		return Board{}, errors.New("desk timed out")
	}
	t.Cleanup(func() { readDeskItem = old })
	screen := newLiveScreen()
	done := runLive(t, newModel(sampleBoard(), nil), screen)
	screen.waitFrame(t, "with the first row", func(f string) bool { return strings.Contains(f, "need the numbers") })

	screen.typed <- "r"
	screen.waitFrame(t, "with the row gone while the desk is still writing", func(f string) bool {
		return !strings.Contains(f, "need the numbers") && strings.Contains(f, "marking") && strings.Contains(f, "DMS  0")
	})
	screen.typed <- "j"
	screen.waitFrame(t, "with the cursor moved while the write is in flight", func(f string) bool {
		return strings.Contains(selected(f), "deploy failed") && !strings.Contains(f, "need the numbers")
	})

	close(release)
	screen.waitFrame(t, "with the row back and the failure shown", func(f string) bool {
		return strings.Contains(f, "need the numbers") && strings.Contains(f, "desk timed out") && strings.Contains(f, "DMS  1")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestMarkSuccessTakesTheDesksAnswer(t *testing.T) {
	next := sampleBoard()
	next.Workspaces[0].DMs = nil
	next.Workspaces[0].Tagged = append(next.Workspaces[0].Tagged, Item{MessageID: "C3:4", Text: "arrived meanwhile"})
	stubDesk(t, next, nil, nil)
	screen := newLiveScreen()
	done := runLive(t, newModel(sampleBoard(), nil), screen)
	screen.waitFrame(t, "with the first row", func(f string) bool { return strings.Contains(f, "need the numbers") })
	screen.typed <- "r"
	screen.waitFrame(t, "with the desk's answer", func(f string) bool {
		return strings.Contains(f, "marked") && strings.Contains(f, "arrived meanwhile") && !strings.Contains(f, "need the numbers")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestSyncKeepsTheKeysLive(t *testing.T) {
	stubDesk(t, freshBoard(), nil, nil)
	release := make(chan struct{})
	old := syncDesk
	syncDesk = func() error {
		<-release
		return nil
	}
	t.Cleanup(func() { syncDesk = old })
	screen := newLiveScreen()
	done := runLive(t, newModel(sampleBoard(), nil), screen)
	screen.waitFrame(t, "with the first row", func(f string) bool { return strings.Contains(f, "need the numbers") })
	screen.typed <- "s"
	screen.waitFrame(t, "saying syncing", func(f string) bool { return strings.Contains(f, "syncing") })
	screen.typed <- "j"
	screen.waitFrame(t, "with the cursor moved during the sync", func(f string) bool {
		return strings.Contains(selected(f), "ops") && strings.Contains(f, "syncing")
	})
	close(release)
	screen.waitFrame(t, "with the refreshed desk", func(f string) bool {
		return strings.Contains(f, "fresh note") && !strings.Contains(f, "syncing")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestLeavingDrainsMarksAndDoesNotWaitOnABackup(t *testing.T) {
	next := sampleBoard()
	next.Workspaces[0].DMs = nil
	var reads int
	old := readDeskItem
	readDeskItem = func(preset, id string) (Board, error) {
		time.Sleep(50 * time.Millisecond)
		reads++
		return next, nil
	}
	t.Cleanup(func() { readDeskItem = old })
	screen := newLiveScreen()
	done := runLive(t, newModel(sampleBoard(), nil), screen)
	screen.waitFrame(t, "with the first row", func(f string) bool { return strings.Contains(f, "need the numbers") })
	start := time.Now()
	screen.typed <- "r"
	screen.typed <- "q"
	waitDone(t, done, 2*time.Second)
	if reads != 1 {
		t.Fatalf("the pending mark was dropped: reads %d", reads)
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("leaving took %v", took)
	}
	var saving bool
	for _, frame := range screen.frames {
		saving = saving || strings.Contains(frame, "saving 1")
	}
	if !saving {
		t.Fatal("leaving did not say it was saving the mark")
	}
}
