//go:build !linux

package tui

import (
	"errors"
	"time"
)

func enterRaw(fd int) (func(), error) {
	return nil, errors.New("raw mode is unavailable")
}

func termSize(fd int) (int, int) { return 80, 24 }

func waitReadable(fd int, wait time.Duration) (bool, error) {
	return false, errors.New("raw mode is unavailable")
}
