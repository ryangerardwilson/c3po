package fsnav

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/xdg"
)

type Entry struct {
	Name   string
	Path   string
	IsDir  bool
	IsText bool
	Lines  int
}

type StructureSnapshot struct {
	Exists    bool
	Signature uint64
}

type PreviewRow struct {
	Entry     Entry
	Depth     int
	Prefix    string
	RootIndex int
	IsRoot    bool
}

const MaxPreviewDepth = 8

const (
	textIntensityThresholdLines = 200
	textIntensityMinimum        = 70
	textIntensityNeutral        = 180
	textIntensityMaximum        = 255
)

func NormalizePath(raw string) string {
	if parsed, err := url.Parse(raw); err == nil && parsed.Scheme == "file" {
		raw = parsed.Path
	}
	raw = os.ExpandEnv(raw)
	if raw == "~" || strings.HasPrefix(raw, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if raw == "~" {
				raw = home
			} else {
				raw = filepath.Join(home, raw[2:])
			}
		}
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return raw
	}
	real, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return real
	}
	return abs
}

func List(dir, filter string, showHidden bool) []Entry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]Entry, 0, len(entries))
	filter = strings.ToLower(strings.TrimSpace(filter))
	for _, entry := range entries {
		name := entry.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if filter != "" && !matches(name, filter) {
			continue
		}
		child := Entry{Name: name, Path: filepath.Join(dir, name), IsDir: entry.IsDir()}
		if !child.IsDir && entry.Type().IsRegular() {
			child.Lines, child.IsText = analyzeFile(child.Path)
		}
		out = append(out, child)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func SnapshotStructure(dir, filter string, showHidden bool, maxDepth int) StructureSnapshot {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return StructureSnapshot{}
	}
	if maxDepth > MaxPreviewDepth {
		maxDepth = MaxPreviewDepth
	}
	if maxDepth < 0 {
		maxDepth = 0
	}

	hash := fnv.New64a()
	_, _ = hash.Write([]byte("o-structure-v1\n"))
	appendStructureEntries(hash, dir, "", filter, showHidden, maxDepth)
	return StructureSnapshot{Exists: true, Signature: hash.Sum64()}
}

func appendStructureEntries(hash hash.Hash64, dir, prefix, filter string, showHidden bool, maxDepth int) {
	entries := structureEntries(dir, filter, showHidden)
	for _, entry := range entries {
		relative := prefix + entry.Name()
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			_, _ = fmt.Fprintf(hash, "%s\x00unknown\n", relative)
			continue
		}

		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		_, _ = fmt.Fprintf(
			hash,
			"%s\x00%s\x00%d\x00%d\n",
			relative,
			kind,
			info.Size(),
			info.ModTime().UnixNano(),
		)
		if entry.IsDir() && maxDepth > 0 {
			appendStructureEntries(hash, path, relative+"/", "", showHidden, maxDepth-1)
		}
	}
}

func structureEntries(dir, filter string, showHidden bool) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]os.DirEntry, 0, len(entries))
	filter = strings.ToLower(strings.TrimSpace(filter))
	for _, entry := range entries {
		name := entry.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if filter != "" && !matches(name, filter) {
			continue
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir() != out[j].IsDir() {
			return out[i].IsDir()
		}
		return strings.ToLower(out[i].Name()) < strings.ToLower(out[j].Name())
	})
	return out
}

func ListPreview(dir, filter string, showHidden bool, maxDepth int) []PreviewRow {
	if maxDepth > MaxPreviewDepth {
		maxDepth = MaxPreviewDepth
	}
	if maxDepth < 0 {
		maxDepth = 0
	}

	roots := List(dir, filter, showHidden)
	rows := make([]PreviewRow, 0, len(roots))
	for index, root := range roots {
		row := PreviewRow{Entry: root, RootIndex: index, IsRoot: true}
		rows = append(rows, row)
		if root.IsDir && maxDepth > 0 {
			appendPreviewChildren(&rows, root.Path, 1, maxDepth, "", showHidden, index)
		}
	}
	return rows
}

func appendPreviewChildren(rows *[]PreviewRow, dir string, depth, maxDepth int, prefix string, showHidden bool, rootIndex int) {
	children := List(dir, "", showHidden)
	for index, child := range children {
		last := index == len(children)-1
		connector := "├── "
		childPrefix := prefix + "│   "
		if last {
			connector = "└── "
			childPrefix = prefix + "    "
		}
		row := PreviewRow{
			Entry:     child,
			Depth:     depth,
			Prefix:    prefix + connector,
			RootIndex: rootIndex,
		}
		*rows = append(*rows, row)
		if child.IsDir && depth < maxDepth {
			appendPreviewChildren(rows, child.Path, depth+1, maxDepth, childPrefix, showHidden, rootIndex)
		}
	}
}

func analyzeFile(path string) (int, bool) {
	file, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	lines := 0
	seenBytes := false
	lastByte := byte('\n')
	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			seenBytes = true
			if bytes.IndexByte(chunk, 0) >= 0 {
				return 0, false
			}
			lines += bytes.Count(chunk, []byte{'\n'})
			lastByte = chunk[n-1]
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		return 0, false
	}
	if seenBytes && lastByte != '\n' {
		lines++
	}
	return lines, true
}

func TextIntensity(lines, maxLines, textCount int) int {
	if textCount <= 0 {
		return textIntensityMaximum
	}
	if maxLines <= textIntensityThresholdLines {
		return textIntensityNeutral
	}
	if lines <= textIntensityThresholdLines {
		return textIntensityMinimum
	}

	minLog := math.Log1p(float64(textIntensityThresholdLines))
	maxLog := math.Log1p(float64(maxLines))
	valueLog := math.Log1p(float64(lines))
	ratio := (valueLog - minLog) / (maxLog - minLog)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return textIntensityMinimum + int(math.Round(ratio*float64(textIntensityMaximum-textIntensityMinimum)))
}

func matches(name, filter string) bool {
	name = strings.ToLower(name)
	if ok, _ := filepath.Match(filter, name); ok {
		return true
	}
	return strings.HasPrefix(name, filter) || strings.Contains(name, filter)
}

func IsTextLike(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md", ".markdown", ".py", ".go", ".js", ".ts", ".tsx", ".jsx", ".json", ".yaml", ".yml", ".toml", ".sh", ".bash", ".zsh", ".css", ".html", ".xml", ".sql", ".c", ".h", ".cpp", ".rs", ".ex", ".exs":
		return true
	default:
		return false
	}
}

func CacheSelectionPath() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = "."
		}
		base = filepath.Join(home, ".cache")
	}
	dest := filepath.Join(xdg.App(base, "o"), "picker-selection.txt")
	_ = xdg.Adopt(dest, filepath.Join(base, "o", "picker-selection.txt"))
	return dest
}

func WriteSelectionCache(paths []string) error {
	path := CacheSelectionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := strings.Join(paths, "\n")
	if body != "" {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0o644)
}
