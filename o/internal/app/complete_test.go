package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func endAt(line string) int {
	return len([]rune(line))
}

func writeFile(t *testing.T, dir, name string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), mode); err != nil {
		t.Fatal(err)
	}
}

func TestUniqueFileCompletesWithASpace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	got, at, matches := completeShell(root, "!ls no", endAt("!ls no"), "", "")
	if got != "!ls notes.txt " || at != endAt(got) || matches != nil {
		t.Fatalf("got %q at %d matches %v", got, at, matches)
	}
}

func TestUniqueDirectoryGainsASlash(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, _, matches := completeShell(root, "!ls no", endAt("!ls no"), "", "")
	if got != "!ls notes/" || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestDirectorySlashContinuesInside(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, "notes")
	if err := os.Mkdir(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, notes, "a.txt", 0o644)
	got, _, _ := completeShell(root, "!ls no", endAt("!ls no"), "", "")
	if got != "!ls notes/" {
		t.Fatalf("first %q", got)
	}
	got, _, matches := completeShell(root, got, endAt(got), "", "")
	if got != "!ls notes/a.txt " || matches != nil {
		t.Fatalf("second %q matches %v", got, matches)
	}
}

func TestSymlinkDirectoryGainsASlash(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "notes")); err != nil {
		t.Fatal(err)
	}
	got, _, matches := completeShell(root, "!ls no", endAt("!ls no"), "", "")
	if got != "!ls notes/" || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestSeveralMatchesFillTheCommonPrefix(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	writeFile(t, root, "notice.txt", 0o644)
	got, _, matches := completeShell(root, "!ls n", endAt("!ls n"), "", "")
	if got != "!ls not" || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestUnchangedWordListsTheMatches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	writeFile(t, root, "notice.txt", 0o644)
	got, _, matches := completeShell(root, "!ls not", endAt("!ls not"), "", "")
	want := []string{"notes.txt", "notice.txt"}
	if got != "!ls not" || !slices.Equal(matches, want) {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestUnchangedDirectoryListsASlash(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"notes", "notice"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, _, matches := completeShell(root, "!ls not", endAt("!ls not"), "", "")
	want := []string{"notes/", "notice/"}
	if got != "!ls not" || !slices.Equal(matches, want) {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestDotFileStaysHidden(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".secret", 0o644)
	writeFile(t, root, "notes.txt", 0o644)
	got, _, _ := completeShell(root, "!ls n", endAt("!ls n"), "", "")
	if got != "!ls notes.txt " {
		t.Fatalf("got %q", got)
	}
}

func TestDotFileCompletesWhenAsked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".secret", 0o644)
	got, _, _ := completeShell(root, "!ls .se", endAt("!ls .se"), "", "")
	if got != "!ls .secret " {
		t.Fatalf("got %q", got)
	}
}

func TestFirstWordCompletesACommand(t *testing.T) {
	bin := t.TempDir()
	writeFile(t, bin, "touch", 0o755)
	got, _, matches := completeShell(t.TempDir(), "!tou", endAt("!tou"), bin, "")
	if got != "!touch " || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestCommandSkipsDirectoriesAndPlainFiles(t *testing.T) {
	bin := t.TempDir()
	writeFile(t, bin, "touch", 0o755)
	writeFile(t, bin, "topic", 0o644)
	if err := os.Mkdir(filepath.Join(bin, "tower"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, _, matches := completeShell(t.TempDir(), "!to", endAt("!to"), bin, "")
	if got != "!touch " || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestListedFileIsNotACommand(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "script", 0o755)
	got, _, _ := completeShell(root, "!sc", endAt("!sc"), "", "")
	if got != "!sc" {
		t.Fatalf("got %q", got)
	}
}

func TestPathCommandCompletesInTheListedDirectory(t *testing.T) {
	listed := t.TempDir()
	other := t.TempDir()
	writeFile(t, listed, "script", 0o755)
	writeFile(t, other, "score", 0o755)
	t.Chdir(other)
	got, _, _ := completeShell(listed, "!./sc", endAt("!./sc"), "", "")
	if got != "!./script " {
		t.Fatalf("got %q", got)
	}
}

func TestEmptyPathComponentIsNotDot(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "tool", 0o755)
	t.Chdir(cwd)
	got, _, _ := completeShell(t.TempDir(), "!to", endAt("!to"), ":", "")
	if got != "!to" {
		t.Fatalf("got %q", got)
	}
}

func TestSpaceInANameIsEscaped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "my notes.txt", 0o644)
	got, _, _ := completeShell(root, "!ls my", endAt("!ls my"), "", "")
	if got != `!ls my\ notes.txt ` {
		t.Fatalf("got %q", got)
	}
}

func TestQuotedFileClosesTheQuote(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "my notes.txt", 0o644)
	line := `!ls "my`
	got, _, _ := completeShell(root, line, endAt(line), "", "")
	if got != `!ls "my notes.txt" ` {
		t.Fatalf("got %q", got)
	}
}

func TestHomePrefixStaysTilde(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, "notes.txt", 0o644)
	got, _, _ := completeShell(t.TempDir(), "!ls ~/no", endAt("!ls ~/no"), "", home)
	if got != "!ls ~/notes.txt " || strings.Contains(got, home) {
		t.Fatalf("got %q", got)
	}
}

func TestUserTildeDoesNotComplete(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, "notes.txt", 0o644)
	line := "!ls ~nobody/n"
	got, _, matches := completeShell(t.TempDir(), line, endAt(line), "", home)
	if got != line || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestExistingSpaceIsKept(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	line := "!ls notes.txt "
	at := endAt("!ls notes.txt")
	got, _, matches := completeShell(root, line, at, "", "")
	if got != line || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestCursorTailIsReplaced(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	writeFile(t, root, "notice.txt", 0o644)
	got, _, _ := completeShell(root, "!ls notes", endAt("!ls no"), "", "")
	if got != "!ls not" {
		t.Fatalf("got %q", got)
	}
}

func TestTabOutsideAShellLineDoesNothing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	got, _, matches := completeShell(root, "mkdir no", endAt("mkdir no"), "", "")
	if got != "mkdir no" || matches != nil {
		t.Fatalf("got %q matches %v", got, matches)
	}
}

func TestTabOnAFileCommandDoesNothing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	m := model{
		currentDir: root,
		commanding: true,
		input:      "mkdir no",
		inputAt:    endAt("mkdir no"),
		width:      80,
		height:     8,
	}
	next, cmd := m.Update(tui.KeyMsg{Type: tui.KeyTab})
	got := next.(model)
	if cmd != nil || got.input != "mkdir no" || got.matches != nil {
		t.Fatalf("input %q matches %v cmd %v", got.input, got.matches, cmd)
	}
}

func TestTabListsMatchesOnThePromptLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", 0o644)
	writeFile(t, root, "notice.txt", 0o644)
	m := model{
		currentDir: root,
		commanding: true,
		input:      "!ls n",
		inputAt:    endAt("!ls n"),
		width:      80,
		height:     8,
	}
	next, cmd := m.Update(tui.KeyMsg{Type: tui.KeyTab})
	got := next.(model)
	if cmd != nil || got.input != "!ls not" || got.matches != nil {
		t.Fatalf("input %q matches %v cmd %v", got.input, got.matches, cmd)
	}
	if strings.Contains(strings.Split(got.View(), "\n")[1], "notes.txt") {
		t.Fatalf("listed too soon: %q", strings.Split(got.View(), "\n")[1])
	}
	next, cmd = got.Update(tui.KeyMsg{Type: tui.KeyTab})
	got = next.(model)
	prompt := strings.Split(got.View(), "\n")[1]
	if cmd != nil || got.input != "!ls not" || !strings.Contains(prompt, "notes.txt") || !strings.Contains(prompt, "notice.txt") || strings.Contains(prompt, ":!") {
		t.Fatalf("input %q prompt %q cmd %v", got.input, prompt, cmd)
	}
	next, _ = got.Update(tui.KeyMsg{Type: tui.KeyRunes, Runes: []rune{'x'}})
	cleared := next.(model)
	if len(cleared.matches) != 0 || strings.Contains(strings.Split(cleared.View(), "\n")[1], "notes.txt") {
		t.Fatalf("matches stayed: %v %q", cleared.matches, strings.Split(cleared.View(), "\n")[1])
	}
}

func TestHelpNamesTab(t *testing.T) {
	help := model{showHelp: true}.renderHelp()
	if !strings.Contains(help, "tab                complete the :! word") {
		t.Fatalf("help missing tab:\n%s", help)
	}
}
