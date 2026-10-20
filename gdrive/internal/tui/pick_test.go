package tui

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeScreen struct {
	keys   []string
	frames []string
}

func (f *fakeScreen) size() (int, int) { return 80, 24 }

func (f *fakeScreen) paint(frame string) { f.frames = append(f.frames, frame) }

func (f *fakeScreen) readKey() (string, error) {
	if len(f.keys) == 0 {
		return "", io.EOF
	}
	key := f.keys[0]
	f.keys = f.keys[1:]
	return key, nil
}

func presets() []Item {
	return []Item{{ID: "1", Label: "1  a@b.c"}, {ID: "2", Label: "2  d@e.f"}}
}

func TestPickUsesTheSecondPreset(t *testing.T) {
	screen := &fakeScreen{keys: []string{"j", "enter"}}
	preset, err := chooseOn(screen, presets())
	if err != nil {
		t.Fatal(err)
	}
	if preset != "2" {
		t.Fatalf("%q", preset)
	}
	if !strings.Contains(screen.frames[0], "gdrive preset") || !strings.Contains(screen.frames[0], "> 1  a@b.c") {
		t.Fatalf("first frame\n%s", screen.frames[0])
	}
	if strings.Contains(strings.Join(screen.frames, "\n"), "upload") {
		t.Fatal(screen.frames)
	}
}

func TestQuitOnThePresetList(t *testing.T) {
	_, err := chooseOn(&fakeScreen{keys: []string{"q"}}, presets())
	if !errors.Is(err, ErrQuit) {
		t.Fatal(err)
	}
}

func TestChooseRejectsAnEmptyList(t *testing.T) {
	_, err := chooseOn(&fakeScreen{}, nil)
	if err == nil || errors.Is(err, ErrQuit) {
		t.Fatal(err)
	}
}
