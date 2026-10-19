//go:build !unix

package deport

import (
	"fmt"
	"syscall"
)

type osKiller struct{}

func (osKiller) Signal(pid int, sig syscall.Signal) error {
	return fmt.Errorf("cannot signal %d on this platform", pid)
}

func (osKiller) Alive(int) bool { return false }
