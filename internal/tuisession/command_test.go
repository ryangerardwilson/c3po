package tuisession

import (
	"bytes"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func run(t *testing.T, args ...string) []byte {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Command(args, &stdout, &stderr); code != 0 {
		t.Fatalf("%v exit %d: %s", args, code, stderr.String())
	}
	return stdout.Bytes()
}

func TestCommandKeepsAnOutsideScreensSession(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	pid := strconv.Itoa(os.Getpid())
	var opened struct {
		ID       string `json:"id"`
		Restored bool   `json:"restored"`
	}
	if err := json.Unmarshal(run(t, "open", "tunnel", pid), &opened); err != nil || opened.ID == "" || opened.Restored {
		t.Fatalf("open %+v %v", opened, err)
	}
	run(t, "place", opened.ID, pid, "routes", "www")
	run(t, "note", opened.ID, pid, `{"args":["-a"]}`)
	var active []ActiveRow
	if err := json.Unmarshal(run(t, "active"), &active); err != nil || len(active) != 1 || active[0].Screen != "tunnel" || active[0].PID != os.Getpid() {
		t.Fatalf("active %+v %v", active, err)
	}
	run(t, "close", opened.ID)
	if err := json.Unmarshal(run(t, "active"), &active); err != nil || len(active) != 0 {
		t.Fatalf("active after close %+v %v", active, err)
	}
	var again struct {
		Restored bool      `json:"restored"`
		Place    placeJSON `json:"place"`
	}
	if err := json.Unmarshal(run(t, "open", "tunnel", pid, opened.ID), &again); err != nil || !again.Restored ||
		again.Place.Path != "routes" || again.Place.Cursor != "www" || len(again.Place.Args) != 1 {
		t.Fatalf("again %+v %v", again, err)
	}
	var stdout, stderr bytes.Buffer
	if code := Command([]string{"open", "o", pid, opened.ID}, &stdout, &stderr); code != 2 {
		t.Fatalf("wrong screen exit %d", code)
	}
	if code := Command([]string{"place", "01a17622-5fd9-7bc2-81ff-3e200a8c16b7", pid, "", ""}, &stdout, &stderr); code != 2 {
		t.Fatalf("missing session exit %d", code)
	}
}
