//go:build !linux

package tui

import (
	"errors"
	"os"
	"time"
)

type terminal struct {
	in, out *os.File
}

func openTerminal() (*terminal, error) {
	return nil, errors.New("could not open a new TTY: unsupported platform")
}

func (t *terminal) close()     {}
func (t *terminal) raw() error { return errors.New("raw mode is unavailable") }
func (t *terminal) restore()   {}

func (t *terminal) size() WindowSizeMsg { return WindowSizeMsg{Width: 80, Height: 24} }

func (t *terminal) readKeys([]byte, time.Duration) ([]KeyMsg, error) {
	return nil, errors.New("raw mode is unavailable")
}

func (p *Program) watchSignals() func() { return func() {} }
