//go:build linux

package tui

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

const escWait = 30 * time.Millisecond

func enterRaw(fd int) (func(), error) {
	var old syscall.Termios
	if err := ioctl(fd, syscall.TCGETS, &old); err != nil {
		return nil, err
	}
	raw := old
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if err := ioctl(fd, syscall.TCSETS, &raw); err != nil {
		return nil, err
	}
	return func() { _ = ioctl(fd, syscall.TCSETS, &old) }, nil
}

func ioctl(fd int, req uint, term *syscall.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(unsafe.Pointer(term)))
	if errno != 0 {
		return errno
	}
	return nil
}

// readKey waits up to wait for a key. ok is false when none arrived.
func readKey(f *os.File, wait time.Duration) (string, bool, error) {
	fd := int(f.Fd())
	if !pollReady(fd, wait) {
		return "", false, nil
	}
	b, err := readByte(fd)
	if err != nil {
		return "", false, err
	}
	if b != 0x1b {
		return decode(b, ""), true, nil
	}
	return decode(b, escapeTail(fd)), true, nil
}

// escapeTail reads the rest of an arrow sequence. A lone escape has
// nothing after it within escWait.
func escapeTail(fd int) string {
	var tail []byte
	for len(tail) < 2 && pollReady(fd, escWait) {
		b, err := readByte(fd)
		if err != nil {
			break
		}
		tail = append(tail, b)
		if len(tail) == 1 && b != '[' && b != 'O' {
			break
		}
	}
	return string(tail)
}

func readByte(fd int) (byte, error) {
	var buf [1]byte
	for {
		n, err := syscall.Read(fd, buf[:])
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, os.ErrClosed
		}
		return buf[0], nil
	}
}

func pollReady(fd int, wait time.Duration) bool {
	ms := max(wait.Milliseconds(), 0)
	pfd := struct {
		fd      int32
		events  int16
		revents int16
	}{fd: int32(fd), events: 1}
	n, _, errno := syscall.Syscall(syscall.SYS_POLL, uintptr(unsafe.Pointer(&pfd)), 1, uintptr(ms))
	return errno == 0 && n > 0 && pfd.revents&1 != 0
}
