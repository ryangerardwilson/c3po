//go:build !linux

package looptab

import "errors"

func enterRaw(fd int) (func(), error) {
	return nil, errors.New("raw mode is unavailable")
}

func termSize(fd int) (int, int) { return 80, 24 }
