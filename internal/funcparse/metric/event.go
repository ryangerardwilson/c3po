package metric

import "strings"

// Event is one scored construct inside a function. Cog, Cyclo, and Depth
// are the points it adds to cognitive, cyclomatic, and nesting.
type Event struct {
	Line    int
	Kind    string
	Nesting int
	Cog     int
	Cyclo   int
	Depth   int
}

// Names lists the function metrics in report order.
var Names = []string{"cognitive", "nesting", "lines", "params", "cyclomatic"}

// Value returns fn's score for a metric name.
func Value(fn Function, name string) int {
	switch name {
	case "cognitive":
		return fn.Cognitive
	case "nesting":
		return fn.Nesting
	case "lines":
		return fn.Lines
	case "params":
		return fn.Params
	case "cyclomatic":
		return fn.Cyclomatic
	default:
		return 0
	}
}

// Limit returns the threshold for a metric name. Zero disables it.
func Limit(th Thresholds, name string) int {
	return Value(Function{
		Cognitive: th.Cognitive, Nesting: th.Nesting, Lines: th.Lines,
		Params: th.Params, Cyclomatic: th.Cyclomatic,
	}, name)
}

// Points returns what e adds to a metric. Lines and params have no events.
func (e Event) Points(name string) int {
	switch name {
	case "cognitive":
		return e.Cog
	case "cyclomatic":
		return e.Cyclo
	case "nesting":
		return e.Depth
	default:
		return 0
	}
}

// Anonymous reports whether name is a position name such as func@12,
// arrow@12, or closure@12 rather than a declared name.
func Anonymous(name string) bool {
	return strings.Contains(name, "@")
}

// AnonKind returns the part of an anonymous name before the @.
func AnonKind(name string) string {
	if i := strings.IndexByte(name, '@'); i >= 0 {
		return name[:i]
	}
	return name
}
