package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
)

func TestPresetItemsAreNumericAndNamed(t *testing.T) {
	cfg := &config.AppConfig{Accounts: map[string]*config.Account{
		"2": {Email: "d@e.f"},
		"1": {Email: "a@b.c"},
		"3": {},
	}}
	items := presetItems(cfg)
	if len(items) != 3 || items[0].ID != "1" || items[0].Label != "1  a@b.c" || items[2].Label != "3" {
		t.Fatalf("%#v", items)
	}
}

func TestScreenReportsNoPresets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("GDRIVE_CONFIG", filepath.Join(home, "gdrive.json"))
	var stdout, stderr bytes.Buffer
	code := Screen(&stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "no Drive presets") {
		t.Fatalf("%d %s %s", code, stdout.String(), stderr.String())
	}
}

func TestBrowseAndUploadAreNotCommands(t *testing.T) {
	for _, args := range [][]string{{"1", "browse"}, {"1", "upload", "a.txt"}} {
		err := Dispatch(args, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("%v %v", args, err)
		}
	}
}
