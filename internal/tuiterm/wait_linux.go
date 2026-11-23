//go:build linux

// Package tuiterm waits on a screen's terminal. It is the one copy of the
// poll the hand-rolled c3po screens use to read a key with a deadline.
package tuiterm

import (
	"time"

	"golang.org/x/sys/unix"
)

// WaitReadable waits up to wait for fd to have input and reports whether it
// does. A wait of zero or less only looks.
func WaitReadable(fd int, wait time.Duration) (bool, error) {
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
