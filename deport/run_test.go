package deport

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/version"
)

func TestMain(m *testing.M) {
	if os.Getenv("DEPORT_HELPER") == "listen" {
		listenHelper()
		return
	}
	os.Exit(m.Run())
}

func noSleep(time.Duration) {}

func fakeEnv(root string, k killer) env {
	return env{procRoot: root, killer: k, sleep: noSleep}
}

func runFake(t *testing.T, root string, k killer, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb, fakeEnv(root, k))
	return out.String(), errb.String(), code
}

func TestParsePort(t *testing.T) {
	for arg, want := range map[string]int{"8082": 8082, "1": 1, "65535": 65535} {
		port, err := parsePort(arg)
		if err != nil || port != want {
			t.Fatalf("parse %q: %d %v", arg, port, err)
		}
	}
	for _, arg := range []string{"0", "65536", "nope", "8082 ", "-1", ""} {
		_, err := parsePort(arg)
		var usage usageError
		if err == nil || !errors.As(err, &usage) {
			t.Fatalf("parse %q err = %v", arg, err)
		}
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		var out, errb bytes.Buffer
		code := Run([]string{arg}, &out, &errb)
		if code != 0 || out.String() != Help || errb.Len() != 0 {
			t.Fatalf("%s: code %d out %q err %q", arg, code, out.String(), errb.String())
		}
	}
	for _, want := range []string{"c3po deport <port>", "c3po deport list", "c3po deport 8082", "c3po tui deport", "8000", "8999", "SIGTERM", "SIGKILL", "is free", "Exit 0"} {
		if !strings.Contains(Help, want) {
			t.Fatalf("help missing %q", want)
		}
	}
	for _, line := range strings.Split(Help, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "deport ") {
			t.Fatalf("help names the old binary: %q", line)
		}
	}
}

func TestVersionPrintsC3poVersion(t *testing.T) {
	for _, arg := range []string{"version", "-v", "--version"} {
		var out, errb bytes.Buffer
		code := Run([]string{arg}, &out, &errb)
		if code != 0 || out.String() != "c3po "+version.String()+"\n" {
			t.Fatalf("%s: code %d out %q", arg, code, out.String())
		}
	}
}

func TestUsageErrorsExit2(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"nope"}, {"0"}, {"65536"}, {"8082", "8083"}, {"help", "extra"}, {"--port"}} {
		root := t.TempDir()
		writeTable(t, root, "tcp", tcpHeader)
		k := &signalLog{}
		out, errb, code := runFake(t, root, k, args...)
		if code != 2 || out != "" || len(k.sigs) != 0 {
			t.Fatalf("%q: code %d out %q sigs %v", args, code, out, k.sigs)
		}
		if !strings.HasPrefix(errb, "deport: ") || !strings.Contains(errb, "Usage:") {
			t.Fatalf("%q: stderr %q", args, errb)
		}
	}
}

func TestRunAlreadyFree(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader)
	k := &signalLog{}
	out, errb, code := runFake(t, root, k, "8082")
	if code != 0 || out != "8082 is free\n" || errb != "" || len(k.sigs) != 0 {
		t.Fatalf("code %d out %q err %q sigs %v", code, out, errb, k.sigs)
	}
}

func TestRunNoTablesExits1(t *testing.T) {
	out, errb, code := runFake(t, t.TempDir(), &signalLog{}, "8082")
	if code != 1 || out != "" || !strings.Contains(errb, "no socket tables") || strings.Contains(errb, "Usage:") {
		t.Fatalf("code %d out %q err %q", code, out, errb)
	}
}

func TestRunInvisibleProcessExits1(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("1F92", 55))
	_, errb, code := runFake(t, root, &signalLog{}, "8082")
	if code != 1 || !strings.Contains(errb, "not visible") {
		t.Fatalf("code %d err %q", code, errb)
	}
}

func TestRunTermThenKill(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("1F92", 55))
	writeProcess(t, root, 4242, 4242, "socket:[55]", "python\x00")
	k := &signalLog{alive: map[int]bool{4242: true}}
	k.release = func(sig syscall.Signal) {
		if sig != syscall.SIGKILL {
			return
		}
		k.alive[4242] = false
		writeTable(t, root, "tcp", tcpHeader)
	}
	out, errb, code := runFake(t, root, k, "8082")
	if code != 0 || errb != "" {
		t.Fatalf("code %d err %q", code, errb)
	}
	if len(k.sigs) != 2 || k.sigs[0] != syscall.SIGTERM || k.sigs[1] != syscall.SIGKILL {
		t.Fatalf("sigs = %v", k.sigs)
	}
	if out != "killed 4242 python\n8082 is free\n" {
		t.Fatalf("output = %q", out)
	}
}

func TestRunStopsAfterTerm(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "udp", tcpHeader+listenLine("1F92", 55))
	writeProcess(t, root, 7, 7, "socket:[55]", "nc\x00")
	k := &signalLog{alive: map[int]bool{7: true}}
	k.release = func(sig syscall.Signal) {
		if sig != syscall.SIGTERM {
			return
		}
		k.alive[7] = false
		writeTable(t, root, "udp", tcpHeader)
	}
	out, _, code := runFake(t, root, k, "8082")
	if code != 0 || out != "killed 7 nc\n8082 is free\n" || len(k.sigs) != 1 {
		t.Fatalf("code %d out %q sigs %v", code, out, k.sigs)
	}
}

func TestRunStillOccupiedExits1(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("1F92", 55))
	writeProcess(t, root, 4242, 4242, "socket:[55]", "python\x00")
	k := &signalLog{alive: map[int]bool{4242: true}}
	var slept time.Duration
	var out, errb bytes.Buffer
	code := run([]string{"8082"}, &out, &errb, env{procRoot: root, killer: k, sleep: func(d time.Duration) { slept += d }})
	if code != 1 || out.String() != "killed 4242 python\n" || !strings.Contains(errb.String(), "port 8082 is still in use") {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if len(k.sigs) != 2 || slept < termTimeout+killTimeout {
		t.Fatalf("sigs %v slept %v", k.sigs, slept)
	}
}

func TestRunSignalFailureExits1(t *testing.T) {
	root := t.TempDir()
	writeTable(t, root, "tcp", tcpHeader+listenLine("1F92", 55))
	writeProcess(t, root, 4242, 4242, "socket:[55]", "python\x00")
	k := &signalLog{alive: map[int]bool{4242: true}, fail: errors.New("operation not permitted")}
	out, errb, code := runFake(t, root, k, "8082")
	if code != 1 || out != "" || !strings.Contains(errb, "operation not permitted") {
		t.Fatalf("code %d out %q err %q", code, out, errb)
	}
}

func TestSignalAllSkipsDeadAndDuplicates(t *testing.T) {
	found := []proc{{pid: 10, name: "a"}, {pid: 11, name: "b"}}
	k := &signalLog{alive: map[int]bool{11: true}}
	killed, err := signalAll(found, k, syscall.SIGKILL, []proc{{pid: 11, name: "b"}})
	if err != nil || len(killed) != 1 || len(k.sigs) != 1 || k.pids[0] != 11 {
		t.Fatalf("killed %v err %v pids %v", killed, err, k.pids)
	}
}

func TestOsKillerRefusesSelfAndInit(t *testing.T) {
	for _, pid := range []int{0, 1, os.Getpid()} {
		if err := (osKiller{}).Signal(pid, syscall.SIGTERM); err == nil {
			t.Fatalf("signalled %d", pid)
		}
	}
	if !(osKiller{}).Alive(os.Getpid()) {
		t.Fatal("self not alive")
	}
}

func TestDeportFreesListener(t *testing.T) {
	if _, err := os.Stat("/proc/net/tcp"); err != nil {
		t.Skip("no /proc/net")
	}
	for _, tc := range []struct {
		proto    string
		stubborn bool
	}{
		{proto: "tcp"},
		{proto: "udp"},
		{proto: "tcp", stubborn: true},
	} {
		t.Run(tc.proto+"-stubborn-"+strconv.FormatBool(tc.stubborn), func(t *testing.T) {
			port, cmd := startHelper(t, tc.proto, tc.stubborn)
			var out, errb bytes.Buffer
			code := Run([]string{strconv.Itoa(port)}, &out, &errb)
			if code != 0 {
				t.Fatalf("code %d err %q", code, errb.String())
			}
			wantKill := fmt.Sprintf("killed %d", cmd.Process.Pid)
			if !strings.Contains(out.String(), wantKill) || !strings.Contains(out.String(), fmt.Sprintf("%d is free\n", port)) {
				t.Fatalf("output = %q", out.String())
			}
			assertCanBind(t, tc.proto, port)
		})
	}
}

type signalLog struct {
	sigs    []syscall.Signal
	pids    []int
	alive   map[int]bool
	fail    error
	release func(syscall.Signal)
}

func (s *signalLog) Signal(pid int, sig syscall.Signal) error {
	if s.fail != nil {
		return s.fail
	}
	s.sigs = append(s.sigs, sig)
	s.pids = append(s.pids, pid)
	if s.release != nil {
		s.release(sig)
	}
	return nil
}

func (s *signalLog) Alive(pid int) bool {
	return s.alive[pid]
}

func listenHelper() {
	if os.Getenv("DEPORT_STUBBORN") == "1" {
		signal.Ignore(syscall.SIGTERM)
	}
	addr, closer, err := helperListen(os.Getenv("DEPORT_PROTO"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closer()
	_, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(port)
	select {}
}

func helperListen(proto string) (net.Addr, func(), error) {
	addr := net.JoinHostPort("127.0.0.1", os.Getenv("DEPORT_PORT"))
	if proto == "udp" {
		conn, err := net.ListenPacket("udp", addr)
		if err != nil {
			return nil, nil, err
		}
		return conn.LocalAddr(), func() { conn.Close() }, nil
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	go acceptAndClose(ln)
	return ln.Addr(), func() { ln.Close() }, nil
}

func freeSafePort(t *testing.T) int {
	t.Helper()
	for port := externalMin; port <= externalMax; port++ {
		tcp, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		tcp.Close()
		udp, err := net.ListenPacket("udp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		udp.Close()
		return port
	}
	t.Skip("no free safe external port")
	return 0
}

func acceptAndClose(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}
}

func startHelper(t *testing.T, proto string, stubborn bool) (int, *exec.Cmd) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "DEPORT_HELPER=listen", "DEPORT_PROTO="+proto, "DEPORT_PORT="+strconv.Itoa(freeSafePort(t)))
	if stubborn {
		cmd.Env = append(cmd.Env, "DEPORT_STUBBORN=1")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	line := make(chan string, 1)
	readErr := make(chan error, 1)
	go func() {
		text, err := bufio.NewReader(stdout).ReadString('\n')
		if err != nil {
			readErr <- err
			return
		}
		line <- text
	}()
	select {
	case text := <-line:
		port, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			t.Fatal(err)
		}
		return port, cmd
	case err := <-readErr:
		t.Fatalf("helper: %v stderr: %s", err, stderr.String())
	case <-time.After(3 * time.Second):
		t.Fatalf("helper timeout stderr: %s", stderr.String())
	}
	return 0, nil
}

func assertCanBind(t *testing.T, proto string, port int) {
	t.Helper()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if proto == "udp" {
		conn, err := net.ListenPacket("udp", addr)
		if err != nil {
			t.Fatalf("port %d still taken: %v", port, err)
		}
		conn.Close()
		return
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("port %d still taken: %v", port, err)
	}
	ln.Close()
}
