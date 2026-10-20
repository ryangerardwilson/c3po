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

func TestPickBrowseUsesTheSecondPreset(t *testing.T) {
	screen := &fakeScreen{keys: []string{"j", "enter", "enter"}}
	choice, err := chooseOn(screen, presets())
	if err != nil {
		t.Fatal(err)
	}
	if choice.Preset != "2" || choice.Action != "browse" || choice.Paths != "" {
		t.Fatalf("%#v", choice)
	}
	if !strings.Contains(screen.frames[0], "gdrive preset") || !strings.Contains(screen.frames[0], "> 1  a@b.c") {
		t.Fatalf("first frame\n%s", screen.frames[0])
	}
}

func TestPickUploadKeepsTypedPaths(t *testing.T) {
	screen := &fakeScreen{keys: []string{"enter", "j", "enter", "enter", "a", ".", "t", "x", "t", "enter"}}
	choice, err := chooseOn(screen, presets())
	if err != nil {
		t.Fatal(err)
	}
	if choice.Preset != "1" || choice.Action != "upload" || choice.Paths != "a.txt" {
		t.Fatalf("%#v", choice)
	}
}

func TestEmptyUploadPathsStayOnThePrompt(t *testing.T) {
	screen := &fakeScreen{keys: []string{"enter", "j", "enter", "enter", "esc", "h", "q"}}
	_, err := chooseOn(screen, presets())
	if !errors.Is(err, ErrQuit) {
		t.Fatal(err)
	}
	joined := strings.Join(screen.frames, "\n---\n")
	if !strings.Contains(joined, "enter one or more local paths") {
		t.Fatal(joined)
	}
	last := screen.frames[len(screen.frames)-1]
	if !strings.Contains(last, "gdrive preset") {
		t.Fatalf("did not return to presets\n%s", last)
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
