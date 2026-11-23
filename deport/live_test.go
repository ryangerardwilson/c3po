package deport

import (
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

func TestFreeMarksTheRowBeforeTheKillEnds(t *testing.T) {
	rows := twoPorts()
	answer := make(chan error)
	m := &model{
		load:    func() ([]occupant, error) { return rows, nil },
		release: func(int) ([]proc, error) { return nil, <-answer },
	}
	screen := newLiveScreen()
	done := runLive(m, screen)
	screen.waitFrame(t, "with both ports", func(f string) bool { return strings.Contains(f, "8082 2 node") })

	screen.typed <- "j"
	screen.typed <- "d"
	screen.typed <- "d"
	screen.waitFrame(t, "with the kill marked while it runs", func(f string) bool {
		return strings.Contains(f, "> 8082 2 node  freeing") && strings.Contains(f, "freeing 8082")
	})
	screen.typed <- "k"
	screen.waitFrame(t, "with the cursor moved during the kill", func(f string) bool {
		return strings.Contains(f, "> 8080 1 python") && strings.Contains(f, "8082 2 node  freeing")
	})

	answer <- errors.New("port 8082 is still in use")
	screen.waitFrame(t, "with the mark gone and the failure shown", func(f string) bool {
		return strings.Contains(f, "  8082 2 node") && !strings.Contains(f, "freeing") && strings.Contains(f, "still in use")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestFreeSuccessTakesThePortReadAfterTheKill(t *testing.T) {
	var mu sync.Mutex
	rows := twoPorts()
	m := &model{
		load: func() ([]occupant, error) {
			mu.Lock()
			defer mu.Unlock()
			return rows, nil
		},
		release: func(port int) ([]proc, error) {
			mu.Lock()
			defer mu.Unlock()
			rows = []occupant{{port: 8080, pid: 1, name: "python"}, {port: 8090, pid: 9, name: "arrived"}}
			return nil, nil
		},
	}
	screen := newLiveScreen()
	done := runLive(m, screen)
	screen.waitFrame(t, "with both ports", func(f string) bool { return strings.Contains(f, "8082 2 node") })
	screen.typed <- "j"
	screen.typed <- "d"
	screen.typed <- "d"
	screen.waitFrame(t, "with the port read after the kill", func(f string) bool {
		return strings.Contains(f, "8082 is free") && strings.Contains(f, "8090 9 arrived") && !strings.Contains(f, "8082 2 node")
	})
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}

func TestLeavingWaitsForTheKill(t *testing.T) {
	var kills atomic.Int32
	m := &model{
		load: func() ([]occupant, error) { return twoPorts(), nil },
		release: func(int) ([]proc, error) {
			time.Sleep(100 * time.Millisecond)
			kills.Add(1)
			return nil, nil
		},
	}
	screen := newLiveScreen()
	done := runLive(m, screen)
	screen.waitFrame(t, "with both ports", func(f string) bool { return strings.Contains(f, "8082 2 node") })
	start := time.Now()
	screen.typed <- "d"
	screen.typed <- "d"
	screen.typed <- "q"
	waitDone(t, done, 2*time.Second)
	if kills.Load() != 1 {
		t.Fatalf("the kill was dropped: %d", kills.Load())
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("leaving took %v", took)
	}
	if !screen.sawFrame(func(f string) bool { return strings.Contains(f, "freeing 1 port") }) {
		t.Fatal("leaving did not say it was freeing the port")
	}
}

func TestIdleListReadsThePortsAgain(t *testing.T) {
	old := refreshEvery
	refreshEvery = 20 * time.Millisecond
	t.Cleanup(func() { refreshEvery = old })
	var reads atomic.Int32
	m := &model{
		load: func() ([]occupant, error) {
			if reads.Add(1) == 1 {
				return twoPorts()[:1], nil
			}
			return twoPorts(), nil
		},
		release: func(int) ([]proc, error) { return nil, nil },
	}
	screen := newLiveScreen()
	done := runLive(m, screen)
	screen.waitFrame(t, "with the port that appeared", func(f string) bool { return strings.Contains(f, "8082 2 node") })
	screen.typed <- "q"
	waitDone(t, done, time.Second)
}
