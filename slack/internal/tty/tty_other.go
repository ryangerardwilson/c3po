//go:build !linux

// Package tty reports whether the process stdio is an interactive terminal.
package tty

import "os"

// IsInteractive is a best-effort fallback off Linux. Production installs
// target Linux; agent runners that must fail closed are Linux-based.
func IsInteractive() bool {
	return fileLooksLikeTerminal(os.Stdin) && fileLooksLikeTerminal(os.Stdout)
}

func fileLooksLikeTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	// Require character device and reject named pipes/sockets.
	mode := fi.Mode()
	return mode&os.ModeCharDevice != 0 && mode&os.ModeNamedPipe == 0 && mode&os.ModeSocket == 0
}
