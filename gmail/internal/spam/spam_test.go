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
	if len(groups) != 1 || Query(groups[0]) != "from:clickup.com" {
		t.Fatalf("%v", groups)
	}
	plain := CleanupSenders([]string{"spam@evil.test"}, nil)
	if len(plain) != 1 || plain[0] != "spam@evil.test" {
		t.Fatalf("plain %v", plain)
	}
	ranked := Rank(map[string]int{"b@bulk.test": 1, "a@bulk.test": 3, "spam@evil.test": 9}, []string{"spam@evil.test"}, nil)
	if len(ranked) != 2 || ranked[0].Sender != "a@bulk.test" || ranked[0].Unread != 3 || ranked[1].Sender != "b@bulk.test" {
		t.Fatalf("rank %v", ranked)
	}
}
