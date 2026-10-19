package tree

import (
	"strings"
	"testing"
)

func TestIntensityThresholdBoundaries(t *testing.T) {
	cases := []struct {
		name                    string
		value, maxValue, thresh int64
		count, want             int
	}{
		{"no entries of the kind", 5, 5, 200, 0, 255},
		{"max at text threshold is neutral", 200, 200, 200, 3, 180},
		{"max under text threshold is neutral", 10, 199, 200, 3, 180},
		{"value at text threshold is floor", 200, 5000, 200, 3, 70},
		{"value one over text threshold", 201, 5000, 200, 3, 70},
		{"max text value is full", 5000, 5000, 200, 3, 255},
		{"log1p midpoint text", 2000, 20000, 200, 3, 162},
		{"max at binary threshold is neutral", 1, 100 * 1024, 100 * 1024, 2, 180},
		{"value at binary threshold is floor", 100 * 1024, 1 << 30, 100 * 1024, 2, 70},
		{"max binary value is full", 1 << 30, 1 << 30, 100 * 1024, 2, 255},
		{"value over max clamps to full", 9000, 5000, 200, 3, 255},
	}
	for _, tc := range cases {
		got := intensityForRange(tc.value, tc.maxValue, tc.count, tc.thresh)
		if got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestThresholdConstants(t *testing.T) {
	if textIntensityThresholdLines != 200 || binaryIntensityThresholdSize != 100*1024 {
		t.Fatal("thresholds changed")
	}
	if minIntensity != 70 || neutralIntensity != 180 || maxIntensity != 255 || treeIntensity != 128 {
		t.Fatal("intensities changed")
	}
}

func TestStyleUsesTextAndBinaryThresholds(t *testing.T) {
	stats := Stats{TextFileCount: 2, MaxTextLines: 200, BinaryFileCount: 2, MaxBinarySize: 100 * 1024}
	text := styleFor(&Node{Type: TypeFile, Lines: 200}, stats)
	if text != "\x1b[38;2;180;180;0m" {
		t.Fatalf("text at threshold: %q", text)
	}
	binary := styleFor(&Node{Type: TypeBinary, Size: 100 * 1024}, stats)
	if binary != "\x1b[38;2;0;180;180m" {
		t.Fatalf("binary at threshold: %q", binary)
	}
	stats.MaxTextLines = 201
	stats.MaxBinarySize = 100*1024 + 1
	if got := styleFor(&Node{Type: TypeFile, Lines: 201}, stats); got != "\x1b[38;2;255;255;0m" {
		t.Fatalf("text over threshold: %q", got)
	}
	if got := styleFor(&Node{Type: TypeBinary, Size: 100*1024 + 1}, stats); got != "\x1b[38;2;0;255;255m" {
		t.Fatalf("binary over threshold: %q", got)
	}
	if got := styleFor(&Node{Type: TypeFile, Lines: 200}, stats); got != "\x1b[38;2;70;70;0m" {
		t.Fatalf("text at threshold with larger max: %q", got)
	}
}

func TestNoColorLeavesLabelsAndGlyphsPlain(t *testing.T) {
	node := &Node{Type: TypeDirectory}
	if colorize("src", node, Stats{}, RenderConfig{}) != "src" || colorizeTree("├── ", RenderConfig{}) != "├── " {
		t.Fatal("expected plain output without color")
	}
	if colorizeTree("", RenderConfig{Color: true}) != "" {
		t.Fatal("empty glyphs stay empty")
	}
	if !strings.HasPrefix(colorizeTree("├── ", RenderConfig{Color: true}), "\x1b[38;2;128;128;128m") {
		t.Fatal("glyphs should be gray")
	}
}

func TestHumanSize(t *testing.T) {
	for size, want := range map[int64]string{
		-1: "?B", 0: "0B", 1023: "1023B", 1024: "1.0KB", 1536: "1.5KB",
		1024 * 1024: "1.0MB", 42 * 1024 * 1024: "42.0MB",
	} {
		if got := humanSize(size); got != want {
			t.Errorf("humanSize(%d)=%q want %q", size, got, want)
		}
	}
}
