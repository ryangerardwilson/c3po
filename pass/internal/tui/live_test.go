package tui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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

// liveSource answers through the test's functions, on the screen's worker.
type liveSource struct {
	vaults func() ([]Vault, error)
	items  func(vault string) ([]Item, error)
	copy   func(vault, id string) error
	delete func(vault, id string) error
}

func (s *liveSource) Vaults(context.Context) ([]Vault, error) { return s.vaults() }

func (s *liveSource) Items(_ context.Context, vault string) ([]Item, error) { return s.items(vault) }

func (s *liveSource) Copy(_ context.Context, vault, id string) error { return s.copy(vault, id) }

func (s *liveSource) Delete(_ context.Context, vault, id string) error { return s.delete(vault, id) }

func rootVault() ([]Vault, error) { return []Vault{{ID: "vault-1", Name: "Root"}}, nil }

func threeItems(string) ([]Item, error) {
	return []Item{loginItem("a", "alpha"), loginItem("b", "bravo"), loginItem("c", "charlie")}, nil
}

func TestTheFirstFrameSaysLoadingBeforeNebulaAnswers(t *testing.T) {
	answer := make(chan error)
	src := &liveSource{vaults: func() ([]Vault, error) {
		if err := <-answer; err != nil {
			return nil, err
		}
		return rootVault()
	}}
	screen := newLiveScreen()
	done := runLive(&model{src: src}, screen)
	screen.waitFrame(t, "saying loading", func(f string) bool {
		return strings.Contains(f, "pass - vaults") && strings.Contains(f, "loading")
	})
	screen.typed <- "j"
	screen.typed <- "?"
	screen.waitFrame(t, "with the shortcuts while nebula is still listing", func(f string) bool {
		return strings.Contains(f, "dd     delete")
	})
	screen.typed <- "?"
	answer <- nil
	screen.waitFrame(t, "with the vaults", func(f string) bool {
		return strings.Contains(f, "> Root") && !strings.Contains(f, "loading")
	})
	screen.typed <- "q"
	if err := waitDone(t, done, time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestOpeningAVaultThatFailsKeepsTheVaultList(t *testing.T) {
	answer := make(chan error)
	src := &liveSource{vaults: rootVault, items: func(string) ([]Item, error) { return nil, <-answer }}
	screen := newLiveScreen()
	done := runLive(&model{src: src}, screen)
	screen.waitFrame(t, "with the vaults", func(f string) bool { return strings.Contains(f, "> Root") })
	screen.typed <- "l"
	screen.waitFrame(t, "naming the vault while it loads", func(f string) bool {
		return strings.Contains(f, "pass - Root") && strings.Contains(f, "loading")
	})
	answer <- errors.New("nebula crypt timed out")
	screen.waitFrame(t, "back on the vault list with the failure", func(f string) bool {
		return strings.Contains(f, "pass - vaults") && strings.Contains(f, "> Root") && strings.Contains(f, "timed out")
	})
	screen.typed <- "q"
	if err := waitDone(t, done, time.Second); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("leave error %v", err)
	}
}

func TestDeleteHidesTheItemBeforeNebulaAnswers(t *testing.T) {
	answer := make(chan error)
	src := &liveSource{
		vaults: rootVault,
		items:  threeItems,
		delete: func(string, string) error { return <-answer },
	}
	screen := newLiveScreen()
	done := runLive(&model{src: src}, screen)
	screen.waitFrame(t, "with the vaults", func(f string) bool { return strings.Contains(f, "> Root") })
	screen.typed <- "l"
	screen.waitFrame(t, "with the items", func(f string) bool { return strings.Contains(f, "bravo") })
	screen.typed <- "j"
	screen.typed <- "d"
	screen.typed <- "d"
	screen.waitFrame(t, "with the item gone while nebula deletes it", func(f string) bool {
		return !strings.Contains(f, "bravo  ada") && strings.Contains(f, "deleting bravo") && strings.Contains(f, "> charlie")
	})
	screen.typed <- "k"
	screen.waitFrame(t, "with the cursor moved during the delete", func(f string) bool {
		return strings.Contains(f, "> alpha") && !strings.Contains(f, "bravo  ada")
	})
	answer <- errors.New("Unable to delete the 1Password item.")
	screen.waitFrame(t, "with the item back and the failure shown", func(f string) bool {
		return strings.Contains(f, "bravo  ada") && strings.Contains(f, "Unable to delete")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestLeavingDrainsTheCopy(t *testing.T) {
	var copies atomic.Int32
	src := &liveSource{
		vaults: rootVault,
		items:  threeItems,
		copy: func(string, string) error {
			time.Sleep(100 * time.Millisecond)
			copies.Add(1)
			return nil
		},
	}
	screen := newLiveScreen()
	done := runLive(&model{src: src}, screen)
	screen.waitFrame(t, "with the vaults", func(f string) bool { return strings.Contains(f, "> Root") })
	screen.typed <- "l"
	screen.waitFrame(t, "with the items", func(f string) bool { return strings.Contains(f, "> alpha") })
	screen.typed <- "enter"
	screen.typed <- "q"
	if err := waitDone(t, done, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if copies.Load() != 1 {
		t.Fatalf("the copy was dropped: %d", copies.Load())
	}
	if !screen.sawFrame(func(f string) bool { return strings.Contains(f, "saving 1") }) {
		t.Fatal("leaving did not say it was saving the copy")
	}
}
