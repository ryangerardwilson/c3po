package l

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/l/internal/tree"
)

type options struct {
	config        tree.Config
	colorMode     tree.ColorMode
	animationMode tree.AnimationMode
	help          bool
	version       bool
}

// switches are flags and commands that take no value.
var switches = map[string]func(*options){
	"help":         func(o *options) { o.help = true },
	"-h":           func(o *options) { o.help = true },
	"--help":       func(o *options) { o.help = true },
	"version":      func(o *options) { o.version = true },
	"--version":    func(o *options) { o.version = true },
	"-a":           func(o *options) { o.config.ShowHidden = true },
	"--all":        func(o *options) { o.config.ShowHidden = true },
	"-f":           func(o *options) { o.config.FilesOnly = true },
	"--files":      func(o *options) { o.config.FilesOnly = true },
	"--no-color":   func(o *options) { o.colorMode = tree.ColorNever },
	"--no-animate": func(o *options) { o.animationMode = tree.AnimationNever },
	"--no-ignore":  func(o *options) { o.config.UseDefaultIgnores = false },
}

// valueFlag is a flag that takes one value. missing is a format with one
// %s for the flag name, used when the value is absent.
type valueFlag struct {
	missing string
	set     func(*options, string) error
}

var (
	depthFlag   = valueFlag{"%s requires a depth", setDepth}
	colorFlag   = valueFlag{"%s requires auto, always, or never", setColor}
	animateFlag = valueFlag{"%s requires auto, always, or never", setAnimate}
	ignoreFlag  = valueFlag{"%s requires a name or glob", addIgnores}
)

// valueFlags take their value from the next argument.
var valueFlags = map[string]valueFlag{
	"-d":        depthFlag,
	"--depth":   depthFlag,
	"--level":   depthFlag,
	"--color":   colorFlag,
	"--animate": animateFlag,
	"--ignore":  ignoreFlag,
}

// inlineFlags take their value after "=".
var inlineFlags = []struct {
	prefix string
	flag   valueFlag
}{
	{"--depth=", depthFlag},
	{"--level=", depthFlag},
	{"--color=", colorFlag},
	{"--animate=", animateFlag},
	{"--ignore=", ignoreFlag},
}

type parser struct {
	opts  options
	paths []string
}

func defaultOptions() options {
	return options{
		config: tree.Config{
			Root:              ".",
			MaxDepth:          2,
			UseDefaultIgnores: true,
		},
		colorMode:     tree.ColorAuto,
		animationMode: tree.AnimationAuto,
	}
}

func parseArgs(args []string) (options, error) {
	p := parser{opts: defaultOptions()}
	for i := 0; i < len(args); i++ {
		used, err := p.take(args, i)
		if err != nil {
			return p.opts, err
		}
		i += used
	}
	if len(p.paths) > 1 {
		return p.opts, errors.New("accepts at most one path")
	}
	if len(p.paths) == 1 {
		p.opts.config.Root = p.paths[0]
	}
	return p.opts, nil
}

// take reads args[i] and returns how many following arguments it used.
func (p *parser) take(args []string, i int) (int, error) {
	arg := args[i]
	if set, ok := switches[arg]; ok {
		set(&p.opts)
		return 0, nil
	}
	if flag, ok := valueFlags[arg]; ok {
		if i+1 >= len(args) {
			return 0, fmt.Errorf(flag.missing, arg)
		}
		return 1, flag.set(&p.opts, args[i+1])
	}
	if flag, value, ok := inlineValue(arg); ok {
		return 0, flag.set(&p.opts, value)
	}
	if arg == "--" {
		p.paths = append(p.paths, args[i+1:]...)
		return len(args) - i, nil
	}
	if strings.HasPrefix(arg, "-") {
		return 0, fmt.Errorf("unknown flag: %s", arg)
	}
	p.paths = append(p.paths, arg)
	return 0, nil
}

func inlineValue(arg string) (valueFlag, string, bool) {
	for _, in := range inlineFlags {
		if value, ok := strings.CutPrefix(arg, in.prefix); ok {
			return in.flag, value, true
		}
	}
	return valueFlag{}, "", false
}

func setDepth(o *options, value string) error {
	depth, err := parseDepth(value)
	if err != nil {
		return err
	}
	o.config.MaxDepth = depth
	return nil
}

func parseDepth(value string) (int, error) {
	if value == "*" {
		return -1, nil
	}
	depth, err := strconv.Atoi(value)
	if err != nil || depth < 0 {
		return 0, errors.New("depth must be a non-negative integer or *")
	}
	return depth, nil
}

func setColor(o *options, value string) error {
	switch mode := tree.ColorMode(value); mode {
	case tree.ColorAuto, tree.ColorAlways, tree.ColorNever:
		o.colorMode = mode
		return nil
	default:
		return errors.New("--color must be auto, always, or never")
	}
}

func setAnimate(o *options, value string) error {
	switch mode := tree.AnimationMode(value); mode {
	case tree.AnimationAuto, tree.AnimationAlways, tree.AnimationNever:
		o.animationMode = mode
		return nil
	default:
		return errors.New("--animate must be auto, always, or never")
	}
}

func addIgnores(o *options, value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			o.config.ExtraIgnores = append(o.config.ExtraIgnores, part)
		}
	}
	return nil
}
