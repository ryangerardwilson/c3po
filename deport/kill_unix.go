//go:build unix

package deport

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

type osKiller struct{}

func (osKiller) Signal(pid int, sig syscall.Signal) error {
	if pid <= 1 || pid == os.Getpid() {
		return fmt.Errorf("refusing to signal %d", pid)
	}
	if err := syscall.Kill(pid, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

func (osKiller) Alive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
