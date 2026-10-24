package looptab

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// job is one row from `looptab now`.
type job struct {
	Index    int
	ID       string
	Schedule string
	Kind     string
	CWD      string
	Action   string
}

func (j job) label() string {
	return j.Schedule + "  " + j.Kind + "  " + j.Action
}

func loadJobs(bin string) ([]job, error) {
	if bin == "" {
		path, err := exec.LookPath("looptab")
		if err != nil {
			return nil, errors.New("looptab is not on PATH")
		}
		bin = path
	}
	cmd := exec.Command(bin, "now")
	cmd.Stdin = bytes.NewReader(nil)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	jobs, parseErr := parseJobList(stdout.String())
	if parseErr == nil {
		return jobs, nil
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i]
		}
		return nil, errors.New(msg)
	}
	return nil, parseErr
}

func parseJobList(text string) ([]job, error) {
	if strings.Contains(text, "No jobs are configured.") {
		return nil, nil
	}
	if !strings.Contains(text, "Registered jobs:") {
		return nil, errors.New("looptab did not list jobs")
	}
	var jobs []job
	for _, line := range strings.Split(text, "\n") {
		row, ok, err := parseJobLine(line)
		if err != nil {
			return nil, err
		}
		if ok {
			jobs = append(jobs, row)
		}
	}
	return jobs, nil
}

func parseJobLine(line string) (job, bool, error) {
	if !strings.HasPrefix(line, "  [") {
		return job{}, false, nil
	}
	end := strings.Index(line, "] ")
	if end < 4 {
		return job{}, false, fmt.Errorf("bad job line: %s", line)
	}
	index, err := strconv.Atoi(line[3:end])
	if err != nil {
		return job{}, false, fmt.Errorf("bad job index: %s", line)
	}
	parts := strings.SplitN(line[end+2:], "  ", 5)
	if len(parts) != 5 || parts[0] == "" {
		return job{}, false, fmt.Errorf("bad job line: %s", line)
	}
	return job{
		Index:    index,
		ID:       parts[0],
		Schedule: parts[1],
		Kind:     parts[2],
		CWD:      parts[3],
		Action:   parts[4],
	}, true, nil
}

func runNow(bin string, index int) error {
	cmd := exec.Command(bin, "now", strconv.Itoa(index))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
