package looptab

import "testing"

func TestParseJobListKeepsSpacedSchedules(t *testing.T) {
	text := "" +
		"Registered jobs:\n" +
		"  [0] abcd1234  daily 5am  grok  ~/Work  @grok \"Check mail\"\n" +
		"  [1] 00abcdef  hourly at 15  command  ~  c3po gmail spam clean\n"
	jobs, err := parseJobList(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs %d", len(jobs))
	}
	if jobs[0].Index != 0 || jobs[0].ID != "abcd1234" || jobs[0].Schedule != "daily 5am" || jobs[0].Kind != "grok" || jobs[0].Action != `@grok "Check mail"` {
		t.Fatalf("first %+v", jobs[0])
	}
	if jobs[1].Index != 1 || jobs[1].Schedule != "hourly at 15" || jobs[1].Kind != "command" || jobs[1].Action != "c3po gmail spam clean" {
		t.Fatalf("second %+v", jobs[1])
	}
}

func TestParseJobListTreatsAnEmptyFileAsNoJobs(t *testing.T) {
	jobs, err := parseJobList("No jobs are configured.\n")
	if err != nil || len(jobs) != 0 {
		t.Fatalf("jobs %v err %v", jobs, err)
	}
}

func TestParseJobLineKeepsDoubleSpacesInTheAction(t *testing.T) {
	line := "  [0] 18c57fbc  daily 5am,9am,12pm,4pm  grok  ~  @grok \"a  b\""
	row, ok, err := parseJobLine(line)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if row.Schedule != "daily 5am,9am,12pm,4pm" || row.Kind != "grok" || row.CWD != "~" || row.Action != "@grok \"a  b\"" {
		t.Fatalf("%+v", row)
	}
}

func TestParseJobListRejectsAMissingHeader(t *testing.T) {
	if _, err := parseJobList("looptab file not found\n"); err == nil {
		t.Fatal("expected an error")
	}
}
