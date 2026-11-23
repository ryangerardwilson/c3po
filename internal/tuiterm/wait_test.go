//go:build linux

package tuiterm

import (
	"os"
	"testing"
	"time"
)

func TestWaitReadableSeesInputAndTimesOut(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	start := time.Now()
	if ready, err := WaitReadable(int(r.Fd()), 30*time.Millisecond); err != nil || ready {
		t.Fatalf("empty pipe ready %v err %v", ready, err)
	}
	if time.Since(start) < 20*time.Millisecond {
		t.Fatal("did not wait")
	}
	if _, err := w.Write([]byte("j")); err != nil {
		t.Fatal(err)
	}
	if ready, err := WaitReadable(int(r.Fd()), time.Second); err != nil || !ready {
		t.Fatalf("pipe with input ready %v err %v", ready, err)
	}
}
