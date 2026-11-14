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

// Check returns the generators.allow pattern that admits argv, or an error
// when none does. It reads nothing but cfg, so a caller can refuse a command
// before it measures or snapshots the tree.
func Check(cfg config.Config, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("exec requires a command after --")
	}
	ok, by := Allowed(cfg, argv)
	if !ok {
		return "", fmt.Errorf("command not in generators.allow (structural prefix from argv[0]): %q", strings.Join(argv, " "))
	}
	return by, nil
}

// Run executes argv under root after verifying allowlist; returns report.
func Run(root string, cfg config.Config, argv []string, enforced bool) (Report, error) {
	rep := Report{Root: root, Command: argv, Enforced: enforced}
	by, err := Check(cfg, argv)
	if err != nil {
		return rep, err
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
	rep.IgnoredObserved = ignoredDelta(root, beforeIgnored, afterIgnored)
	if len(rep.IgnoredObserved) > 0 {
		rep.AuditNotes = append(rep.AuditNotes, "ignored paths reported by before/after stat identity, hashed after the run (not hash-guarded by default)")
	}
	if trees := ignoredTrees(beforeIgnored, afterIgnored); len(trees) > 0 {
		rep.AuditNotes = append(rep.AuditNotes, "ignored trees reported only when created or removed, contents not hashed: "+strings.Join(trees, " "))
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

// ignoredEntry is the stat identity of one ignored file. Ignored files are
// not hashed before the run; one whose identity changed is hashed after it.
// tree marks an ignored dependency or build tree (node_modules, dist, .venv,
// ...) or a nested repository, whose contents are neither listed nor hashed.
type ignoredEntry struct {
	size  int64
	mtime int64
	mode  os.FileMode
	tree  bool
}

// ignoredSnapshot lists ignored paths with git's collapsed view, so a wholly
// ignored directory is one line rather than every file below it.
func ignoredSnapshot(root string) (map[string]ignoredEntry, error) {
	cmd := exec.Command("git", "-C", root, "status", "--porcelain=v1", "--ignored", "-unormal")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	snap := make(map[string]ignoredEntry)
	for path, st := range measure.ParsePorcelainPaths(string(out)) {
		if st != "!!" {
			continue
		}
		if tree, ok := ignoredTree(path); ok {
			snap[tree] = ignoredEntry{tree: true}
			continue
		}
		if strings.HasSuffix(path, "/") {
			walkIgnoredDir(root, strings.TrimSuffix(path, "/"), snap)
			continue
		}
		addIgnoredFile(snap, root, path)
	}
	return snap, nil
}

// ignoredTree returns the "dir/" prefix of rel that measure never descends
// into (node_modules, vendor, dist, .venv, ...), if any.
func ignoredTree(rel string) (string, bool) {
	parts := strings.Split(strings.TrimSuffix(rel, "/"), "/")
	for i, part := range parts {
		if measure.SkipDirName(part) && (i < len(parts)-1 || strings.HasSuffix(rel, "/")) {
			return strings.Join(parts[:i+1], "/") + "/", true
		}
	}
	return "", false
}

// walkIgnoredDir records the files under one collapsed ignored directory by
// stat. Dependency trees and nested repositories are recorded whole.
func walkIgnoredDir(root, rel string, snap map[string]ignoredEntry) {
	_ = filepath.WalkDir(filepath.Join(root, rel), func(abs string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		r, err := filepath.Rel(root, abs)
		if err != nil {
			return nil
		}
		r = filepath.ToSlash(r)
		if d.IsDir() {
			if r == rel {
				return nil
			}
			if _, err := os.Stat(filepath.Join(abs, ".git")); err == nil || measure.SkipDirName(d.Name()) {
				snap[r+"/"] = ignoredEntry{tree: true}
				return filepath.SkipDir
			}
			return nil
		}
		addIgnoredFile(snap, root, r)
		return nil
	})
}

func addIgnoredFile(snap map[string]ignoredEntry, root, rel string) {
	info, err := os.Lstat(filepath.Join(root, rel))
	if err != nil {
		return
	}
	snap[rel] = ignoredEntry{size: info.Size(), mtime: info.ModTime().UnixNano(), mode: info.Mode()}
}

// ignoredDelta reports ignored paths the generator created, changed, or
// removed. A file is hashed only when it is new or its stat identity changed.
func ignoredDelta(root string, before, after map[string]ignoredEntry) []Change {
	var changes []Change
	hash := func(path string, e ignoredEntry) string {
		if e.tree {
			return ""
		}
		h, _ := files.HashFile(filepath.Join(root, path))
		return h
	}
	for path, ae := range after {
		be, existed := before[path]
		switch {
		case !existed:
			changes = append(changes, Change{Path: path, Status: "created_ignored", Hash: hash(path, ae)})
		case ae.tree || be.tree:
		case be != ae:
			changes = append(changes, Change{Path: path, Status: "modified_ignored", Hash: hash(path, ae)})
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

// ignoredTrees lists the ignored trees whose contents were not audited.
func ignoredTrees(snaps ...map[string]ignoredEntry) []string {
	seen := map[string]bool{}
	var out []string
	for _, snap := range snaps {
		for path, e := range snap {
			if e.tree && !seen[path] {
				seen[path] = true
				out = append(out, path)
			}
		}
	}
	sort.Strings(out)
	return out
}
