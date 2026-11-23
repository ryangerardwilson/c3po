// Package tuijob runs a screen's slow work off its key loop. Jobs run one at
// a time, in the order the screen queued them, on one worker goroutine. Each
// job returns a function that the loop runs when it calls Apply, so screen
// state is only ever touched by the loop. The package holds no domain logic.
package tuijob

import (
	"sync"
	"sync/atomic"
	"time"
)

// Work runs on the worker. It returns the change to make on the loop, or nil.
// Work must bound its own I/O with a timeout.
type Work func() func()

// Queue is one screen's serial worker. Its waiting jobs and finished
// results are unbounded lists, so neither Go nor the worker ever blocks the
// screen loop.
type Queue struct {
	mu        sync.Mutex
	wake      *sync.Cond
	waiting   []Work
	finished  []func()
	closed    bool
	queued    atomic.Int64 // queued or running, not yet finished
	unapplied atomic.Int64 // finished or not, not yet applied
}

// New starts the worker.
func New() *Queue {
	q := &Queue{}
	q.wake = sync.NewCond(&q.mu)
	go q.run()
	return q
}

func (q *Queue) run() {
	for {
		q.mu.Lock()
		for len(q.waiting) == 0 && !q.closed {
			q.wake.Wait()
		}
		if len(q.waiting) == 0 {
			q.mu.Unlock()
			return
		}
		work := q.waiting[0]
		q.waiting = q.waiting[1:]
		q.mu.Unlock()
		apply := work()
		// The result is handed over before the job counts as finished, so
		// Apply after Wait always finds it.
		q.mu.Lock()
		q.finished = append(q.finished, apply)
		q.mu.Unlock()
		q.queued.Add(-1)
	}
}

// Go queues work behind every job queued before it.
func (q *Queue) Go(work Work) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.queued.Add(1)
	q.unapplied.Add(1)
	q.waiting = append(q.waiting, work)
	q.wake.Signal()
}

// Busy reports whether a job has not been applied yet.
func (q *Queue) Busy() bool {
	return q != nil && q.unapplied.Load() > 0
}

// Pending is the number of jobs not applied yet.
func (q *Queue) Pending() int {
	if q == nil {
		return 0
	}
	return int(q.unapplied.Load())
}

// Apply runs the result of every finished job on the caller, in order, and
// reports whether any ran. It does not wait.
func (q *Queue) Apply() bool {
	if q == nil {
		return false
	}
	q.mu.Lock()
	ready := q.finished
	q.finished = nil
	q.mu.Unlock()
	for _, apply := range ready {
		q.unapplied.Add(-1)
		if apply != nil {
			apply()
		}
	}
	return len(ready) > 0
}

// Wait blocks until every queued job has finished or d passes, and reports
// whether the worker is idle. It does not apply the results.
func (q *Queue) Wait(d time.Duration) bool {
	if q == nil {
		return true
	}
	deadline := time.Now().Add(d)
	for q.queued.Load() > 0 {
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(2 * time.Millisecond)
	}
	return true
}

// Close lets the worker finish what is queued and stop. Go after Close is
// ignored.
func (q *Queue) Close() {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.wake.Broadcast()
}
