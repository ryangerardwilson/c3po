//go:build linux

package cyclotree

import (
	"errors"
	"os"
	"syscall"
	"time"
	"unsafe"
)

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

func termRows(fd int) int {
	var ws struct {
		Row, Col, X, Y uint16
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x5413, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 || ws.Row == 0 {
		return 24
	}
	return int(ws.Row)
}

func readKey(f *os.File) (rune, error) {
	b, err := readByte(f)
	if err != nil {
		return 0, err
	}
	if b != 0x1b {
		return decodeSeq(b, ""), nil
	}
	rest, err := readArrow(f)
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		return 0, err
	}
	return decodeSeq(0x1b, rest), nil
}

func readByte(f *os.File) (byte, error) {
	var buf [1]byte
	_, err := f.Read(buf[:])
	return buf[0], err
}

func readArrow(f *os.File) (string, error) {
	_ = f.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
	defer f.SetReadDeadline(time.Time{})
	var buf [2]byte
	for i := 0; i < len(buf); i++ {
		_, err := f.Read(buf[i : i+1])
		if err != nil {
			return string(buf[:i]), err
		}
		if i == 0 && buf[0] != '[' && buf[0] != 'O' {
			return string(buf[:1]), nil
		}
	}
	return string(buf[:]), nil
}

func nextKey(f *os.File) (rune, bool) {
	if f == nil || !pollReady(int(f.Fd()), 0) {
		return 0, false
	}
	k, err := readKey(f)
	if err != nil {
		return 0, false
	}
	return k, true
}

func readKeyWait(f *os.File, wait time.Duration) (rune, bool, error) {
	if f == nil || !pollReady(int(f.Fd()), wait) {
		return 0, false, nil
	}
	k, err := readKey(f)
	if err != nil {
		return 0, false, err
	}
	return k, true, nil
}

func pollReady(fd int, wait time.Duration) bool {
	ms := wait.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	pfd := struct {
		fd      int32
		events  int16
		revents int16
	}{fd: int32(fd), events: 1}
	n, _, errno := syscall.Syscall(syscall.SYS_POLL, uintptr(unsafe.Pointer(&pfd)), 1, uintptr(ms))
	return errno == 0 && n > 0 && pfd.revents&1 != 0
}
