package deport

import (
	"errors"
	"fmt"
	"syscall"
	"time"
)

// killer delivers signals to a process.
type killer interface {
	Signal(pid int, sig syscall.Signal) error
	Alive(pid int) bool
}

// free kills every process bound to port and waits until the port is clear.
// A nil result with a nil error means the port was already free.
// A port outside the safe external band is refused before any signal.
func free(procRoot string, port int, k killer, sleep func(time.Duration)) ([]proc, error) {
	if !externalPort(port) {
		return nil, usageError(fmt.Sprintf("%d is not a safe external port", port))
	}
	found, err := boundProcesses(procRoot, port)
	if err != nil || len(found) == 0 {
		return nil, err
	}
	killed, _ := signalAll(found, k, syscall.SIGTERM, nil)
	clear, err := waitClear(procRoot, port, termTimeout, sleep)
	if err != nil || clear {
		return killed, err
	}
	killed, sigErr := signalAll(found, k, syscall.SIGKILL, killed)
	clear, err = waitClear(procRoot, port, killTimeout, sleep)
	if err != nil || clear {
		return killed, err
	}
	if sigErr != nil {
		return killed, sigErr
	}
	return killed, fmt.Errorf("port %d is still in use", port)
}

// boundProcesses finds the holders of port. It looks twice, because a
// socket can change hands between reading the tables and reading fds.
func boundProcesses(procRoot string, port int) ([]proc, error) {
	found, busy, err := lookOnce(procRoot, port)
	if err != nil || !busy || len(found) > 0 {
		return found, err
	}
	found, busy, err = lookOnce(procRoot, port)
	if err != nil || !busy {
		return nil, err
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("port %d is in use, but its process is not visible", port)
	}
	return found, nil
}

func lookOnce(procRoot string, port int) ([]proc, bool, error) {
	inodes, err := portInodes(procRoot, port)
	if err != nil || len(inodes) == 0 {
		return nil, false, err
	}
	found, err := holders(procRoot, inodes)
	return found, true, err
}

func signalAll(found []proc, k killer, sig syscall.Signal, killed []proc) ([]proc, error) {
	var sigErr error
	for _, p := range found {
		if sig == syscall.SIGKILL && !k.Alive(p.pid) {
			continue
		}
		if err := k.Signal(p.pid, sig); err != nil {
			sigErr = errors.Join(sigErr, err)
			continue
		}
		if !containsPID(killed, p.pid) {
			killed = append(killed, p)
		}
	}
	return killed, sigErr
}

func containsPID(procs []proc, pid int) bool {
	for _, p := range procs {
		if p.pid == pid {
			return true
		}
	}
	return false
}

func waitClear(procRoot string, port int, timeout time.Duration, sleep func(time.Duration)) (bool, error) {
	var waited time.Duration
	for {
		inodes, err := portInodes(procRoot, port)
		if err != nil {
			return false, err
		}
		if len(inodes) == 0 {
			return true, nil
		}
		if waited >= timeout {
			return false, nil
		}
		sleep(pollEvery)
		waited += pollEvery
	}
}
