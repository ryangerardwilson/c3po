package tuijob

import (
	"testing"
	"time"
)

func TestJobsRunInOrderAndApplyOnTheCaller(t *testing.T) {
	q := New()
	defer q.Close()
	var got []int
	release := make(chan struct{})
	q.Go(func() func() {
		<-release
		return func() { got = append(got, 1) }
	})
	q.Go(func() func() { return func() { got = append(got, 2) } })
	if !q.Busy() || q.Pending() != 2 {
		t.Fatalf("pending %d", q.Pending())
	}
	if q.Wait(20 * time.Millisecond) {
		t.Fatal("a blocked job reported idle")
	}
	if q.Apply() || len(got) != 0 {
		t.Fatalf("applied before the job finished: %v", got)
	}
	close(release)
	if !q.Wait(time.Second) {
		t.Fatal("jobs did not finish")
	}
	if !q.Apply() || len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("got %v", got)
	}
	if q.Busy() {
		t.Fatal("busy after apply")
	}
}

func TestANilQueueIsIdle(t *testing.T) {
	var q *Queue
	if q.Busy() || q.Apply() || !q.Wait(0) || q.Pending() != 0 {
		t.Fatal("nil queue is not idle")
	}
	q.Close()
}

func TestGoAfterCloseIsIgnored(t *testing.T) {
	q := New()
	q.Close()
	q.Go(func() func() { t.Fatal("ran after close"); return nil })
	if q.Busy() {
		t.Fatal("busy after close")
	}
}

func TestGoNeverBlocksTheLoop(t *testing.T) {
	q := New()
	defer q.Close()
	release := make(chan struct{})
	q.Go(func() func() { <-release; return nil })
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			q.Go(func() func() { return nil })
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("queueing 1000 jobs behind a running one blocked")
	}
	close(release)
	if !q.Wait(5 * time.Second) {
		t.Fatal("jobs did not finish")
	}
	q.Apply()
	if q.Busy() {
		t.Fatalf("pending %d", q.Pending())
	}
}
