package fsnav

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSortsDirectoriesFirstAndFilters(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := List(root, "", false)
	if len(entries) != 2 || !entries[0].IsDir || entries[0].Name != "docs" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
	filtered := List(root, "*.md", false)
	if len(filtered) != 1 || filtered[0].Name != "note.md" {
		t.Fatalf("unexpected filtered entries: %#v", filtered)
	}
}

func TestWriteSelectionCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)
	if err := WriteSelectionCache([]string{"/tmp/a", "/tmp/b"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(tmp, "c3po", "o", "picker-selection.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "/tmp/a\n/tmp/b\n" {
		t.Fatalf("cache = %q", string(body))
	}
}

func TestListControlsDotEntryVisibility(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, entry := range List(root, "", false) {
		if strings.HasPrefix(entry.Name, ".") {
			t.Fatalf("hidden dot entry returned: %#v", entry)
		}
	}

	visible := List(root, "", true)
	names := make(map[string]bool, len(visible))
	for _, entry := range visible {
		names[entry.Name] = true
	}
	for _, name := range []string{".config", ".env", "note.md"} {
		if !names[name] {
			t.Fatalf("expected %q in %#v", name, names)
		}
	}

	if got := List(root, ".env", false); len(got) != 0 {
		t.Fatalf("filter bypassed hidden state: %#v", got)
	}
	if got := List(root, ".env", true); len(got) != 1 || got[0].Name != ".env" {
		t.Fatalf("visible filtered entries = %#v", got)
	}
}

func TestListPreviewBuildsTreeRowsByDepth(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	sub := filepath.Join(docs, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "guide.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "deep.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	depthZero := ListPreview(root, "", false, 0)
	if len(depthZero) != 2 || !depthZero[0].IsRoot || depthZero[0].Entry.Name != "docs" {
		t.Fatalf("depth-zero preview = %#v", depthZero)
	}

	depthOne := ListPreview(root, "", false, 1)
	if got := previewNames(depthOne); strings.Join(got, ",") != "docs,sub,guide.md,note.md" {
		t.Fatalf("depth-one names = %#v", got)
	}
	if depthOne[1].Depth != 1 || depthOne[1].Prefix != "├── " || depthOne[1].RootIndex != 0 {
		t.Fatalf("unexpected first child: %#v", depthOne[1])
	}
	if depthOne[2].Prefix != "└── " {
		t.Fatalf("unexpected last child prefix: %#v", depthOne[2])
	}

	depthTwo := ListPreview(root, "", false, 2)
	deep := depthTwo[2]
	if deep.Entry.Name != "deep.md" || deep.Depth != 2 || deep.Prefix != "│   └── " {
		t.Fatalf("unexpected deep row: %#v", deep)
	}
}

func TestListPreviewRespectsHiddenEntriesAndDepthCap(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hidden", ".secret"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	hidden := ListPreview(root, "", false, 1)
	if len(hidden) != 0 {
		t.Fatalf("hidden descendants leaked: %#v", hidden)
	}
	visible := ListPreview(root, "", true, 1)
	if len(visible) != 2 || visible[0].Entry.Name != ".hidden" || visible[1].Entry.Name != ".secret" {
		t.Fatalf("visible dot preview = %#v", visible)
	}

	deepRoot := root
	for i := 0; i < MaxPreviewDepth+2; i++ {
		deepRoot = filepath.Join(deepRoot, fmt.Sprintf("d%d", i))
	}
	if err := os.MkdirAll(deepRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	rows := ListPreview(root, "", false, MaxPreviewDepth+2)
	maxDepth := 0
	for _, row := range rows {
		if row.Depth > maxDepth {
			maxDepth = row.Depth
		}
	}
	if maxDepth != MaxPreviewDepth {
		t.Fatalf("max preview depth = %d, want %d", maxDepth, MaxPreviewDepth)
	}
}

func previewNames(rows []PreviewRow) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Entry.Name)
	}
	return names
}

func TestListAnalyzesTextLinesAndBinaryContent(t *testing.T) {
	root := t.TempDir()
	three := filepath.Join(root, "three.txt")
	partial := filepath.Join(root, "partial.txt")
	binary := filepath.Join(root, "image.bin")
	if err := os.WriteFile(three, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(partial, []byte("one\ntwo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}

	entries := List(root, "", false)
	byName := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		byName[entry.Name] = entry
	}
	if got := byName["three.txt"]; !got.IsText || got.Lines != 3 {
		t.Fatalf("three-line metadata = %#v", got)
	}
	if got := byName["partial.txt"]; !got.IsText || got.Lines != 2 {
		t.Fatalf("partial-line metadata = %#v", got)
	}
	if got := byName["image.bin"]; got.IsText || got.Lines != 0 {
		t.Fatalf("binary metadata = %#v", got)
	}

	preview := ListPreview(root, "", false, 1)
	for _, row := range preview {
		if row.Entry.Name == "partial.txt" && (!row.Entry.IsText || row.Entry.Lines != 2) {
			t.Fatalf("preview metadata missing: %#v", row)
		}
	}
}

func TestTextIntensityUsesLRange(t *testing.T) {
	tests := []struct {
		name      string
		lines     int
		maxLines  int
		count     int
		intensity int
	}{
		{name: "no text files", count: 0, intensity: 255},
		{name: "all small", lines: 200, maxLines: 200, count: 3, intensity: 180},
		{name: "threshold in large range", lines: 200, maxLines: 10125, count: 5, intensity: 70},
		{name: "slightly above threshold", lines: 264, maxLines: 10125, count: 5, intensity: 83},
		{name: "active maximum", lines: 10125, maxLines: 10125, count: 5, intensity: 255},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := TextIntensity(test.lines, test.maxLines, test.count); got != test.intensity {
				t.Fatalf("TextIntensity(%d, %d, %d) = %d, want %d", test.lines, test.maxLines, test.count, got, test.intensity)
			}
		})
	}
}

func TestSnapshotStructureDetectsFilesystemChanges(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(docs, "old.txt")
	if err := os.WriteFile(original, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before := SnapshotStructure(root, "", false, 1)
	if !before.Exists || before.Signature == 0 {
		t.Fatalf("invalid initial snapshot: %#v", before)
	}
	if unchanged := SnapshotStructure(root, "", false, 1); unchanged != before {
		t.Fatalf("stable snapshot changed: %#v and %#v", before, unchanged)
	}

	created := filepath.Join(root, "new.txt")
	if err := os.WriteFile(created, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if afterCreate := SnapshotStructure(root, "", false, 1); afterCreate == before {
		t.Fatal("snapshot did not detect creation")
	}

	if err := os.WriteFile(original, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeEdit := SnapshotStructure(root, "", false, 1)
	if err := os.WriteFile(original, []byte("one\ntwo\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if afterEdit := SnapshotStructure(root, "", false, 1); afterEdit == beforeEdit {
		t.Fatal("snapshot did not detect file metadata change")
	}

	if err := os.Remove(created); err != nil {
		t.Fatal(err)
	}
	if afterDelete := SnapshotStructure(root, "", false, 1); afterDelete == beforeEdit {
		t.Fatal("snapshot did not detect deletion")
	}
}

func TestSnapshotStructureRespectsPreviewFilterAndHiddenState(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	hidden := filepath.Join(root, ".hidden")
	if err := os.WriteFile(hidden, []byte("hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(docs, "nested.txt")
	if err := os.WriteFile(nested, []byte("nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	shallow := SnapshotStructure(root, "", false, 0)
	deep := SnapshotStructure(root, "", false, 1)
	if shallow == deep {
		t.Fatal("preview depth did not change snapshot boundary")
	}

	withoutHidden := SnapshotStructure(root, "", false, 1)
	withHidden := SnapshotStructure(root, "", true, 1)
	if withoutHidden == withHidden {
		t.Fatal("hidden setting did not change snapshot boundary")
	}

	filtered := SnapshotStructure(root, "docs", false, 1)
	if err := os.WriteFile(filepath.Join(root, "unfiltered-root.txt"), []byte("root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if afterRootCreate := SnapshotStructure(root, "docs", false, 1); afterRootCreate != filtered {
		t.Fatal("root filter admitted an unrelated root change")
	}

	if deleted := SnapshotStructure(filepath.Join(root, "missing"), "", false, 1); deleted.Exists {
		t.Fatalf("missing directory reported existing: %#v", deleted)
	}
}
