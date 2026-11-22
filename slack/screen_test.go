package slack

import (
	"bytes"
	"strings"
	"testing"
)

func TestScreenHelpNamesTheKeys(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Screen([]string{"help"}, &out, &errb); code != 0 {
		t.Fatalf("code %d\n%s", code, errb.String())
	}
	for _, want := range []string{"r2d2 slack", "h and l switch workspace", "Opening an item does not mark it read", "syncs the desk from Slack", "syncs about every 10 minutes", "last synced", "token is not shown"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q\n%s", want, out.String())
		}
	}
	out.Reset()
	if code := Screen([]string{"version"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "c3po 0.5.0") {
		t.Fatalf("version %d\n%s", code, out.String())
	}
	errb.Reset()
	if code := Screen([]string{"extra"}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "r2d2 slack") {
		t.Fatalf("extra %d\n%s", code, errb.String())
	}
}
