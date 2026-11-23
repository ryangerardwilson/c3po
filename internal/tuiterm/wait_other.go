//go:build !linux

// Package tuiterm waits on a screen's terminal. It is the one copy of the
// poll the hand-rolled c3po screens use to read a key with a deadline.
package tuiterm

import (
	"errors"
	"time"
)

// WaitReadable needs raw mode, which this platform does not offer.
func WaitReadable(fd int, wait time.Duration) (bool, error) {
	return false, errors.New("raw mode is unavailable")
}
