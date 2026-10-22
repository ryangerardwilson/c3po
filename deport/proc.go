package deport

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// externalMin and externalMax are the safe external ports. 8080 and 8082
// sit in this band. A port outside it is internal and is never signalled.
const (
	externalMin = 8000
	externalMax = 8999
)

func externalPort(port int) bool {
	return port >= externalMin && port <= externalMax
}

// proc is a process holding a socket on the requested port.
type proc struct {
	pid  int
	name string
}

// occupant is one process holding a safe external port.
// pid 0 means the port is taken and its process is not visible.
type occupant struct {
	port int
	pid  int
	name string
}

func formatOccupant(row occupant) string {
	if row.pid == 0 {
		return strconv.Itoa(row.port)
	}
	return fmt.Sprintf("%d %d %s", row.port, row.pid, row.name)
}

// portInodes returns socket inodes whose local port is port.
// Every TCP state counts. A listener, an accepted connection, and a socket
// explicitly bound to that port all keep the port occupied. Client sockets
// whose remote port matches are left alone.
func portInodes(procRoot string, port int) (map[uint64]struct{}, error) {
	texts, err := socketTables(procRoot)
	if err != nil {
		return nil, err
	}
	inodes := map[uint64]struct{}{}
	for _, text := range texts {
		for _, inode := range inodesInTable(text, port) {
			inodes[inode] = struct{}{}
		}
	}
	return inodes, nil
}

func socketTables(procRoot string) ([]string, error) {
	var texts []string
	for _, name := range []string{"tcp", "tcp6", "udp", "udp6"} {
		text, err := os.ReadFile(filepath.Join(procRoot, "net", name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		texts = append(texts, string(text))
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("no socket tables in %s", filepath.Join(procRoot, "net"))
	}
	return texts, nil
}

func inodesInTable(text string, port int) []uint64 {
	var out []uint64
	eachLocal(text, func(found int, inode uint64) {
		if found == port {
			out = append(out, inode)
		}
	})
	return out
}

func eachLocal(text string, fn func(port int, inode uint64)) {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		port, ok := hexPort(fields[1])
		if !ok {
			continue
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil || inode == 0 {
			continue
		}
		fn(port, inode)
	}
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

// occupied lists processes holding a safe external port.
// An occupied port with no visible process is included with pid 0.
func occupied(procRoot string) ([]occupant, error) {
	texts, err := socketTables(procRoot)
	if err != nil {
		return nil, err
	}
	inodePorts := map[uint64]map[int]struct{}{}
	ports := map[int]struct{}{}
	for _, text := range texts {
		eachLocal(text, func(port int, inode uint64) {
			if !externalPort(port) {
				return
			}
			ports[port] = struct{}{}
			set := inodePorts[inode]
			if set == nil {
				set = map[int]struct{}{}
				inodePorts[inode] = set
			}
			set[port] = struct{}{}
		})
	}
	found, err := occupants(procRoot, inodePorts)
	if err != nil {
		return nil, err
	}
	seen := map[int]struct{}{}
	for _, row := range found {
		seen[row.port] = struct{}{}
	}
	for port := range ports {
		if _, ok := seen[port]; !ok {
			found = append(found, occupant{port: port})
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].port != found[j].port {
			return found[i].port < found[j].port
		}
		return found[i].pid < found[j].pid
	})
	return found, nil
}

func occupants(procRoot string, inodePorts map[uint64]map[int]struct{}) ([]occupant, error) {
	if len(inodePorts) == 0 {
		return nil, nil
	}
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, err
	}
	var found []occupant
	seen := map[[2]int]struct{}{}
	self := os.Getpid()
	for _, entry := range entries {
		pid, ok := candidatePID(entry, self)
		if !ok || !threadGroupLeader(procRoot, pid) {
			continue
		}
		name := ""
		for _, inode := range processInodes(procRoot, pid) {
			portSet, ok := inodePorts[inode]
			if !ok {
				continue
			}
			if name == "" {
				name = processName(procRoot, pid)
			}
			for port := range portSet {
				key := [2]int{port, pid}
				if _, dup := seen[key]; dup {
					continue
				}
				seen[key] = struct{}{}
				found = append(found, occupant{port: port, pid: pid, name: name})
			}
		}
	}
	return found, nil
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
	for _, inode := range processInodes(procRoot, pid) {
		if _, yes := inodes[inode]; yes {
			return true
		}
	}
	return false
}

func processInodes(procRoot string, pid int) []uint64 {
	fdDir := filepath.Join(procRoot, strconv.Itoa(pid), "fd")
	fds, err := os.ReadDir(fdDir)
	if err != nil {
		return nil
	}
	var out []uint64
	for _, fd := range fds {
		target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
		if err != nil {
			continue
		}
		inode, ok := socketInode(target)
		if !ok {
			continue
		}
		out = append(out, inode)
	}
	return out
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
