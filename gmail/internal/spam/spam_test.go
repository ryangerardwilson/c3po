package spam

import "testing"

func TestCandidatesAndGroups(t *testing.T) {
	counts := map[string]int{
		"a@evil.test": 6, "b@evil.test": 5, "me@example.com": 9,
		"x@gmail.com": 9, "ok@safe.test": 8,
	}
	got := Candidates(counts, []string{"ok@safe.test"}, nil, "me@example.com", 5)
	if len(got) != 1 || got[0].Sender != "a@evil.test" {
		t.Fatalf("%v", got)
	}
	senders := CleanupSenders([]string{"spam@evil.test", "@clickup.com"}, []string{"spam@evil.test"})
	groups := Groups(senders, 25)
	if len(groups) != 2 || Query(groups[0]) != "from:clickup.com" || Query(groups[1]) != "from:mail.clickup.com" {
		t.Fatalf("%v %q", groups, Query(groups[0]))
	}
}
