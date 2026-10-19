package deport

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const tcpHeader = "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n"

func TestInodesInTableUsesLocalPort(t *testing.T) {
	text := tcpHeader +
		"   0: 3500007F:0035 00000000:0000 0A 00000000:00000000 00:00000000 00000000   974        0 9621 1 0000000089a25235 100 0 0 10 5\n" +
		"   1: 0100007F:1F92 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 55 1 0000000000000000 100 0 0 10 0\n" +
		"   2: 0100007F:C000 0100007F:1F92 01 00000000:00000000 00:00000000 00000000     0        0 77 1 0000000000000000 100 0 0 10 0\n" +
		"   3: 00000000000000000000000000000000:1F92 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 88 1 0000000000000000 100 0 0 10 0\n"

	got := inodesInTable(text, 8082)
	if len(got) != 2 || got[0] != 55 || got[1] != 88 {
		t.Fatalf("inodes = %v", got)
	}
	if dns := inodesInTable(text, 53); len(dns) != 1 || dns[0] != 9621 {
		t.Fatalf("dns inodes = %v", dns)
	}
}

func TestHoldersFindsLeaderOnce(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("1F92", 55))
	writeProcess(t, root, 4242, 4242, "socket:[55]", "/usr/bin/python\x00-m\x00http.server\x00")
	writeProcess(t, root, 4243, 4242, "socket:[55]", "python\x00")
	writeProcess(t, root, 99, 99, "socket:[1]", "other\x00")

	found, err := holders(root, map[uint64]struct{}{55: {}})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].pid != 4242 || found[0].name != "python" {
		t.Fatalf("holders = %+v", found)
	}
}

func writeTable(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "net")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func listenLine(portHex string, inode int) string {
	return fmt.Sprintf("   0: 0100007F:%s 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 %d 1 0000000000000000 100 0 0 10 0\n", portHex, inode)
}

func writeProcess(t *testing.T, root string, pid, tgid int, socket, cmdline string) {
	t.Helper()
	dir := filepath.Join(root, strconv.Itoa(pid))
	fdDir := filepath.Join(dir, "fd")
	if err := os.MkdirAll(fdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	status := fmt.Sprintf("Name:\tproc\nTgid:\t%d\n", tgid)
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(socket, filepath.Join(fdDir, "3")); err != nil {
		t.Fatal(err)
	}
}

func TestPortInodesReadsEveryTable(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp6", tcpHeader+listenLine("1F92", 61))
	writeTable(t, root, "udp", tcpHeader+listenLine("1F92", 62))
	got, err := portInodes(root, 8082)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[61]; !ok || len(got) != 2 {
		t.Fatalf("inodes = %v", got)
	}
	if _, err := portInodes(t.TempDir(), 8082); err == nil || !strings.Contains(err.Error(), "no socket tables") {
		t.Fatalf("missing tables err = %v", err)
	}
}

func TestSocketInodeAndName(t *testing.T) {
	if n, ok := socketInode("socket:[123]"); !ok || n != 123 {
		t.Fatalf("socket inode %d %v", n, ok)
	}
	for _, bad := range []string{"pipe:[1]", "socket:[0]", "socket:[x]", "socket:[5"} {
		if _, ok := socketInode(bad); ok {
			t.Fatalf("%q parsed", bad)
		}
	}
	if oneLine(" a\nb\n") != "a b" || oneLine("\n") != "?" {
		t.Fatal("oneLine")
	}
	if p, ok := hexPort("0100007F:1F92"); !ok || p != 8082 {
		t.Fatalf("hexPort %d %v", p, ok)
	}
	if _, ok := hexPort("0100007F:"); ok {
		t.Fatal("empty port parsed")
	}
}

func TestProcessNameFallsBackToComm(t *testing.T) {
	root := t.TempDir()
	writeProcess(t, root, 50, 50, "socket:[1]", "")
	if err := os.WriteFile(filepath.Join(root, "50", "comm"), []byte("worker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := processName(root, 50); got != "worker" {
		t.Fatalf("name = %q", got)
	}
	if got := processName(root, 51); got != "?" {
		t.Fatalf("missing name = %q", got)
	}
}
