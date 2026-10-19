package deport

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// proc is a process holding a socket on the requested port.
type proc struct {
	pid  int
	name string
}

// portInodes returns socket inodes whose local port is port.
// Every TCP state counts. A listener, an accepted connection, and a socket
// explicitly bound to that port all keep the port occupied. Client sockets
// whose remote port matches are left alone.
func portInodes(procRoot string, port int) (map[uint64]struct{}, error) {
	foundTable := false
	inodes := map[uint64]struct{}{}
	for _, name := range []string{"tcp", "tcp6", "udp", "udp6"} {
		text, err := os.ReadFile(filepath.Join(procRoot, "net", name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		foundTable = true
		for _, inode := range inodesInTable(string(text), port) {
			inodes[inode] = struct{}{}
		}
	}
	if !foundTable {
		return nil, fmt.Errorf("no socket tables in %s", filepath.Join(procRoot, "net"))
	}
	return inodes, nil
}

func inodesInTable(text string, port int) []uint64 {
	var out []uint64
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		localPort, ok := hexPort(fields[1])
		if !ok || localPort != port {
			continue
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil || inode == 0 {
			continue
		}
		out = append(out, inode)
	}
	return out
}

func hexPort(addr string) (int, bool) {
	colon := strings.LastIndex(addr, ":")
	if colon < 0 || colon == len(addr)-1 {
		return 0, false
	}
	n, err := strconv.ParseUint(addr[colon+1:], 16, 16)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

func holders(procRoot string, inodes map[uint64]struct{}) ([]proc, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, err
	}
	var found []proc
	self := os.Getpid()
	for _, entry := range entries {
		pid, ok := candidatePID(entry, self)
		if !ok || !threadGroupLeader(procRoot, pid) || !holdsInode(procRoot, pid, inodes) {
			continue
		}
		found = append(found, proc{pid: pid, name: processName(procRoot, pid)})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].pid < found[j].pid })
	return found, nil
}

// candidatePID is the pid named by a /proc entry, skipping init and self.
func candidatePID(entry os.DirEntry, self int) (int, bool) {
	if !entry.IsDir() {
		return 0, false
	}
	pid, err := strconv.Atoi(entry.Name())
	if err != nil || pid <= 1 || pid == self {
		return 0, false
	}
	return pid, true
}

func threadGroupLeader(procRoot string, pid int) bool {
	text, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "status"))
	if err != nil {
		return true
	}
	for _, line := range strings.Split(string(text), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if ok && key == "Tgid" {
			return strings.TrimSpace(val) == strconv.Itoa(pid)
		}
	}
	return true
}

func holdsInode(procRoot string, pid int, inodes map[uint64]struct{}) bool {
	fdDir := filepath.Join(procRoot, strconv.Itoa(pid), "fd")
	fds, err := os.ReadDir(fdDir)
	if err != nil {
		return false
	}
	for _, fd := range fds {
		target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
		if err != nil {
			continue
		}
		inode, ok := socketInode(target)
		if !ok {
			continue
		}
		if _, yes := inodes[inode]; yes {
			return true
		}
	}
	return false
}

func socketInode(target string) (uint64, bool) {
	inner, ok := strings.CutPrefix(target, "socket:[")
	if !ok {
		return 0, false
	}
	inner, ok = strings.CutSuffix(inner, "]")
	if !ok {
		return 0, false
	}
	inode, err := strconv.ParseUint(inner, 10, 64)
	if err != nil || inode == 0 {
		return 0, false
	}
	return inode, true
}

func processName(procRoot string, pid int) string {
	if name := argv0(procRoot, pid); name != "" {
		return oneLine(name)
	}
	comm, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "comm"))
	if err != nil {
		return "?"
	}
	return oneLine(string(comm))
}

func argv0(procRoot string, pid int) string {
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return ""
	}
	first, _, _ := strings.Cut(string(raw), "\x00")
	if first == "" {
		return ""
	}
	return filepath.Base(first)
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "?"
	}
	return s
}
