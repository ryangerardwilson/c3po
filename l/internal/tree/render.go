package tree

import (
	"fmt"
	"io"
	"math"
	"time"
)

// ColorMode is auto, always, or never.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// AnimationMode is auto, always, or never.
type AnimationMode string

const (
	AnimationAuto   AnimationMode = "auto"
	AnimationAlways AnimationMode = "always"
	AnimationNever  AnimationMode = "never"
)

const (
	textIntensityThresholdLines  int64 = 200
	binaryIntensityThresholdSize int64 = 100 * 1024
	treeIntensity                      = 128
	minIntensity                       = 70
	neutralIntensity                   = 180
	maxIntensity                       = 255
	DefaultAnimationDelay              = 8 * time.Millisecond
	maxAnimationDuration               = 900 * time.Millisecond
)

// RenderConfig controls color and animation.
type RenderConfig struct {
	Color          bool
	Animate        bool
	AnimationDelay time.Duration
}

type renderer struct {
	out   io.Writer
	stats Stats
	cfg   RenderConfig
}

// Render writes the tree under root. A nil root writes nothing.
func Render(out io.Writer, root *Node, stats Stats, cfg RenderConfig) error {
	if root == nil {
		return nil
	}
	if cfg.Animate && cfg.AnimationDelay > 0 {
		cfg.AnimationDelay = cappedAnimationDelay(root, cfg.AnimationDelay)
	}
	r := renderer{out: out, stats: stats, cfg: cfg}
	return r.line(root, "", "", "")
}

func cappedAnimationDelay(root *Node, delay time.Duration) time.Duration {
	count := countNodes(root)
	if count <= 0 {
		return delay
	}
	return min(delay, maxAnimationDuration/time.Duration(count))
}

func countNodes(node *Node) int {
	if node == nil {
		return 0
	}
	count := 1
	for _, child := range node.Children {
		count += countNodes(child)
	}
	return count
}

// line writes node after prefix and connector, then its children under
// childPrefix.
func (r renderer) line(node *Node, prefix, connector, childPrefix string) error {
	structure := colorizeTree(prefix+connector, r.cfg)
	label := node.Name
	if detail := detailFor(node); detail != "" {
		label += " " + detail
	}
	if err := r.write(structure, colorize(label, node, r.stats, r.cfg)); err != nil {
		return err
	}

	for i, child := range node.Children {
		conn, next := "├── ", childPrefix+"│   "
		if i == len(node.Children)-1 {
			conn, next = "└── ", childPrefix+"    "
		}
		if err := r.line(child, childPrefix, conn, next); err != nil {
			return err
		}
	}
	return nil
}

func (r renderer) write(structure string, label string) error {
	if !r.cfg.Animate {
		_, err := fmt.Fprintln(r.out, structure+label)
		return err
	}

	if structure != "" {
		if _, err := fmt.Fprint(r.out, structure); err != nil {
			return err
		}
		sleepAnimation(r.cfg.AnimationDelay / 2)
	}
	if _, err := fmt.Fprintln(r.out, label); err != nil {
		return err
	}
	sleepAnimation(r.cfg.AnimationDelay)
	return nil
}

func sleepAnimation(delay time.Duration) {
	if delay > 0 {
		time.Sleep(delay)
	}
}

func detailFor(node *Node) string {
	switch node.Type {
	case TypeFile:
		return fmt.Sprintf("%s %dL", humanSize(node.Size), node.Lines)
	case TypeBinary:
		return humanSize(node.Size)
	default:
		return ""
	}
}

func humanSize(size int64) string {
	if size < 0 {
		return "?B"
	}
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	value := float64(size)
	unit := "KB"
	for _, next := range units {
		value = value / 1024
		unit = next
		if math.Abs(value) < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f%s", value, unit)
}

func colorize(line string, node *Node, stats Stats, cfg RenderConfig) string {
	if !cfg.Color {
		return line
	}
	style := styleFor(node, stats)
	if style == "" {
		return line
	}
	return style + line + "\x1b[0m"
}

func colorizeTree(line string, cfg RenderConfig) string {
	if !cfg.Color || line == "" {
		return line
	}
	return rgbStyle(treeIntensity, treeIntensity, treeIntensity) + line + "\x1b[0m"
}

// styleFor is white for directories, cyan for binaries scaled by size,
// and yellow for text files scaled by line count.
func styleFor(node *Node, stats Stats) string {
	switch node.Type {
	case TypeDirectory:
		return rgbStyle(255, 255, 255)
	case TypeBinary:
		intensity := intensityForRange(node.Size, stats.MaxBinarySize, stats.BinaryFileCount, binaryIntensityThresholdSize)
		return rgbStyle(0, intensity, intensity)
	case TypeFile:
		intensity := intensityForRange(int64(node.Lines), int64(stats.MaxTextLines), stats.TextFileCount, textIntensityThresholdLines)
		return rgbStyle(intensity, intensity, 0)
	default:
		return ""
	}
}

func rgbStyle(red int, green int, blue int) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", red, green, blue)
}

// intensityForRange scales value between threshold and maxValue on a
// log1p scale, from minIntensity to maxIntensity. Values at or under the
// threshold get minIntensity. When maxValue is at or under the threshold,
// every value gets neutralIntensity.
func intensityForRange(value int64, maxValue int64, count int, threshold int64) int {
	if count <= 0 {
		return maxIntensity
	}
	if maxValue <= threshold {
		return neutralIntensity
	}
	if value <= threshold {
		return minIntensity
	}

	minLog := math.Log1p(float64(threshold))
	maxLog := math.Log1p(float64(maxValue))
	valueLog := math.Log1p(float64(value))
	ratio := (valueLog - minLog) / (maxLog - minLog)
	ratio = math.Min(math.Max(ratio, 0), 1)

	return minIntensity + int(math.Round(ratio*float64(maxIntensity-minIntensity)))
}
