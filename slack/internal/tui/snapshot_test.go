package tui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

func TestAKeyDoesNotPublishTheBackup(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	var n atomic.Int32
	old := publishPlace
	publishPlace = func() error {
		n.Add(1)
		return nil
	}
	t.Cleanup(func() { publishPlace = old })
	sess, err := tuisession.Open("slack", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	m := newModel(workspaces(), store(), sess)
	m.noteSession()
	if n.Load() != 0 {
		t.Fatalf("publishes %d", n.Load())
	}
}

func TestOpenScreenPublishesOnATimer(t *testing.T) {
	var n atomic.Int32
	old := publishPlace
	oldEvery := snapshotEvery
	publishPlace = func() error {
		n.Add(1)
		return nil
	}
	snapshotEvery = 15 * time.Millisecond
	stop := startSnapshots()
	t.Cleanup(func() {
		publishPlace = old
		snapshotEvery = oldEvery
		stop()
	})
	deadline := time.Now().Add(time.Second)
	for n.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if n.Load() == 0 {
		t.Fatal("timer did not publish")
	}
}

func TestLeavingPublishesOnce(t *testing.T) {
	var n atomic.Int32
	old := publishPlace
	oldEvery := snapshotEvery
	publishPlace = func() error {
		n.Add(1)
		return nil
	}
	snapshotEvery = time.Hour
	t.Cleanup(func() {
		publishPlace = old
		snapshotEvery = oldEvery
	})
	stop := startSnapshots()
	stop()
	if n.Load() != 1 {
		t.Fatalf("publishes %d", n.Load())
	}
}
