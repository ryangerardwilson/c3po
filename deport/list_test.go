package deport

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRefusesInternalPort(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("0016", 55))
	writeProcess(t, root, 4242, 4242, "socket:[55]", "sshd\x00")
	k := &signalLog{alive: map[int]bool{4242: true}}
	for _, arg := range []string{"22", "53", "3000", "7999", "9000", "65535"} {
		out, errb, code := runFake(t, root, k, arg)
		if code != 2 || out != "" || len(k.sigs) != 0 || !strings.Contains(errb, arg+" is not a safe external port") {
			t.Fatalf("%s: code %d out %q err %q sigs %v", arg, code, out, errb, k.sigs)
		}
	}
}

func TestListPrintsExternalOnly(t *testing.T) {
	root := t.TempDir()
	body := tcpHeader +
		listenLine("0016", 1) +
		listenLine("1F40", 2) +
		listenLine("1F90", 3) +
		listenLine("2327", 4) +
		listenLine("2328", 5)
	writeTable(t, root, "tcp", body)
	writeProcess(t, root, 10, 10, "socket:[2]", "a\x00")
	writeProcess(t, root, 11, 11, "socket:[3]", "b\x00")
	writeProcess(t, root, 12, 12, "socket:[4]", "c\x00")
	writeProcess(t, root, 13, 13, "socket:[1]", "sshd\x00")
	writeProcess(t, root, 14, 14, "socket:[5]", "other\x00")
	out, errb, code := runFake(t, root, &signalLog{}, "list")
	if code != 0 || errb != "" {
		t.Fatalf("code %d err %q out %q", code, errb, out)
	}
	want := "8000 10 a\n8080 11 b\n8999 12 c\n"
	if out != want {
		t.Fatalf("list = %q", out)
	}
}

func TestListEmptyWhenOnlyInternalPortsAreTaken(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("0016", 1))
	writeProcess(t, root, 13, 13, "socket:[1]", "sshd\x00")
	out, errb, code := runFake(t, root, &signalLog{}, "list")
	if code != 0 || out != "no occupied external ports\n" || errb != "" {
		t.Fatalf("code %d out %q err %q", code, out, errb)
	}
}

func TestListShowsAPortWithNoProcess(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("1F90", 9))
	out, errb, code := runFake(t, root, &signalLog{}, "list")
	if code != 0 || errb != "" || out != "8080\n" {
		t.Fatalf("code %d out %q err %q", code, out, errb)
	}
}

func TestLiveInternalPortStaysUp(t *testing.T) {
	before, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		t.Skip(err.Error())
	}
	if !strings.Contains(string(before), ":0035 ") {
		t.Skip("port 53 is not listening")
	}
	var out, errb bytes.Buffer
	code := Run([]string{"53"}, &out, &errb)
	after, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		t.Fatal(err)
	}
	if code != 2 || out.Len() != 0 || !strings.Contains(errb.String(), "53 is not a safe external port") {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if !strings.Contains(string(after), ":0035 ") {
		t.Fatal("port 53 disappeared")
	}
}

func TestOccupiedMergesOneProcess(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("1F90", 3)+listenLine("0016", 1))
	writeTable(t, root, "udp", tcpHeader+listenLine("1F90", 4))
	writeProcess(t, root, 11, 11, "socket:[3]", "node\x00")
	writeProcess(t, root, 12, 12, "socket:[4]", "node\x00")
	got, err := occupied(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].port != 8080 || got[0].pid != 11 || got[1].port != 8080 || got[1].pid != 12 {
		t.Fatalf("occupied = %+v", got)
	}
}
