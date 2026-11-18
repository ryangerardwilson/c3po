package tui

import (
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// snapshotEvery is how often an open slack screen publishes its place.
var snapshotEvery = 10 * time.Minute

// publishPlace uploads the c3po database. Tests replace it.
var publishPlace = func() error { return sqlxdb.Publish() }

func startSnapshots() func() {
	stop := make(chan struct{})
	go snapshotLoop(stop)
	return func() {
		close(stop)
		_ = publishPlace()
	}
}

func snapshotLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(snapshotEvery)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			_ = publishPlace()
		}
	}
}
