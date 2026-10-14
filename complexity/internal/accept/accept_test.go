package accept

import "testing"

func TestPutReplacesAndRoundTrips(t *testing.T) {
	root := t.TempDir()
	var b Book
	b.Put(Record{File: "a.go", Function: "f", Metric: "cognitive", Score: 20, Reason: "old"})
	b.Put(Record{File: "a.go", Function: "f", Metric: "cognitive", Score: 21, Reason: "new"})
	if len(b.Accepted) != 1 || b.Accepted[0].Score != 21 {
		t.Fatalf("%+v", b)
	}
	if err := Save(root, b); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := got.Find("a.go", "f", "cognitive"); !ok || r.Reason != "new" {
		t.Fatalf("%+v %v", r, ok)
	}
	if _, ok := got.Find("a.go", "f", "nesting"); ok {
		t.Fatal("nesting was not accepted")
	}
}

func TestMissingFileIsEmpty(t *testing.T) {
	b, err := Load(t.TempDir())
	if err != nil || len(b.Accepted) != 0 {
		t.Fatalf("%+v %v", b, err)
	}
}
