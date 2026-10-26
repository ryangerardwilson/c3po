package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
	"github.com/ryangerardwilson/c3po/tunnel/internal/secrets"
	"github.com/ryangerardwilson/c3po/tunnel/internal/store"
)

const (
	scanEvery = 2 * time.Second
	keyWait   = 100 * time.Millisecond
)

// Run opens the route editor on the terminal.
func Run(home string, load func() (secrets.Tunnel, error), ensure store.Ensure, sess *tuisession.Session) error {
	file, err := routes.EnsureFile(routes.Path(home))
	if err != nil {
		return err
	}
	in, out := os.Stdin, os.Stdout
	restore, err := enterRaw(int(in.Fd()))
	if err != nil {
		return fmt.Errorf("c3po tui tunnel needs a terminal")
	}
	defer restore()
	m := newModel(home, file, os.Getenv("NO_COLOR") == "")
	m.session = sess
	m.scan()
	m.restoreSession()
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[0m\x1b[?25h\x1b[?1049l")
	return loop(&m, screen{in: in, out: out, load: load, ensure: ensure})
}

type screen struct {
	in     *os.File
	out    *os.File
	load   func() (secrets.Tunnel, error)
	ensure store.Ensure
}

func loop(m *model, s screen) error {
	saves := make(chan error, 1)
	next := time.Now().Add(scanEvery)
	last := ""
	for {
		last = paint(s.out, m.View(), last)
		k, ok, err := readKey(s.in, keyWait)
		if err != nil {
			return err
		}
		if ok && s.apply(m, k, saves) == actQuit {
			m.noteSession()
			return nil
		}
		m.noteSession()
		select {
		case err := <-saves:
			m.saved(err)
		default:
		}
		if time.Now().After(next) {
			m.scan()
			next = time.Now().Add(scanEvery)
		}
	}
}

// apply runs one key. A save runs in the background so the screen
// stays live while DNS is checked.
func (s screen) apply(m *model, k string, saves chan<- error) action {
	act := m.key(k)
	if act == actSave {
		home, name, port := m.home, m.name.value, m.port.value
		go func() {
			_, _, err := store.Add(home, name, port, s.load, s.ensure)
			saves <- err
		}()
	}
	return act
}

// paint draws one frame from the top left, unless it equals last. Each
// line is erased to its end, and the rest of the screen is cleared. It
// returns the frame now on screen.
func paint(out *os.File, frame, last string) string {
	if frame == last {
		return last
	}
	lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
	var b strings.Builder
	b.WriteString("\x1b[H")
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line + "\x1b[K")
	}
	b.WriteString("\x1b[J")
	_, _ = out.WriteString(b.String())
	return frame
}
