//go:build linux

package tui

import (
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
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

func termSize(fd int) (int, int) {
	var ws struct {
		Row, Col, X, Y uint16
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 || ws.Row == 0 || ws.Col == 0 {
		return 80, 24
	}
	return int(ws.Col), int(ws.Row)
}

func waitReadable(fd int, wait time.Duration) (bool, error) {
	if wait < 0 {
		wait = 0
	}
	deadline := time.Now().Add(wait)
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		remain := time.Until(deadline)
		ms := int(remain / time.Millisecond)
		if remain > 0 && ms == 0 {
			ms = 1
		}
		if ms < 0 {
			ms = 0
		}
		n, err := unix.Poll(fds, ms)
		if err == unix.EINTR {
			if ms == 0 {
				return false, nil
			}
			continue
		}
		if err != nil {
			return false, err
		}
		return n > 0, nil
	}
}
