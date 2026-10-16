package vibetree

import (
	"fmt"
	"io"
	"math"
	"os"
)

const (
	colorAuto   = "auto"
	colorAlways = "always"
	colorNever  = "never"

	minIntensity     = 70
	neutralIntensity = 180
	maxIntensity     = 255
	// lineFloor matches l: counts at or under 200 lines share one dim yellow.
	lineFloor = 200
	// cycloFloor keeps a straight function dim. 200 would dim almost every file.
	cycloFloor = 1
)

func shouldColor(out io.Writer, mode string) bool {
	switch mode {
	case colorAlways:
		return true
	case colorNever:
		return false
	default:
		return isTerminal(out)
	}
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func setColor(opt *options, value string) error {
	switch value {
	case colorAuto, colorAlways, colorNever:
		opt.color = value
		return nil
	default:
		return fmt.Errorf("vibetree: --color must be auto, always, or never")
	}
}

func floorOf(mode string) int {
	if mode == modeCyclo {
		return cycloFloor
	}
	return lineFloor
}

// paint colors a tree line yellow. Brightness follows size on the same curve
// l uses for text files, above floor.
func paint(line string, lines, maxLines, floor int, on bool) string {
	if !on {
		return line
	}
	n := intensity(lines, maxLines, floor)
	return fmt.Sprintf("\x1b[38;2;%d;%d;0m%s\x1b[0m", n, n, line)
}

func intensity(lines, maxLines, floor int) int {
	if maxLines <= floor {
		return neutralIntensity
	}
	if lines <= floor {
		return minIntensity
	}
	minLog := math.Log1p(float64(floor))
	maxLog := math.Log1p(float64(maxLines))
	valueLog := math.Log1p(float64(lines))
	ratio := (valueLog - minLog) / (maxLog - minLog)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return minIntensity + int(math.Round(ratio*float64(maxIntensity-minIntensity)))
}
