//go:build linux

package tui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

// terminal is the input and output side of the controlling terminal.
type terminal struct {
	in, out *os.File
	opened  *os.File
	saved   *syscall.Termios
}

// openTerminal uses stdin and stdout when they are terminals, and
// /dev/tty for whichever side is not.
func openTerminal() (*terminal, error) {
	t := &terminal{in: os.Stdin, out: os.Stdout}
	if isTerminal(t.in) && isTerminal(t.out) {
		return t, nil
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("could not open a new TTY: %w", err)
	}
	t.opened = tty
	if !isTerminal(t.in) {
		t.in = tty
	}
	if !isTerminal(t.out) {
		t.out = tty
	}
	return t, nil
}

func (t *terminal) close() {
	if t.opened != nil {
		_ = t.opened.Close()
	}
}

func isTerminal(f *os.File) bool {
	var term syscall.Termios
	return ioctl(int(f.Fd()), syscall.TCGETS, unsafe.Pointer(&term)) == nil
}

// raw puts the input side in raw mode, saving the first state seen.
func (t *terminal) raw() error {
	fd := int(t.in.Fd())
	var old syscall.Termios
	if err := ioctl(fd, syscall.TCGETS, unsafe.Pointer(&old)); err != nil {
		return err
	}
	if t.saved == nil {
		t.saved = &old
	}
	raw := *t.saved
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	return ioctl(fd, syscall.TCSETS, unsafe.Pointer(&raw))
}

// restore puts back the terminal state saved by raw.
func (t *terminal) restore() {
	if t.saved != nil {
		_ = ioctl(int(t.in.Fd()), syscall.TCSETS, unsafe.Pointer(t.saved))
	}
}

func ioctl(fd int, req uint, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// size is the output side's size, or 80x24 when it cannot be read.
func (t *terminal) size() WindowSizeMsg {
	var ws struct {
		Row, Col, X, Y uint16
	}
	err := ioctl(int(t.out.Fd()), syscall.TIOCGWINSZ, unsafe.Pointer(&ws))
	if err != nil || ws.Row == 0 || ws.Col == 0 {
		return WindowSizeMsg{Width: 80, Height: 24}
	}
	return WindowSizeMsg{Width: int(ws.Col), Height: int(ws.Row)}
}

// readKeys waits up to wait for input and parses what is available.
func (t *terminal) readKeys(buf []byte, wait time.Duration) ([]KeyMsg, error) {
	fd := int(t.in.Fd())
	if !pollReady(fd, wait) {
		return nil, nil
	}
	n, err := syscall.Read(fd, buf)
	if err == syscall.EINTR || err == syscall.EAGAIN {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, os.ErrClosed
	}
	return ParseKeys(buf[:n]), nil
}

func pollReady(fd int, wait time.Duration) bool {
	pfd := struct {
		fd      int32
		events  int16
		revents int16
	}{fd: int32(fd), events: 1}
	n, _, errno := syscall.Syscall(syscall.SYS_POLL, uintptr(unsafe.Pointer(&pfd)), 1, uintptr(wait.Milliseconds()))
	return errno == 0 && n > 0 && pfd.revents&1 != 0
}

// watchSignals turns resize into WindowSizeMsg, SIGINT into an
// interrupt, and SIGTERM into a quit. It returns the stop function.
func (p *Program) watchSignals() func() {
	sig := make(chan os.Signal, 4)
	signal.Notify(sig, syscall.SIGWINCH, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for s := range sig {
			p.send(signalMsg(s, p.term))
		}
	}()
	return func() {
		signal.Stop(sig)
		close(sig)
	}
}

func signalMsg(s os.Signal, t *terminal) Msg {
	switch s {
	case syscall.SIGWINCH:
		return t.size()
	case syscall.SIGINT:
		return interruptMsg{}
	}
	return QuitMsg{}
}
