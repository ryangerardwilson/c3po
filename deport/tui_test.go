package deport

import (
	"io"
	"strings"
	"testing"
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

func twoPorts() []occupant {
	return []occupant{
		{port: 8080, pid: 1, name: "python"},
		{port: 8082, pid: 2, name: "node"},
	}
}

func TestJKMovesAcrossPorts(t *testing.T) {
	rows := twoPorts()
	m := &model{
		load:    func() ([]occupant, error) { return rows, nil },
		release: func(int) ([]proc, error) { return nil, nil },
	}
	screen := &fakeScreen{width: 80, height: 24, keys: []string{"j", "k", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(screen.frames) < 3 {
		t.Fatalf("frames %d", len(screen.frames))
	}
	if !strings.Contains(screen.frames[0], "> 8080 1 python") || strings.Contains(screen.frames[0], "> 8082") {
		t.Fatalf("first\n%s", screen.frames[0])
	}
	if !strings.Contains(screen.frames[1], "> 8082 2 node") || strings.Contains(screen.frames[1], "> 8080") {
		t.Fatalf("down\n%s", screen.frames[1])
	}
	if !strings.Contains(screen.frames[2], "> 8080 1 python") {
		t.Fatalf("up\n%s", screen.frames[2])
	}
}

func TestDDFreesTheSelectedPort(t *testing.T) {
	rows := twoPorts()
	var freed []int
	m := &model{
		load: func() ([]occupant, error) { return rows, nil },
		release: func(port int) ([]proc, error) {
			freed = append(freed, port)
			var next []occupant
			for _, row := range rows {
				if row.port != port {
					next = append(next, row)
				}
			}
			rows = next
			return []proc{{pid: 2, name: "node"}}, nil
		},
	}
	screen := &fakeScreen{width: 80, height: 24, keys: []string{"j", "d", "d", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(freed) != 1 || freed[0] != 8082 {
		t.Fatalf("freed %v", freed)
	}
	last := screen.frames[len(screen.frames)-1]
	if strings.Contains(last, "> 8082") || strings.Contains(last, "  8082") {
		t.Fatalf("row remained\n%s", last)
	}
	if !strings.Contains(last, "8082 is free") || !strings.Contains(last, "> 8080") {
		t.Fatalf("last\n%s", last)
	}
}

func TestDThenMotionDoesNotFree(t *testing.T) {
	rows := twoPorts()
	var freed []int
	m := &model{
		load: func() ([]occupant, error) { return rows, nil },
		release: func(port int) ([]proc, error) {
			freed = append(freed, port)
			return nil, nil
		},
	}
	screen := &fakeScreen{width: 40, height: 20, keys: []string{"d", "j", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(freed) != 0 || m.cursor != 1 || m.armed {
		t.Fatalf("cursor %d armed %v freed %v", m.cursor, m.armed, freed)
	}
	if !strings.Contains(screen.frames[1], "\x1b[90md\x1b[0m") {
		t.Fatalf("arm frame\n%s", screen.frames[1])
	}
}

func TestEscCancelsFree(t *testing.T) {
	rows := twoPorts()
	var freed []int
	m := &model{
		load: func() ([]occupant, error) { return rows, nil },
		release: func(port int) ([]proc, error) {
			freed = append(freed, port)
			return nil, nil
		},
	}
	screen := &fakeScreen{width: 40, height: 20, keys: []string{"d", "esc", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(freed) != 0 || m.armed || m.status != "" {
		t.Fatalf("armed %v status %q freed %v", m.armed, m.status, freed)
	}
}

func TestDDOnAnEmptyListDoesNotSignal(t *testing.T) {
	var freed []int
	m := &model{
		load: func() ([]occupant, error) { return nil, nil },
		release: func(port int) ([]proc, error) {
			freed = append(freed, port)
			return nil, nil
		},
	}
	screen := &fakeScreen{width: 40, height: 20, keys: []string{"d", "d", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(freed) != 0 || m.status != "no occupied external ports" {
		t.Fatalf("status %q freed %v", m.status, freed)
	}
	if !strings.Contains(screen.frames[0], "no occupied external ports") {
		t.Fatalf("empty\n%s", screen.frames[0])
	}
}

func TestDDKeepsTheRowWhenThePortStaysTaken(t *testing.T) {
	rows := twoPorts()[:1]
	m := &model{
		load: func() ([]occupant, error) { return rows, nil },
		release: func(int) ([]proc, error) {
			return nil, usageError("port 8080 is still in use")
		},
	}
	screen := &fakeScreen{width: 80, height: 24, keys: []string{"d", "d", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "> 8080") || !strings.Contains(last, "still in use") {
		t.Fatalf("last\n%s", last)
	}
}
