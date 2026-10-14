package measure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/version"
)

const cacheMaxAge = 60 * time.Second

type cacheEntry struct {
	Root               string    `json:"root"`
	Fingerprint        string    `json:"fingerprint"`
	LOC                int       `json:"loc"`
	FileCount          int       `json:"file_count"`
	ProseLOC           int       `json:"prose_loc"`
	NestedReposSkipped int       `json:"nested_repos_skipped"`
	GitIgnoreActive    bool      `json:"gitignore_active"`
	StoredAt           time.Time `json:"stored_at"`
}

// CountOpts controls cache behavior.
type CountOpts struct {
	Refresh bool
}

// CountCached returns Count results, using the XDG cache when valid.
func CountCached(root string, cfg config.Config) (Result, error) {
	return CountCachedOpts(root, cfg, CountOpts{})
}

// CountCachedOpts is CountCached with options.
func CountCachedOpts(root string, cfg config.Config, opts CountOpts) (Result, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	fp, err := fingerprint(root, cfg)
	if err != nil {
		return Count(root, cfg)
	}
	if !opts.Refresh {
		if ent, ok := loadCache(cfg, root, fp); ok {
			age := time.Since(ent.StoredAt).Milliseconds()
			return Result{
				Root:               root,
				LOC:                ent.LOC,
				FileCount:          ent.FileCount,
				ProseLOC:           ent.ProseLOC,
				NestedReposSkipped: ent.NestedReposSkipped,
				GitIgnoreActive:    ent.GitIgnoreActive,
				CacheHit:           true,
				CacheAgeMs:         age,
			}, nil
		}
	}
	res, err := Count(root, cfg)
	if err != nil {
		return res, err
	}
	_ = storeCache(cfg, cacheEntry{
		Root:               root,
		Fingerprint:        fp,
		LOC:                res.LOC,
		FileCount:          res.FileCount,
		ProseLOC:           res.ProseLOC,
		NestedReposSkipped: res.NestedReposSkipped,
		GitIgnoreActive:    res.GitIgnoreActive,
		StoredAt:           time.Now().UTC(),
	})
	res.CacheHit = false
	res.CacheAgeMs = 0
	return res, nil
}

func fingerprint(root string, cfg config.Config) (string, error) {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "root=%s\n", root)
	_, _ = fmt.Fprintf(h, "v=4\n") // scope rules plus parser/graph binary identity
	_, _ = fmt.Fprintf(h, "binary=%s\nsource=%s\n", version.String(), version.SourceHash)
	_, _ = fmt.Fprintf(h, "threshold=%d\n", cfg.ThresholdLOC)
	_, _ = fmt.Fprintf(h, "include=%s\n", strings.Join(cfg.Include, "\x00"))
	_, _ = fmt.Fprintf(h, "exclude=%s\n", strings.Join(cfg.Exclude, "\x00"))
	_, _ = fmt.Fprintf(h, "prose=%s\ncount_prose=%v\n", strings.Join(cfg.Prose, "\x00"), cfg.Enforcement.CountProse)
	st, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(h, "root_mtime=%d\n", st.ModTime().UnixNano())
	// gitignore mtime
	for _, p := range []string{
		filepath.Join(root, ".gitignore"),
		filepath.Join(root, ".git", "info", "exclude"),
	} {
		if st, err := os.Stat(p); err == nil {
			_, _ = fmt.Fprintf(h, "%s=%d\n", p, st.ModTime().UnixNano())
		}
	}
	headPath := filepath.Join(root, ".git", "HEAD")
	if data, err := os.ReadFile(headPath); err == nil {
		_, _ = h.Write(data)
		ref := strings.TrimSpace(string(data))
		if strings.HasPrefix(ref, "ref: ") {
			refPath := filepath.Join(root, ".git", strings.TrimSpace(strings.TrimPrefix(ref, "ref: ")))
			if refData, err := os.ReadFile(refPath); err == nil {
				_, _ = h.Write(refData)
			}
			if st, err := os.Stat(refPath); err == nil {
				_, _ = fmt.Fprintf(h, "ref_mtime=%d\n", st.ModTime().UnixNano())
			}
		}
	} else {
		if data, err := os.ReadFile(filepath.Join(root, ".git")); err == nil && strings.HasPrefix(string(data), "gitdir:") {
			_, _ = h.Write(data)
		}
	}
	// Working-tree dirty fingerprint includes porcelain + content hashes of
	// dirty/untracked paths so repeated edits while already dirty invalidate.
	if dirty := gitDirtyFingerprint(root); len(dirty) > 0 {
		_, _ = h.Write(dirty)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func cacheFilePath(cfg config.Config, root string) (string, error) {
	dir, err := cfg.CacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(root))
	return filepath.Join(dir, "measure", hex.EncodeToString(sum[:])+".json"), nil
}

func loadCache(cfg config.Config, root, fp string) (cacheEntry, bool) {
	path, err := cacheFilePath(cfg, root)
	if err != nil {
		return cacheEntry{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheEntry{}, false
	}
	var ent cacheEntry
	if err := json.Unmarshal(data, &ent); err != nil {
		return cacheEntry{}, false
	}
	if ent.Fingerprint != fp || ent.Root != root {
		return cacheEntry{}, false
	}
	if time.Since(ent.StoredAt) > cacheMaxAge {
		return cacheEntry{}, false
	}
	return ent, true
}

func storeCache(cfg config.Config, ent cacheEntry) error {
	path, err := cacheFilePath(cfg, ent.Root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(ent)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
