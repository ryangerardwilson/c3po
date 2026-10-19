//go:build !linux

package cyclotree

import (
	"errors"
	"os"
	"time"
)

func enterRaw(fd int) (func(), error) {
	return nil, errors.New("raw mode is unavailable")
}

func termRows(fd int) int { return 24 }

func readKey(f *os.File) (rune, error) {
	return 0, errors.New("raw mode is unavailable")
}

func nextKey(_ *os.File) (rune, bool) { return 0, false }

func readKeyWait(f *os.File, _ time.Duration) (rune, bool, error) {
	k, err := readKey(f)
	if err != nil {
		return 0, false, err
	}
	return k, true, nil
}
