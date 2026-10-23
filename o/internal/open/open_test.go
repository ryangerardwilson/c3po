package open

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorkbookCommandIsTheXLScreen(t *testing.T) {
	cmd := WorkbookCommand("/tmp/budget.xlsx")
	args := cmd.Args
	if len(args) < 4 || args[len(args)-3] != "tui" || args[len(args)-2] != "xl" || args[len(args)-1] != "/tmp/budget.xlsx" {
		t.Fatalf("args %q", args)
	}
}

func TestOpenWorkbookSkipsTheDesktopOpener(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "Budget.XLSX")
	other := filepath.Join(dir, "scan.pdf")
	if err := os.WriteFile(book, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got []string
	var desk []string
	prevCmd := WorkbookCommand
	prevDetached := detached
	WorkbookCommand = func(path string) *exec.Cmd {
		got = append(got, path)
		return exec.Command("/bin/true")
	}
	detached = func(path string) error {
		desk = append(desk, path)
		return nil
	}
	t.Cleanup(func() {
		WorkbookCommand = prevCmd
		detached = prevDetached
	})

	if code := OpenFiles([]string{book}); code != 0 {
		t.Fatalf("workbook exit %d", code)
	}
	if len(desk) != 0 {
		t.Fatalf("workbook reached the desktop opener: %q", desk)
	}
	if len(got) != 1 || got[0] != book {
		t.Fatalf("opened %q", got)
	}

	got, desk = nil, nil
	if code := OpenFiles([]string{book, other}); code != 0 {
		t.Fatalf("mixed exit %d", code)
	}
	if len(desk) != 1 || desk[0] != other {
		t.Fatalf("desktop opener got %q", desk)
	}
	if len(got) != 1 || got[0] != book {
		t.Fatalf("mixed workbook %q", got)
	}
}
