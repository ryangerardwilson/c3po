package execgen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/files"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
)

// Change is one path altered by a generator relative to pre-run state.
type Change struct {
	Path   string `json:"path"`
	Status string `json:"status"` // created|modified|deleted|created_ignored|modified_ignored|deleted_ignored
	Hash   string `json:"hash,omitempty"`
}

// Report is the result of a trusted generator run.
type Report struct {
	Root             string   `json:"root"`
	Command          []string `json:"command"`
	ExitCode         int      `json:"exit_code"`
	BeforePorcelain  string   `json:"before_porcelain"`
	AfterPorcelain   string   `json:"after_porcelain"`
	PreExistingDirty []string `json:"pre_existing_dirty,omitempty"`
	Changes          []Change `json:"changes"`
	IgnoredObserved  []Change `json:"ignored_observed,omitempty"`
	AllowedBy        string   `json:"allowed_by,omitempty"`
	Enforced         bool     `json:"enforced"`
	AuditComplete    bool     `json:"audit_complete"`
	AuditNotes       []string `json:"audit_notes,omitempty"`
}

// Run executes argv under root after verifying allowlist; returns report.
func Run(root string, cfg config.Config, argv []string, enforced bool) (Report, error) {
	rep := Report{Root: root, Command: argv, Enforced: enforced}
	if len(argv) == 0 {
		return rep, fmt.Errorf("exec requires a command after --")
	}
	ok, by := Allowed(cfg, argv)
	if !ok {
		return rep, fmt.Errorf("command not in generators.allow (structural prefix from argv[0]): %q", strings.Join(argv, " "))
	}
	rep.AllowedBy = by

	beforeText, beforeErr := measure.GitPorcelain(root)
	before := snapFromPorcelain(root, beforeText)
	if beforeErr != nil {
		rep.AuditNotes = append(rep.AuditNotes, "before porcelain unavailable: "+beforeErr.Error())
	}
	rep.BeforePorcelain = beforeText
	for p := range before {
		rep.PreExistingDirty = append(rep.PreExistingDirty, p)
	}
	sort.Strings(rep.PreExistingDirty)

	// Content identity of pre-existing paths detects changes even when their
	// porcelain status remains unchanged or returns to clean.
	beforeHash := map[string]string{}
	for p, e := range before {
		if e.Hash != "" {
			beforeHash[p] = e.Hash
		}
	}
	beforeIgnored, beforeIgnoredErr := ignoredSnapshot(root)
	if beforeIgnoredErr != nil {
		rep.AuditNotes = append(rep.AuditNotes, "before ignored snapshot unavailable: "+beforeIgnoredErr.Error())
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	runErr := cmd.Run()
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			rep.ExitCode = ee.ExitCode()
		} else {
			rep.ExitCode = 1
			return rep, runErr
		}
	}

	afterText, afterErr := measure.GitPorcelain(root)
	after := snapFromPorcelain(root, afterText)
	if afterErr != nil {
		rep.AuditNotes = append(rep.AuditNotes, "after porcelain unavailable: "+afterErr.Error())
	}
	rep.AfterPorcelain = afterText
	afterIgnored, afterIgnoredErr := ignoredSnapshot(root)
	if afterIgnoredErr != nil {
		rep.AuditNotes = append(rep.AuditNotes, "after ignored snapshot unavailable: "+afterIgnoredErr.Error())
	}
	rep.AuditComplete = beforeErr == nil && afterErr == nil && beforeIgnoredErr == nil && afterIgnoredErr == nil
	if !rep.AuditComplete {
		rep.AuditNotes = append(rep.AuditNotes, "audit incomplete without before/after git and ignored snapshots")
	}

	rep.Changes = delta(root, before, after, beforeHash)
	rep.IgnoredObserved = ignoredDelta(beforeIgnored, afterIgnored)
	if len(rep.IgnoredObserved) > 0 {
		rep.AuditNotes = append(rep.AuditNotes, "ignored paths reported as before/after content deltas (not hash-guarded by default)")
	}

	return rep, nil
}

type entry struct {
	Status string
	Hash   string
}

func snapFromPorcelain(root, text string) map[string]entry {
	m := map[string]entry{}
	for path, st := range measure.ParsePorcelainPaths(text) {
		e := entry{Status: st}
		if !strings.Contains(st, "D") {
			if h, err := files.HashFile(filepath.Join(root, path)); err == nil {
				e.Hash = h
			}
		}
		m[path] = e
	}
	return m
}

func delta(root string, before, after map[string]entry, beforeHash map[string]string) []Change {
	var out []Change
	for path, ae := range after {
		be, ok := before[path]
		if !ok {
			// new dirty path
			c := Change{Path: path, Status: "created", Hash: ae.Hash}
			if c.Hash == "" {
				if h, err := files.HashFile(filepath.Join(root, path)); err == nil {
					c.Hash = h
				}
			}
			// if it was untracked and still untracked with same hash - shouldn't be in before
			out = append(out, c)
			continue
		}
		// pre-existing dirty: only report if content hash changed
		bh := beforeHash[path]
		if bh == "" {
			bh = be.Hash
		}
		ah := ae.Hash
		if ah == "" {
			if h, err := files.HashFile(filepath.Join(root, path)); err == nil {
				ah = h
			}
		}
		if bh != "" && ah != "" && bh != ah {
			out = append(out, Change{Path: path, Status: "modified", Hash: ah})
		}
		// same content: do not attribute to generator
	}
	for path, be := range before {
		if _, ok := after[path]; ok {
			continue
		}
		abs := filepath.Join(root, path)
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			out = append(out, Change{Path: path, Status: "deleted"})
			continue
		}
		// The path may have returned to a clean state. Compare its current
		// content with the dirty pre-run snapshot before omitting it.
		if ah, err := files.HashFile(abs); err == nil && be.Hash != "" && ah != be.Hash {
			out = append(out, Change{Path: path, Status: "modified", Hash: ah})
		}
	}
	// Brand-new files committed by a generator are outside porcelain after the
	// run; AuditComplete covers the documented git working-tree surface only.
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func ignoredSnapshot(root string) (map[string]entry, error) {
	cmd := exec.Command("git", "-C", root, "status", "--porcelain=v1", "--ignored", "-uall")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	all := snapFromPorcelain(root, string(out))
	ignored := make(map[string]entry)
	for path, e := range all {
		if strings.Contains(e.Status, "!") {
			ignored[path] = e
		}
	}
	return ignored, nil
}

func ignoredDelta(before, after map[string]entry) []Change {
	var changes []Change
	for path, ae := range after {
		be, existed := before[path]
		switch {
		case !existed:
			changes = append(changes, Change{Path: path, Status: "created_ignored", Hash: ae.Hash})
		case be.Hash != "" && ae.Hash != "" && be.Hash != ae.Hash:
			changes = append(changes, Change{Path: path, Status: "modified_ignored", Hash: ae.Hash})
		}
	}
	for path := range before {
		if _, exists := after[path]; !exists {
			changes = append(changes, Change{Path: path, Status: "deleted_ignored"})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes
}
