package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath joins root with path, requiring the result to stay under root.
func ResolvePath(root, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Clean(filepath.Join(root, path))
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %q escapes workspace root", path)
	}
	return abs, nil
}

// IsBinary reports whether the file looks binary (contains a NUL in the first 512KiB).
func IsBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 512*1024)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true, nil
		}
	}
	return false, nil
}

// HashFile returns the SHA-256 hex digest of a file.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ReadLines returns inclusive 1-based line range as a single string with newlines.
func ReadLines(path string, start, end int) (string, error) {
	if start < 1 || end < start {
		return "", fmt.Errorf("invalid line range %d-%d", start, end)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := splitKeepEnds(string(data))
	if start > len(lines) {
		return "", fmt.Errorf("start line %d past end of file (%d lines)", start, len(lines))
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start-1:end], ""), nil
}

// LineCount returns number of lines (same rules as measure for non-empty).
func LineCount(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	n := strings.Count(string(data), "\n")
	if data[len(data)-1] != '\n' {
		n++
	}
	return n, nil
}

// PatchExact replaces exactly one occurrence of old with new when hash matches.
func PatchExact(path, expectHash, old, new string) (newHash string, err error) {
	return PatchOps(path, expectHash, []Op{{Old: old, New: new}})
}

// FormatReadRange returns content for [start,end] with optional line numbers and header.
// If end-start+1 exceeds maxLines and strict is false, clamps and returns nextStart > 0.
// With strict, a range longer than maxLines or past the last line fails.
// The header names the total line count and, when clamped, the next range.
func FormatReadRange(path, rel string, start, end, maxLines int, number, strict bool) (text string, nextStart int, err error) {
	if start < 1 || end < start {
		return "", 0, fmt.Errorf("invalid line range %d-%d", start, end)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	lines := splitKeepEnds(string(data))
	total := len(lines)
	if start > total {
		return "", 0, fmt.Errorf("start line %d past end of file (%d lines)", start, total)
	}
	if end > total {
		if strict {
			return "", 0, fmt.Errorf("line range %d-%d passes end of file (%d lines)", start, end, total)
		}
		end = total
	}
	span := end - start + 1
	if span > maxLines {
		if strict {
			return "", 0, fmt.Errorf("line range length %d exceeds max_read_lines %d", span, maxLines)
		}
		end = start + maxLines - 1
		if end > total {
			end = total
		}
		nextStart = end + 1
		if nextStart > total {
			nextStart = 0
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s lines %d-%d of %d", rel, start, end, total)
	if nextStart > 0 {
		fmt.Fprintf(&b, " (capped at %d; next: --lines %d-%d)", maxLines, nextStart, min(nextStart+maxLines-1, total))
	}
	b.WriteByte('\n')
	for i := start - 1; i < end; i++ {
		line := lines[i]
		if number {
			// strip trailing newline for numbering, re-add
			body := strings.TrimSuffix(line, "\n")
			body = strings.TrimSuffix(body, "\r")
			fmt.Fprintf(&b, "%6d\t%s\n", i+1, body)
		} else {
			b.WriteString(line)
			if !strings.HasSuffix(line, "\n") && i == end-1 {
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), nextStart, nil
}

func fileMode(path string) os.FileMode {
	st, err := os.Stat(path)
	if err != nil {
		return 0o644
	}
	return st.Mode().Perm()
}

func splitKeepEnds(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
