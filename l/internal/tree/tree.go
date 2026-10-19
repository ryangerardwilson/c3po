// Package tree builds and renders the compact tree for c3po l.
package tree

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Type is the kind of a node.
type Type string

const (
	TypeDirectory Type = "D"
	TypeFile      Type = "F"
	TypeBinary    Type = "B"
)

// Config controls which entries Build reads.
type Config struct {
	Root              string
	MaxDepth          int
	ShowHidden        bool
	FilesOnly         bool
	UseDefaultIgnores bool
	ExtraIgnores      []string
}

// Node is one entry in the tree.
type Node struct {
	Name     string
	Path     string
	Type     Type
	Size     int64
	Lines    int
	Children []*Node
}

// Stats holds the ranges used to scale color intensity.
type Stats struct {
	TextFileCount   int
	MinTextLines    int
	MaxTextLines    int
	BinaryFileCount int
	MinBinarySize   int64
	MaxBinarySize   int64
}

// defaultIgnoredDirs are skipped unless UseDefaultIgnores is false.
// A directory named internal is not in this list and is never hidden.
var defaultIgnoredDirs = map[string]struct{}{
	".cache":        {},
	".git":          {},
	".hg":           {},
	".mypy_cache":   {},
	".next":         {},
	".nuxt":         {},
	".parcel-cache": {},
	".pytest_cache": {},
	".ruff_cache":   {},
	".svn":          {},
	".turbo":        {},
	".venv":         {},
	".vite":         {},
	".vscode":       {},
	"__pycache__":   {},
	"bin":           {},
	"build":         {},
	"coverage":      {},
	"dist":          {},
	"env":           {},
	"node_modules":  {},
	"target":        {},
	"venv":          {},
}

var defaultIgnoredFiles = map[string]struct{}{
	".DS_Store": {},
}

// Build reads the tree under cfg.Root. With FilesOnly and no visible text
// file, it returns a nil node and no error.
func Build(cfg Config) (*Node, Stats, error) {
	root := cfg.Root
	if root == "" {
		root = "."
	}
	if cfg.MaxDepth < -1 {
		return nil, Stats{}, errors.New("depth must be -1 or greater")
	}

	info, err := os.Lstat(root)
	if err != nil {
		return nil, Stats{}, err
	}

	node := nodeForPath(root, displayName(root), info)
	if node.Type == TypeDirectory {
		if err := walkChildren(node, 0, cfg); err != nil {
			return nil, Stats{}, err
		}
	}
	if cfg.FilesOnly && !pruneFilesOnly(node) {
		return nil, Stats{}, nil
	}

	var stats Stats
	collectStats(node, &stats)
	return node, stats, nil
}

func displayName(path string) string {
	if path == "" {
		return "."
	}
	return filepath.Clean(path)
}

func nodeForPath(path string, name string, info os.FileInfo) *Node {
	node := &Node{
		Name: name,
		Path: path,
		Size: info.Size(),
	}
	if info.IsDir() {
		node.Type = TypeDirectory
		return node
	}

	lines, binary := analyzeFile(path)
	if binary {
		node.Type = TypeBinary
		return node
	}
	node.Type = TypeFile
	node.Lines = lines
	return node
}

func walkChildren(parent *Node, depth int, cfg Config) error {
	if cfg.MaxDepth >= 0 && depth >= cfg.MaxDepth {
		return nil
	}

	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		return err
	}
	sortEntries(entries)

	for _, entry := range entries {
		if err := addChild(parent, entry, depth, cfg); err != nil {
			return err
		}
	}
	return nil
}

// sortEntries puts directories first, then sorts by name without case.
func sortEntries(entries []os.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.IsDir() != right.IsDir() {
			return left.IsDir()
		}
		return strings.ToLower(left.Name()) < strings.ToLower(right.Name())
	})
}

func addChild(parent *Node, entry os.DirEntry, depth int, cfg Config) error {
	name := entry.Name()
	if shouldSkip(name, entry.IsDir(), cfg) {
		return nil
	}
	info, err := entry.Info()
	if err != nil {
		return nil
	}
	child := nodeForPath(filepath.Join(parent.Path, name), name, info)
	parent.Children = append(parent.Children, child)
	if child.Type != TypeDirectory {
		return nil
	}
	return walkChildren(child, depth+1, cfg)
}

func pruneFilesOnly(node *Node) bool {
	switch node.Type {
	case TypeFile:
		return true
	case TypeDirectory:
		kept := node.Children[:0]
		for _, child := range node.Children {
			if pruneFilesOnly(child) {
				kept = append(kept, child)
			}
		}
		node.Children = kept
		return len(node.Children) > 0
	default:
		return false
	}
}

func shouldSkip(name string, isDir bool, cfg Config) bool {
	if !cfg.ShowHidden && strings.HasPrefix(name, ".") {
		return true
	}
	if cfg.UseDefaultIgnores && defaultIgnored(name, isDir) {
		return true
	}
	for _, pattern := range cfg.ExtraIgnores {
		if matchesIgnore(pattern, name) {
			return true
		}
	}
	return false
}

func defaultIgnored(name string, isDir bool) bool {
	set := defaultIgnoredFiles
	if isDir {
		set = defaultIgnoredDirs
	}
	_, ok := set[name]
	return ok
}

func matchesIgnore(pattern string, name string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if !strings.ContainsAny(pattern, "*?[") {
		return pattern == name
	}
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

// analyzeFile counts lines. A file that cannot be read, or that holds a
// NUL byte, is binary. A final line without a newline still counts.
func analyzeFile(path string) (int, bool) {
	file, err := os.Open(path)
	if err != nil {
		return 0, true
	}
	defer file.Close()

	var scan lineScan
	buf := make([]byte, 32*1024)
	for {
		n, readErr := file.Read(buf)
		if n > 0 && scan.add(buf[:n]) {
			return 0, true
		}
		if errors.Is(readErr, io.EOF) {
			return scan.total(), false
		}
		if readErr != nil {
			return 0, true
		}
	}
}

type lineScan struct {
	lines    int
	seen     bool
	lastByte byte
}

// add reads one chunk and reports whether it holds a NUL byte.
func (s *lineScan) add(chunk []byte) bool {
	if bytes.IndexByte(chunk, 0) >= 0 {
		return true
	}
	s.seen = true
	s.lines += bytes.Count(chunk, []byte{'\n'})
	s.lastByte = chunk[len(chunk)-1]
	return false
}

func (s *lineScan) total() int {
	if s.seen && s.lastByte != '\n' {
		return s.lines + 1
	}
	return s.lines
}

func collectStats(node *Node, stats *Stats) {
	switch node.Type {
	case TypeFile:
		stats.addText(node.Lines)
	case TypeBinary:
		stats.addBinary(node.Size)
	}
	for _, child := range node.Children {
		collectStats(child, stats)
	}
}

func (s *Stats) addText(lines int) {
	s.TextFileCount++
	if s.TextFileCount == 1 || lines < s.MinTextLines {
		s.MinTextLines = lines
	}
	s.MaxTextLines = max(s.MaxTextLines, lines)
}

func (s *Stats) addBinary(size int64) {
	s.BinaryFileCount++
	if s.BinaryFileCount == 1 || size < s.MinBinarySize {
		s.MinBinarySize = size
	}
	s.MaxBinarySize = max(s.MaxBinarySize, size)
}
