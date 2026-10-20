// Package app is the navigator screen: listing, cascade reveal,
// structure polling, keys, and the picker and save modes.
package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

type Mode string

const (
	ModeBrowse Mode = "browse"
	ModePick   Mode = "pick"
	ModeSave   Mode = "save"
)

const (
	defaultListingAnimationDelay = 8 * time.Millisecond
	listingAnimationDuration     = 900 * time.Millisecond
	structurePollInterval        = 500 * time.Millisecond
)

// Config is how one navigator session starts.
type Config struct {
	StartDir     string
	RevealTarget string
	Mode         Mode
	Allowed      string
	Extensions   []string
	Multi        bool
	// Descend lets l enter a directory during pick. enter confirms the row.
	Descend bool
}

// Result is what the session hands back: picked paths or a shell cd target.
type Result struct {
	Selections []string
	ShellCD    string
}

type listingAnimationTickMsg struct {
	generation int
}

type structurePollMsg struct {
	generation int
	snapshot   fsnav.StructureSnapshot
}

type openedMsg struct {
	name string
	ok   bool
}

type model struct {
	cfg Config

	currentDir   string
	entries      []fsnav.Entry
	previewRows  []fsnav.PreviewRow
	previewDepth int
	selected     int
	filter       string
	showHidden   bool
	input        string
	filtering    bool
	marks        map[string]bool
	status       string
	showHelp     bool
	result       Result

	animationGeneration int
	animationStep       int
	animationActive     bool
	animationDelay      time.Duration

	watchGeneration   int
	structureSnapshot fsnav.StructureSnapshot
	watchInterval     time.Duration

	width  int
	height int
}

// Run shows the navigator on the terminal until it quits. Errors go to
// stderr and return exit code 1.
func Run(cfg Config, stderr io.Writer) (Result, int) {
	finalModel, err := tui.NewProgram(newModel(cfg)).Run()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return Result{}, 1
	}
	if final, ok := finalModel.(model); ok {
		return final.result, 0
	}
	return Result{}, 0
}

func newModel(cfg Config) model {
	m := model{
		cfg:            cfg,
		currentDir:     startDirectory(cfg.StartDir),
		marks:          map[string]bool{},
		status:         "ready",
		width:          80,
		height:         24,
		animationDelay: defaultListingAnimationDelay,
		watchInterval:  structurePollInterval,
	}
	m.reload()
	if cfg.RevealTarget != "" {
		m.selectPath(fsnav.NormalizePath(cfg.RevealTarget))
	}
	return m
}

func startDirectory(raw string) string {
	start := fsnav.NormalizePath(raw)
	if start == "" {
		start, _ = os.Getwd()
	}
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		start = filepath.Dir(start)
	}
	return start
}

func (m model) Init() tui.Cmd {
	if m.animationActive {
		return tui.Batch(m.structurePollTick(), m.animationTick())
	}
	return m.structurePollTick()
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case listingAnimationTickMsg:
		return m.updateAnimationTick(msg)
	case structurePollMsg:
		return m.updateStructurePoll(msg)
	case openedMsg:
		m.status = openedStatus(msg)
	case tui.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func openedStatus(msg openedMsg) string {
	if msg.ok {
		return "opened " + msg.name
	}
	return "failed to open " + msg.name
}
