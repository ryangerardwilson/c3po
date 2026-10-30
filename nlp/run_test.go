package nlp

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/nlp/internal/app"
)

func TestHelpAndVersionDoNotStart(t *testing.T) {
	restore := swapRun(t, func(app.Config, *tuisession.Session) int {
		t.Fatal("runApp should not be called")
		return 1
	}, func(io.Writer) bool { return true })
	defer restore()

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit = %d", code)
	}
	if strings.Contains(stdout.String(), "bookmark") {
		t.Fatalf("help still documents bookmarks: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "neuro-linguistic programming") {
		t.Fatalf("help missing objective: %s", stdout.String())
	}

	stdout.Reset()
	if code := Run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("version exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "c3po ") {
		t.Fatalf("version %q", stdout.String())
	}
}

func TestPipeDoesNotStart(t *testing.T) {
	restore := swapRun(t, func(app.Config, *tuisession.Session) int {
		t.Fatal("runApp should not be called")
		return 1
	}, func(io.Writer) bool { return false })
	defer restore()

	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("pipe exit = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "c3po tui nlp") {
		t.Fatalf("pipe help %s", stdout.String())
	}
}

func TestBookmarksCommandIsRejected(t *testing.T) {
	restore := swapRun(t, func(app.Config, *tuisession.Session) int {
		t.Fatal("runApp should not be called")
		return 1
	}, func(io.Writer) bool { return true })
	defer restore()

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"bookmarks", "list"}, &stdout, &stderr); code != 2 {
		t.Fatalf("bookmarks exit = %d", code)
	}
	if strings.Contains(stdout.String(), "bookmark") || strings.Contains(stderr.String(), "bookmark") {
		t.Fatalf("output mentioned bookmarks: %s %s", stdout.String(), stderr.String())
	}
}

func TestTerminalStartsAndResumeRestores(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SQLX_ROOT", filepath.Join(home, "sqlx"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	var id string
	var seen int
	restore := swapRun(t, func(cfg app.Config, sess *tuisession.Session) int {
		seen++
		if len(cfg.Courses) == 0 {
			t.Fatal("expected courses")
		}
		if seen == 1 {
			id = sess.ID()
			if err := sess.Place("anchor", "parts 0 1"); err != nil {
				t.Fatal(err)
			}
			return 0
		}
		body, ok := sess.Restored()
		if !ok || body.Screen != "nlp" || body.Path != "anchor" || body.Cursor != "parts 0 1" {
			t.Fatalf("restored ok=%v %+v", ok, body)
		}
		if strings.Contains(body.Cursor+body.Path, "#") {
			t.Fatal("session stored markdown")
		}
		return 0
	}, func(io.Writer) bool { return true })
	defer restore()

	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("open exit = %d stderr=%s", code, stderr.String())
	}
	if code := Run([]string{"--resume", id}, &stdout, &stderr); code != 0 {
		t.Fatalf("resume exit = %d stderr=%s", code, stderr.String())
	}
	if seen != 2 || id == "" {
		t.Fatalf("seen %d id %q", seen, id)
	}
}

func swapRun(t *testing.T, run func(app.Config, *tuisession.Session) int, term func(io.Writer) bool) func() {
	t.Helper()
	prevRun, prevTerm := runApp, termOK
	runApp, termOK = run, term
	return func() {
		runApp, termOK = prevRun, prevTerm
	}
}
