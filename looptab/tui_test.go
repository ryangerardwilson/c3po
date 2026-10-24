package looptab

import (
	"errors"
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

func twoJobs() []job {
	return []job{
		{Index: 0, ID: "abcd1234", Schedule: "daily 5am", Kind: "grok", Action: `@grok "Check mail"`},
		{Index: 1, ID: "00abcdef", Schedule: "hourly at 15", Kind: "command", Action: "c3po gmail spam clean"},
	}
}

func TestJKMovesAcrossJobs(t *testing.T) {
	rows := twoJobs()
	m := &model{load: func() ([]job, error) { return rows, nil }}
	screen := &fakeScreen{width: 80, height: 24, keys: []string{"j", "k", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(screen.frames[0], "j/k move") || !strings.Contains(screen.frames[0], "> daily 5am  grok") {
		t.Fatalf("first\n%s", screen.frames[0])
	}
	if !strings.Contains(screen.frames[1], "> hourly at 15  command") {
		t.Fatalf("down\n%s", screen.frames[1])
	}
	if !strings.Contains(screen.frames[2], "> daily 5am  grok") {
		t.Fatalf("up\n%s", screen.frames[2])
	}
}

func TestEnterRunsTheSelectedIndex(t *testing.T) {
	rows := twoJobs()
	var ran []int
	m := &model{
		load: func() ([]job, error) { return rows, nil },
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := &fakeScreen{width: 80, height: 24, keys: []string{"j", "enter", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != 1 {
		t.Fatalf("ran %v", ran)
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "ran 00abcdef") {
		t.Fatalf("last\n%s", last)
	}
}

func TestQuestionMarkBlocksTheRun(t *testing.T) {
	rows := twoJobs()
	var ran []int
	m := &model{
		load: func() ([]job, error) { return rows, nil },
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := &fakeScreen{width: 80, height: 24, keys: []string{"?", "j", "enter", "esc", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 || m.cursor != 0 {
		t.Fatalf("cursor %d ran %v", m.cursor, ran)
	}
	if strings.Contains(screen.frames[0], "j/k move") || !strings.Contains(screen.frames[1], "enter") || !strings.Contains(screen.frames[1], "quit") {
		t.Fatalf("frames\n%s\n%s", screen.frames[0], screen.frames[1])
	}
	if !strings.Contains(screen.frames[len(screen.frames)-1], "> daily 5am") || strings.Contains(screen.frames[len(screen.frames)-1], "j/k move") {
		t.Fatalf("restored\n%s", screen.frames[len(screen.frames)-1])
	}
}

func TestEnterOnAnEmptyListDoesNotRun(t *testing.T) {
	var ran []int
	m := &model{
		load: func() ([]job, error) { return nil, nil },
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := &fakeScreen{width: 40, height: 20, keys: []string{"enter", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 || m.status != "no jobs" || !strings.Contains(screen.frames[1], "no jobs") {
		t.Fatalf("status %q ran %v\n%s", m.status, ran, screen.frames[1])
	}
}

func TestEnterWhenLooptabIsMissingDoesNotRun(t *testing.T) {
	var ran []int
	m := &model{
		load: func() ([]job, error) { return nil, errors.New("looptab is not on PATH") },
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := &fakeScreen{width: 40, height: 20, keys: []string{"enter", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 || !strings.Contains(screen.frames[0], "looptab is not on PATH") {
		t.Fatalf("ran %v\n%s", ran, screen.frames[0])
	}
}

func TestAFailedRunStaysOnTheList(t *testing.T) {
	rows := twoJobs()
	m := &model{
		load: func() ([]job, error) { return rows, nil },
		run:  func(job) error { return errors.New("exit status 1") },
	}
	screen := &fakeScreen{width: 80, height: 24, keys: []string{"enter", "q"}}
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "abcd1234 failed") || !strings.Contains(last, "> daily 5am") {
		t.Fatalf("last\n%s", last)
	}
}
