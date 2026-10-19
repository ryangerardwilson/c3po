package tree

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildUsesCommonSenseIgnoresAndHiddenRules(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "src"))
	mustMkdir(t, filepath.Join(root, "node_modules"))
	mustMkdir(t, filepath.Join(root, "__pycache__"))
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWrite(t, filepath.Join(root, "src", "app.go"), "one\ntwo\n")
	mustWrite(t, filepath.Join(root, "node_modules", "dep.js"), "ignored\n")
	mustWrite(t, filepath.Join(root, "__pycache__", "x.pyc"), "\x00binary")
	mustWrite(t, filepath.Join(root, ".env"), "hidden\n")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          2,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	if strings.Contains(output, "node_modules") || strings.Contains(output, "__pycache__") || strings.Contains(output, ".git") {
		t.Fatalf("expected generated/dependency dirs to be ignored:\n%s", output)
	}
	if strings.Contains(output, ".env") {
		t.Fatalf("expected hidden file to be hidden by default:\n%s", output)
	}
	if !strings.Contains(output, "src") || !strings.Contains(output, "app.go") {
		t.Fatalf("expected source files to be shown:\n%s", output)
	}
}

func TestBuildCanIncludeHiddenAndDisableIgnores(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "node_modules"))
	mustWrite(t, filepath.Join(root, "node_modules", "dep.js"), "dep\n")
	mustWrite(t, filepath.Join(root, ".env"), "hidden\n")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          2,
		ShowHidden:        true,
		UseDefaultIgnores: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	for _, want := range []string{"node_modules", "dep.js", ".env"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output:\n%s", want, output)
		}
	}
}

func TestBuildClassifiesTextAndBinaryAndCountsFinalLine(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "text.txt"), "one\ntwo")
	mustWrite(t, filepath.Join(root, "empty.txt"), "")
	mustWrite(t, filepath.Join(root, "blob.bin"), "\x00abc")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          1,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	for _, want := range []string{
		"├── blob.bin 4B",
		"├── empty.txt 0B 0L",
		"└── text.txt 7B 2L",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output:\n%s", want, output)
		}
	}
}

func TestDepthLimitsChildren(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "src", "nested"))
	mustWrite(t, filepath.Join(root, "src", "nested", "deep.go"), "deep\n")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          1,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	if !strings.Contains(output, "src") {
		t.Fatalf("expected depth 1 child:\n%s", output)
	}
	if strings.Contains(output, "nested") || strings.Contains(output, "deep.go") {
		t.Fatalf("expected depth limit to hide grandchildren:\n%s", output)
	}
}

func TestFilesOnlyPrunesBinariesAndEmptyDirectories(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "textdir"))
	mustMkdir(t, filepath.Join(root, "bindir"))
	mustMkdir(t, filepath.Join(root, "emptydir"))
	mustWrite(t, filepath.Join(root, "text.txt"), "one\n")
	mustWrite(t, filepath.Join(root, "blob.bin"), "\x00abc")
	mustWrite(t, filepath.Join(root, "textdir", "nested.txt"), "nested\n")
	mustWrite(t, filepath.Join(root, "bindir", "nested.bin"), "\x00nested")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          2,
		FilesOnly:         true,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	for _, want := range []string{"text.txt", "textdir", "nested.txt"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in files-only output:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{"blob.bin", "bindir", "nested.bin", "emptydir"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("did not expect %q in files-only output:\n%s", unwanted, output)
		}
	}
}

func TestFilesOnlyCanProduceNoRoot(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "blob.bin"), "\x00abc")

	node, stats, err := Build(Config{
		Root:              root,
		MaxDepth:          1,
		FilesOnly:         true,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if node != nil {
		t.Fatalf("expected no tree when files-only has no text files, got %#v stats=%#v", node, stats)
	}
}

func TestColorUsesTypeHuesAndRelativeIntensity(t *testing.T) {
	dir := &Node{Name: "src", Type: TypeDirectory}
	line := colorize("D src", dir, Stats{}, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[38;2;255;255;255m") {
		t.Fatalf("expected directory to be white, got %q", line)
	}

	huge := &Node{Name: "huge.go", Type: TypeFile, Size: 10, Lines: 20000}
	stats := Stats{TextFileCount: 2, MinTextLines: 100, MaxTextLines: 20000}
	line = colorize("F huge.go 10B 20000L", huge, stats, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[38;2;255;255;0m") {
		t.Fatalf("expected huge file to be bright yellow, got %q", line)
	}

	small := &Node{Name: "small.go", Type: TypeFile, Size: 10, Lines: 100}
	line = colorize("F small.go 10B 100L", small, stats, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[38;2;70;70;0m") {
		t.Fatalf("expected small file to be dark yellow, got %q", line)
	}

	bigBinary := &Node{Name: "big.bin", Type: TypeBinary, Size: 1024 * 1024}
	binaryStats := Stats{BinaryFileCount: 2, MinBinarySize: 1024, MaxBinarySize: 1024 * 1024}
	line = colorize("B big.bin 1.0MB", bigBinary, binaryStats, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[38;2;0;255;255m") {
		t.Fatalf("expected big binary to be bright cyan, got %q", line)
	}

	smallBinary := &Node{Name: "small.bin", Type: TypeBinary, Size: 1024}
	line = colorize("B small.bin 1.0KB", smallBinary, binaryStats, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[38;2;0;70;70m") {
		t.Fatalf("expected small binary to be dark cyan, got %q", line)
	}
}

func TestRenderKeepsTreeGlyphsGrayWhileColoringLabels(t *testing.T) {
	root := &Node{
		Name: "root",
		Type: TypeDirectory,
		Children: []*Node{
			{Name: "large.go", Type: TypeFile, Size: 1024, Lines: 1000},
			{Name: "asset.bin", Type: TypeBinary, Size: 1024 * 1024},
		},
	}
	stats := Stats{
		TextFileCount:   1,
		MaxTextLines:    1000,
		BinaryFileCount: 1,
		MaxBinarySize:   1024 * 1024,
	}

	var buf bytes.Buffer
	if err := Render(&buf, root, stats, RenderConfig{Color: true}); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	fileLine := "\x1b[38;2;128;128;128m├── \x1b[0m\x1b[38;2;255;255;0mlarge.go 1.0KB 1000L\x1b[0m"
	if !strings.Contains(output, fileLine) {
		t.Fatalf("expected gray tree glyphs followed by yellow file label, got %q", output)
	}

	binaryLine := "\x1b[38;2;128;128;128m└── \x1b[0m\x1b[38;2;0;255;255masset.bin 1.0MB\x1b[0m"
	if !strings.Contains(output, binaryLine) {
		t.Fatalf("expected gray tree glyphs followed by cyan binary label, got %q", output)
	}

	if strings.Contains(output, "\x1b[38;2;255;255;0m├──") || strings.Contains(output, "\x1b[38;2;0;255;255m└──") {
		t.Fatalf("expected tree glyphs not to inherit file or binary color, got %q", output)
	}
}

func TestRenderAnimationKeepsCapturedOutputStable(t *testing.T) {
	root := &Node{
		Name: "root",
		Type: TypeDirectory,
		Children: []*Node{
			{Name: "src", Type: TypeDirectory, Children: []*Node{
				{Name: "app.go", Type: TypeFile, Size: 12, Lines: 2},
			}},
			{Name: "asset.bin", Type: TypeBinary, Size: 1024 * 1024},
		},
	}
	stats := Stats{
		TextFileCount:   1,
		MaxTextLines:    2,
		BinaryFileCount: 1,
		MaxBinarySize:   1024 * 1024,
	}

	var plain bytes.Buffer
	if err := Render(&plain, root, stats, RenderConfig{Color: true}); err != nil {
		t.Fatal(err)
	}

	var animated bytes.Buffer
	if err := Render(&animated, root, stats, RenderConfig{Color: true, Animate: true}); err != nil {
		t.Fatal(err)
	}

	if animated.String() != plain.String() {
		t.Fatalf("animation should not change captured bytes\nplain:    %q\nanimated: %q", plain.String(), animated.String())
	}
}

func TestAnimationDelayCapsLargeTrees(t *testing.T) {
	root := &Node{Name: "root", Type: TypeDirectory}
	for i := 0; i < 1000; i++ {
		root.Children = append(root.Children, &Node{Name: "file.go", Type: TypeFile})
	}

	delay := cappedAnimationDelay(root, 8*time.Millisecond)
	if delay >= 8*time.Millisecond {
		t.Fatalf("expected large tree delay to be capped, got %s", delay)
	}
	if delay <= 0 {
		t.Fatalf("expected capped delay to stay positive, got %s", delay)
	}
}

func TestTextFileColorUsesYellowIntensityRangeForDominantFileShape(t *testing.T) {
	stats := Stats{TextFileCount: 5, MinTextLines: 13, MaxTextLines: 10125}

	cases := []struct {
		name  string
		lines int
		code  string
	}{
		{name: "x", lines: 13, code: "\x1b[38;2;70;70;0m"},
		{name: "README.md", lines: 18, code: "\x1b[38;2;70;70;0m"},
		{name: "y", lines: 50, code: "\x1b[38;2;70;70;0m"},
		{name: "AGENTS.md", lines: 264, code: "\x1b[38;2;83;83;0m"},
		{name: "BIBLE.md", lines: 10125, code: "\x1b[38;2;255;255;0m"},
	}
	for _, tc := range cases {
		node := &Node{Name: tc.name, Type: TypeFile, Lines: tc.lines}
		line := colorize("F "+tc.name, node, stats, RenderConfig{Color: true})
		if !strings.Contains(line, tc.code) {
			t.Fatalf("expected %s to use %q, got %q", tc.name, tc.code, line)
		}
	}
}

func TestBinaryColorUsesCyanIntensityRange(t *testing.T) {
	stats := Stats{BinaryFileCount: 4, MinBinarySize: 10, MaxBinarySize: 10 * 1024 * 1024}

	cases := []struct {
		name string
		size int64
		code string
	}{
		{name: "tiny.bin", size: 10, code: "\x1b[38;2;0;70;70m"},
		{name: "small.bin", size: 50 * 1024, code: "\x1b[38;2;0;70;70m"},
		{name: "middle.bin", size: 1024 * 1024, code: "\x1b[38;2;0;163;163m"},
		{name: "huge.bin", size: 10 * 1024 * 1024, code: "\x1b[38;2;0;255;255m"},
	}
	for _, tc := range cases {
		node := &Node{Name: tc.name, Type: TypeBinary, Size: tc.size}
		line := colorize("B "+tc.name, node, stats, RenderConfig{Color: true})
		if !strings.Contains(line, tc.code) {
			t.Fatalf("expected %s to use %q, got %q", tc.name, tc.code, line)
		}
	}
}

func TestColorUsesNeutralIntensityWhenAllEntriesOfATypeMatch(t *testing.T) {
	stats := Stats{TextFileCount: 3, MinTextLines: 100, MaxTextLines: 100}
	node := &Node{Name: "service.go", Type: TypeFile, Lines: 100}

	line := colorize("F service.go 100B 100L", node, stats, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[38;2;180;180;0m") {
		t.Fatalf("expected neutral yellow for equal-sized files, got %q", line)
	}

	binaryStats := Stats{BinaryFileCount: 2, MinBinarySize: 4096, MaxBinarySize: 4096}
	binary := &Node{Name: "asset.bin", Type: TypeBinary, Size: 4096}
	line = colorize("B asset.bin 4.0KB", binary, binaryStats, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[38;2;0;180;180m") {
		t.Fatalf("expected neutral cyan for equal-sized binaries, got %q", line)
	}
}

func TestSmallFilesAndBinariesDoNotVaryBySize(t *testing.T) {
	textStats := Stats{TextFileCount: 4, MinTextLines: 1, MaxTextLines: 5000}
	for _, lines := range []int{1, 50, 199, 200} {
		node := &Node{Name: "small.go", Type: TypeFile, Lines: lines}
		line := colorize("small.go", node, textStats, RenderConfig{Color: true})
		if !strings.Contains(line, "\x1b[38;2;70;70;0m") {
			t.Fatalf("expected %d-line file to use stable low yellow, got %q", lines, line)
		}
	}

	binaryStats := Stats{BinaryFileCount: 4, MinBinarySize: 1, MaxBinarySize: 2 * 1024 * 1024}
	for _, size := range []int64{1, 1024, 99 * 1024, 100 * 1024} {
		node := &Node{Name: "small.bin", Type: TypeBinary, Size: size}
		line := colorize("small.bin", node, binaryStats, RenderConfig{Color: true})
		if !strings.Contains(line, "\x1b[38;2;0;70;70m") {
			t.Fatalf("expected %d-byte binary to use stable low cyan, got %q", size, line)
		}
	}
}

func renderNoColor(t *testing.T, node *Node) string {
	t.Helper()
	var stats Stats
	collectStats(node, &stats)
	var buf bytes.Buffer
	if err := Render(&buf, node, stats, RenderConfig{}); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
