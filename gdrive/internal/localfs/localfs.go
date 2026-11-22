// Package localfs is the registered folder on disk, addressed by the
// slash-separated relpaths the syncer uses.
package localfs

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
)

// Tree is one registered folder. It is the syncer.LocalSource and
// syncer.LocalTarget the gdrive commands use.
type Tree struct {
	Root string
}

// Locate is the file path of relpath. The empty relpath is Root itself.
func (t Tree) Locate(relpath string) string {
	if relpath == "" {
		return t.Root
	}
	return filepath.Join(t.Root, filepath.FromSlash(relpath))
}

// Stat follows links. A missing entry is an error matching fs.ErrNotExist.
func (t Tree) Stat(relpath string) (syncer.LocalEntry, error) {
	info, err := os.Stat(t.Locate(relpath))
	if err != nil {
		return syncer.LocalEntry{}, err
	}
	return entry(relpath, info), nil
}

// Hash is the hex SHA-1 of the file at relpath.
func (t Tree) Hash(relpath string) (string, error) {
	file, err := os.Open(t.Locate(relpath))
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha1.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// MakeDir creates relpath and its parents.
func (t Tree) MakeDir(relpath string) error {
	return os.MkdirAll(t.Locate(relpath), 0o755)
}

// RelPath maps a file path under Root to its relpath, and reports false
// for a path outside Root.
func (t Tree) RelPath(path string) (string, bool) {
	rel, err := filepath.Rel(t.Root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// Scan walks Root without following links and lists every folder and
// file under it, unhashed.
func (t Tree) Scan() (map[string]syncer.LocalEntry, error) {
	result := map[string]syncer.LocalEntry{}
	err := filepath.WalkDir(t.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == t.Root {
			return nil
		}
		return t.scanEntry(path, d, result)
	})
	return result, err
}

func (t Tree) scanEntry(path string, d os.DirEntry, result map[string]syncer.LocalEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	rel, err := filepath.Rel(t.Root, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	result[rel] = entry(rel, info)
	return nil
}

func entry(relpath string, info os.FileInfo) syncer.LocalEntry {
	if info.IsDir() {
		return syncer.LocalEntry{RelPath: relpath, Kind: "dir", MTimeNS: info.ModTime().UnixNano()}
	}
	return syncer.LocalEntry{RelPath: relpath, Kind: "file", Size: info.Size(), MTimeNS: info.ModTime().UnixNano()}
}
