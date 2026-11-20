package cli

import (
	"strings"
	"testing"
)

func TestTimerScriptCallsC3PONotify(t *testing.T) {
	script := buildTimerServiceScript("c3po gdrive sync run")
	bin := shellQuote(c3poBin())
	for _, want := range []string{
		bin + " notify heading 'gdrive' body 'Hourly backup started';",
		bin + " notify heading 'gdrive' body 'Hourly backup finished successfully';",
		bin + " notify --urgency critical heading 'gdrive' body 'Hourly backup failed';",
		`exit "$rc";`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q\n%s", want, script)
		}
	}
	for _, gone := range []string{"notify()", "notify-send", "quickshell"} {
		if strings.Contains(script, gone) {
			t.Fatalf("script still has %q\n%s", gone, script)
		}
	}
}
