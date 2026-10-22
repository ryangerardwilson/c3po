//go:build !linux

package deport

import "errors"

func enterRaw(fd int) (func(), error) {
	return nil, errors.New("raw mode is unavailable")
}

func termSize(fd int) (int, int) { return 80, 24 }
