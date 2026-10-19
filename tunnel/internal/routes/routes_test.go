package routes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalize(t *testing.T) {
	name, host, err := Normalize(Zone, "Notes")
	if err != nil || name != "notes" || host != "notes.ryangerardwilson.com" {
		t.Fatalf("notes: %s %s %v", name, host, err)
	}
	name, host, err = Normalize(Zone, "notes.ryangerardwilson.com")
	if err != nil || name != "notes" || host != "notes.ryangerardwilson.com" {
		t.Fatalf("full: %s %s %v", name, host, err)
	}
	name, host, err = Normalize(Zone, "@")
	if err != nil || name != Apex || host != Zone {
		t.Fatalf("apex: %s %s %v", name, host, err)
	}
	if _, _, err := Normalize(Zone, "a.b"); err == nil {
		t.Fatal("nested label was accepted")
	}
	if _, _, err := Normalize(Zone, "-bad"); err == nil {
		t.Fatal("bad label was accepted")
	}
	if err := CheckPort(0); err == nil {
		t.Fatal("port 0 was accepted")
	}
	if err := CheckPort(65535); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertAndRemove(t *testing.T) {
	file := File{Zone: Zone}
	next, host, err := Upsert(file, "blog", 3000)
	if err != nil || host != "blog.ryangerardwilson.com" || len(next.Routes) != 1 {
		t.Fatalf("add: %+v %s %v", next, host, err)
	}
	next, _, err = Upsert(next, "blog", 4000)
	if err != nil || len(next.Routes) != 1 || next.Routes[0].Port != 4000 {
		t.Fatalf("replace: %+v %v", next, err)
	}
	next, err = Remove(next, "blog.ryangerardwilson.com")
	if err != nil || len(next.Routes) != 0 {
		t.Fatalf("remove: %+v %v", next, err)
	}
	if _, err := Remove(next, "blog"); err == nil {
		t.Fatal("missing route was removed")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.json")
	file, host, err := Upsert(Default(Zone), "lab", 9090)
	if err != nil {
		t.Fatal(err)
	}
	if host != "lab.ryangerardwilson.com" {
		t.Fatal(host)
	}
	if err := Save(path, file); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	loaded, err := Load(path)
	if err != nil || !Same(loaded, file) {
		t.Fatalf("loaded %+v %v", loaded, err)
	}
}

func TestMissingFileIsTheApex(t *testing.T) {
	file, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Routes) != 1 || file.Routes[0].Name != Apex || file.Routes[0].Port != 8080 {
		t.Fatalf("%+v", file)
	}
}
