// Package accept stores one-line justifications for a function score that
// stays over a limit. A later ratchet passes that score and no higher.
package accept

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

// FileName is the record kept at the git root.
const FileName = ".complexity-accept.json"

// Record is one accepted score.
type Record struct {
	File       string `json:"file"`
	Function   string `json:"function"`
	Metric     string `json:"metric"`
	Score      int    `json:"score"`
	Reason     string `json:"reason"`
	AcceptedAt string `json:"accepted_at"`
}

// Book is every record in one file.
type Book struct {
	Accepted []Record `json:"accepted"`
}

// Load reads root's record. A missing file is an empty book.
func Load(root string) (Book, error) {
	var b Book
	if root == "" {
		return b, nil
	}
	buf, err := os.ReadFile(filepath.Join(root, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return b, nil
	}
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(buf, &b)
	return b, err
}

// Save writes the book sorted by file, function, and metric.
func Save(root string, b Book) error {
	sort.Slice(b.Accepted, func(i, j int) bool { return less(b.Accepted[i], b.Accepted[j]) })
	buf, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, FileName), append(buf, '\n'), 0o644)
}

func less(a, b Record) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Function != b.Function {
		return a.Function < b.Function
	}
	return a.Metric < b.Metric
}

// Put replaces the record for the same file, function, and metric.
func (b *Book) Put(r Record) {
	for i, old := range b.Accepted {
		if old.File == r.File && old.Function == r.Function && old.Metric == r.Metric {
			b.Accepted[i] = r
			return
		}
	}
	b.Accepted = append(b.Accepted, r)
}

// Find returns the record for one function metric.
func (b Book) Find(file, function, metric string) (Record, bool) {
	for _, r := range b.Accepted {
		if r.File == file && r.Function == function && r.Metric == metric {
			return r, true
		}
	}
	return Record{}, false
}

// Merge joins books from several git roots.
func Merge(books ...Book) Book {
	var out Book
	for _, b := range books {
		out.Accepted = append(out.Accepted, b.Accepted...)
	}
	return out
}
