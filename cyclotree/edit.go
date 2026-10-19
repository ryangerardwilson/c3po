package cyclotree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type vimLaunch struct {
	dir  string
	args []string
}

func cursorFile(s *screen) (string, int, bool) {
	if s == nil {
		return "", 0, false
	}
	s.layout()
	if s.cursor < 0 || s.cursor >= len(s.rows) {
		return "", 0, false
	}
	n := s.rows[s.cursor].n
	if n == nil || n.path == "" || n.line < 1 {
		return "", 0, false
	}
	return n.path, n.line, true
}

func vimCall(dir string, s *screen) (vimLaunch, bool) {
	rel, line, ok := cursorFile(s)
	if !ok {
		return vimLaunch{}, false
	}
	return vimLaunch{
		dir:  dir,
		args: []string{fmt.Sprintf("+%d", line), filepath.FromSlash(rel)},
	}, true
}

func editRow(dir string, s *screen, in, out *os.File, restore *func()) error {
	call, ok := vimCall(dir, s)
	if !ok {
		return nil
	}
	leaveScreen(out, restore)
	err := runVim(call, in, out)
	return enterScreen(in, out, restore, err)
}

func leaveScreen(out *os.File, restore *func()) {
	fmt.Fprint(out, "\x1b[0m\x1b[?25h\x1b[?1049l")
	if restore == nil || *restore == nil {
		return
	}
	(*restore)()
	*restore = nil
}

func runVim(call vimLaunch, in, out *os.File) error {
	cmd := exec.Command("vim", call.args...)
	cmd.Dir = call.dir
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

func enterScreen(in, out *os.File, restore *func(), runErr error) error {
	next, err := enterRaw(int(in.Fd()))
	if next == nil {
		next = func() {}
	}
	if restore != nil {
		*restore = next
	}
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	if err != nil {
		return err
	}
	var start *exec.Error
	if errors.As(runErr, &start) {
		return runErr
	}
	return nil
}
