//go:build !linux

package tui

import (
	"errors"
	"os"
	"time"
)

func enterRaw(fd int) (func(), error) {
	return nil, errors.New("raw mode is unavailable")
}

func readKey(f *os.File, wait time.Duration) (string, bool, error) {
	return "", false, errors.New("raw mode is unavailable")
}
