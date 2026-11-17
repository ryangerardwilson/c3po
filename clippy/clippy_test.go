package clippy

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

type fakeBoard struct {
	text   string
	err    error
	called bool
}

func (f *fakeBoard) Copy(text string, stderr io.Writer) error {
	f.called = true
	f.text = text
	return f.err
}

func TestStdinIsCopiedAndNotPrinted(t *testing.T) {
	board := &fakeBoard{}
	var out, errb bytes.Buffer
	text := "line one\nline two\n"
	code := run(nil, &out, &errb, strings.NewReader(text), board)
	if code != 0 || out.String() != "copied\n" || errb.Len() != 0 {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if !board.called || board.text != text {
		t.Fatalf("clipboard %q called %v", board.text, board.called)
	}
	if strings.Contains(out.String()+errb.String(), "line one") {
		t.Fatal("printed the text")
	}
}

func TestArgumentsAreOneString(t *testing.T) {
	board := &fakeBoard{}
	var out, errb bytes.Buffer
	code := run([]string{"hello", "world"}, &out, &errb, strings.NewReader("ignored\n"), board)
	if code != 0 || board.text != "hello world" {
		t.Fatalf("code %d clipboard %q", code, board.text)
	}
}

func TestNewlineInsideAnArgumentIsKept(t *testing.T) {
	board := &fakeBoard{}
	var out, errb bytes.Buffer
	code := run([]string{"hello\nworld"}, &out, &errb, strings.NewReader(""), board)
	if code != 0 || board.text != "hello\nworld" {
		t.Fatalf("code %d clipboard %q", code, board.text)
	}
}

func TestEmptyTextIsRefused(t *testing.T) {
	board := &fakeBoard{}
	var out, errb bytes.Buffer
	code := run(nil, &out, &errb, strings.NewReader(""), board)
	if code != 2 || board.called || out.Len() != 0 {
		t.Fatalf("code %d called %v out %q", code, board.called, out.String())
	}
	if !strings.Contains(errb.String(), "usage: c3po clippy") {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestMissingWLCopyDoesNotPrintTheText(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var out, errb bytes.Buffer
	code := run(nil, &out, &errb, strings.NewReader("secret line\n"), WLCopy{})
	if code != 1 || errb.String() != "wl-copy is required.\n" || out.Len() != 0 {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if strings.Contains(errb.String(), "secret") {
		t.Fatal("printed the text")
	}
}

func TestCopyFailureDoesNotPrintTheText(t *testing.T) {
	board := &fakeBoard{err: errors.New("Unable to copy to the clipboard.")}
	var out, errb bytes.Buffer
	code := run([]string{"secret line"}, &out, &errb, strings.NewReader(""), board)
	if code != 1 || errb.String() != "Unable to copy to the clipboard.\n" || out.Len() != 0 {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if strings.Contains(errb.String(), "secret") {
		t.Fatal("printed the text")
	}
}

func TestReadErrorDoesNotPrintTheText(t *testing.T) {
	board := &fakeBoard{}
	var out, errb bytes.Buffer
	code := run(nil, &out, &errb, failReader{}, board)
	if code != 1 || board.called || out.Len() != 0 {
		t.Fatalf("code %d called %v out %q err %q", code, board.called, out.String(), errb.String())
	}
	if !strings.Contains(errb.String(), "clippy: broken pipe") || strings.Contains(errb.String(), "secret") {
		t.Fatalf("stderr %q", errb.String())
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, errors.New("broken pipe")
}

func TestTerminalWithNoTextPrintsHelp(t *testing.T) {
	board := &fakeBoard{}
	file, err := os.Open("/dev/null")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	var out, errb bytes.Buffer
	code := run(nil, &out, &errb, file, board)
	if code != 0 || board.called || !strings.Contains(out.String(), "c3po clippy") {
		t.Fatalf("code %d called %v out %q", code, board.called, out.String())
	}
}

func TestHelpAndVersionDoNotCopy(t *testing.T) {
	board := &fakeBoard{}
	var out, errb bytes.Buffer
	code := run([]string{"help"}, &out, &errb, strings.NewReader("secret"), board)
	if code != 0 || board.called || !strings.Contains(out.String(), "c3po clippy") {
		t.Fatalf("help code %d called %v out %q", code, board.called, out.String())
	}
	out.Reset()
	code = run([]string{"version"}, &out, &errb, strings.NewReader("secret"), board)
	if code != 0 || board.called || !strings.HasPrefix(out.String(), "c3po ") {
		t.Fatalf("version code %d called %v out %q", code, board.called, out.String())
	}
}
