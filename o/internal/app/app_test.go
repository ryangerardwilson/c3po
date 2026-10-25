package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
	"github.com/ryangerardwilson/c3po/o/internal/style"
	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func periodKey() tui.KeyMsg {
	return tui.KeyMsg{Type: tui.KeyRunes, Runes: []rune{'.'}}
}

func entryNames(m model) map[string]bool {
	names := make(map[string]bool, len(m.entries))
	for _, entry := range m.entries {
		names[entry.Name] = true
	}
	return names
}

func TestDotEntriesAreHiddenUntilPeriodTogglesThem(t *testing.T) {
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

	m := newModel(Config{StartDir: root})
	if m.showHidden {
		t.Fatal("new session shows dot entries by default")
	}
	if names := entryNames(m); names[".config"] || names[".env"] {
		t.Fatalf("dot entries visible initially: %#v", names)
	}

	next, _ := m.Update(periodKey())
	m, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if !m.showHidden {
		t.Fatal("period key did not enable dot entry visibility")
	}
	if names := entryNames(m); !names[".config"] || !names[".env"] {
		t.Fatalf("dot entries missing after toggle: %#v", names)
	}

	dotEntryIndex := -1
	for i, entry := range m.entries {
		if entry.Name == ".env" {
			dotEntryIndex = i
			break
		}
	}
	if dotEntryIndex < 0 {
		t.Fatal(".env entry missing")
	}
	m.selected = dotEntryIndex
	m.toggleMark()
	if !m.marks[filepath.Join(root, ".env")] {
		t.Fatal(".env was not marked")
	}

	next, _ = m.Update(periodKey())
	m, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.showHidden {
		t.Fatal("second period key did not hide dot entries")
	}
	if names := entryNames(m); names[".config"] || names[".env"] {
		t.Fatalf("dot entries remained visible: %#v", names)
	}
	if m.marks[filepath.Join(root, ".env")] {
		t.Fatal("mark survived after its entry left the listing")
	}

	fresh := newModel(Config{StartDir: root})
	if fresh.showHidden {
		t.Fatal("new session inherited visible dot entries")
	}
}

func TestRightNavigatesIntoDirectoryWithoutShellHandoff(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModel(Config{StartDir: root})
	next, cmd := m.Update(tui.KeyMsg{Type: tui.KeyRight})
	m, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.currentDir != child {
		t.Fatalf("currentDir = %q, want %q", m.currentDir, child)
	}
	if m.result.ShellCD != "" {
		t.Fatalf("right navigation requested shell handoff %q", m.result.ShellCD)
	}
	if cmd == nil {
		t.Fatal("right navigation did not schedule listing commands")
	}
}

func TestEnterAcceptsCurrentListingDirectory(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	if len(m.entries) < 2 || !m.entries[0].IsDir {
		t.Fatalf("unexpected fixture entries: %#v", m.entries)
	}

	next, cmd := m.Update(tui.KeyMsg{Type: tui.KeyEnter})
	m, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.result.ShellCD != root {
		t.Fatalf("ShellCD with child highlighted = %q, want %q", m.result.ShellCD, root)
	}
	if cmd == nil {
		t.Fatal("Enter did not request exit with child highlighted")
	}

	m = newModel(Config{StartDir: root, Mode: ModeBrowse})
	m.selected = 1
	next, cmd = m.Update(tui.KeyMsg{Type: tui.KeyEnter})
	m, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.result.ShellCD != root {
		t.Fatalf("ShellCD with file highlighted = %q, want %q", m.result.ShellCD, root)
	}
	if cmd == nil {
		t.Fatal("Enter did not request exit with file highlighted")
	}
}

func TestEnterAcceptsEmptyListingDirectory(t *testing.T) {
	root := t.TempDir()
	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	if len(m.entries) != 0 {
		t.Fatalf("expected empty listing, got %#v", m.entries)
	}

	next, cmd := m.Update(tui.KeyMsg{Type: tui.KeyEnter})
	m, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.result.ShellCD != root {
		t.Fatalf("ShellCD = %q, want %q", m.result.ShellCD, root)
	}
	if cmd == nil {
		t.Fatal("Enter did not request exit from empty listing")
	}
}

func TestNHandoffUsesCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	next, cmd := m.Update(tui.KeyMsg{Type: tui.KeyRunes, Runes: []rune{'n'}})
	m, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.result.ShellCD != root {
		t.Fatalf("ShellCD = %q, want %q", m.result.ShellCD, root)
	}
	if cmd == nil {
		t.Fatal("n did not request exit")
	}
}

func TestHelpDocumentsDotEntryToggle(t *testing.T) {
	help := model{showHelp: true}.renderHelp()
	if !strings.Contains(help, ".                  show or hide dot entries") {
		t.Fatalf("help missing dot entry toggle:\n%s", help)
	}
	if !strings.Contains(help, "staged cascade") {
		t.Fatalf("help missing listing animation:\n%s", help)
	}
	if !strings.Contains(help, "relative line count") {
		t.Fatalf("help missing relative intensity:\n%s", help)
	}
	if !strings.Contains(help, "r                  reload") || !strings.Contains(help, "polled every 500ms") {
		t.Fatalf("help missing live reload:\n%s", help)
	}
	if !strings.Contains(help, "= / -") {
		t.Fatalf("help missing preview depth controls:\n%s", help)
	}
	if !strings.Contains(help, ":                  touch, mkdir, or rm") || !strings.Contains(help, "!                  run a shell command") {
		t.Fatalf("help missing command prompt:\n%s", help)
	}
	if !strings.Contains(help, "x / y              cut or copy the current item or the visual range") || !strings.Contains(help, "p                  paste into this directory") {
		t.Fatalf("help missing cut and paste:\n%s", help)
	}
	if !strings.Contains(help, "v                  select consecutive items") || !strings.Contains(help, "backspace/delete   delete the visual range") {
		t.Fatalf("help missing visual mode:\n%s", help)
	}
	if strings.Contains(help, "wl-copy") {
		t.Fatalf("help still copies paths with wl-copy:\n%s", help)
	}
	if !strings.Contains(help, "dd                 delete the current item or the visual range") {
		t.Fatalf("help missing delete:\n%s", help)
	}
	view := model{showHelp: true, height: 24, width: 80}.View()
	if n := len(strings.Split(view, "\n")); n != 24 {
		t.Fatalf("help view lines = %d\n%s", n, view)
	}
	if !strings.Contains(view, "select consecutive") || !strings.HasSuffix(strings.TrimRight(view, "\n"), "quit") {
		t.Fatalf("help view dropped a key:\n%s", view)
	}
}

func TestListingRevealsStructureThenLabelsSequentially(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	if !m.animationActive || m.animationStep != 0 {
		t.Fatalf("unexpected initial animation state: active=%v step=%d", m.animationActive, m.animationStep)
	}
	if view := m.View(); strings.Contains(view, "child") || strings.Contains(view, "note.md") {
		t.Fatalf("labels appeared before animation:\n%s", view)
	}

	tick := listingAnimationTickMsg{generation: m.animationGeneration}
	next, cmd := m.Update(tick)
	m, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if cmd == nil {
		t.Fatal("first animation stage did not schedule its successor")
	}
	view := m.View()
	if strings.Contains(view, "child") || strings.Contains(view, "note.md") {
		t.Fatalf("label appeared during structure stage:\n%s", view)
	}
	if !strings.Contains(view, ">") {
		t.Fatalf("structural gutter missing:\n%s", view)
	}

	wanted := []string{"child", "note.md"}
	for index, wantVisible := range wanted {
		for tickIndex := 0; tickIndex < 2*index+1; tickIndex++ {
			next, cmd = m.Update(listingAnimationTickMsg{generation: m.animationGeneration})
			m, ok = next.(model)
			if !ok {
				t.Fatalf("Update returned %T", next)
			}
		}
		view = m.View()
		if !strings.Contains(view, wantVisible) {
			t.Fatalf("%q missing after label stage:\n%s", wantVisible, view)
		}
		if index < len(wanted)-1 && cmd == nil {
			t.Fatal("incomplete animation stopped scheduling")
		}
	}
	if m.animationActive {
		t.Fatal("animation remained active after all labels")
	}
}

func TestCompletedAnimationMatchesStaticListing(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	animated := newModel(Config{StartDir: root, Mode: ModeBrowse})
	for animated.animationActive {
		next, _ := animated.Update(listingAnimationTickMsg{generation: animated.animationGeneration})
		animated = next.(model)
	}
	static := newModel(Config{StartDir: root, Mode: ModeBrowse})
	static.animationActive = false
	static.animationStep = static.animationTargetSteps()
	if animated.View() != static.View() {
		t.Fatalf(
			"completed animated view differs from static view:\nanimated:\n%s\nstatic:\n%s",
			animated.View(),
			static.View(),
		)
	}
}

func TestListingReloadSupersedesStaleAnimation(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	staleGeneration := m.animationGeneration

	next, _ := m.Update(periodKey())
	m, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if !m.animationActive || m.animationStep != 0 || m.animationGeneration == staleGeneration {
		t.Fatalf("reload did not reset animation: active=%v step=%d generation=%d", m.animationActive, m.animationStep, m.animationGeneration)
	}

	next, cmd := m.Update(listingAnimationTickMsg{generation: staleGeneration})
	m, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.animationStep != 0 || cmd != nil {
		t.Fatalf("stale tick affected listing: step=%d cmd=%v", m.animationStep, cmd)
	}

	next, cmd = m.Update(listingAnimationTickMsg{generation: m.animationGeneration})
	m, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if m.animationStep != 1 || cmd == nil {
		t.Fatalf("current tick was not applied: step=%d cmd=%v", m.animationStep, cmd)
	}
}

func TestVisibleWindowControlsAnimationTarget(t *testing.T) {
	m := model{height: 14, selected: 19}
	m.previewRows = make([]fsnav.PreviewRow, 20)
	for i := range m.previewRows {
		m.previewRows[i] = fsnav.PreviewRow{IsRoot: true, RootIndex: i}
	}
	start, end := m.visibleWindow()
	if start != 9 || end != 20 {
		t.Fatalf("visible window = %d..%d, want 9..20", start, end)
	}
	if got := m.animationTargetSteps(); got != 22 {
		t.Fatalf("animationTargetSteps() = %d, want 22", got)
	}
}

func TestListingAnimationDelayUsesLCapBasis(t *testing.T) {
	m := model{animationDelay: defaultListingAnimationDelay}
	m.previewRows = make([]fsnav.PreviewRow, 100)
	if got := m.animationPhaseDelay(); got != 4*time.Millisecond {
		t.Fatalf("default phase delay = %s, want 4ms", got)
	}

	m.previewRows = make([]fsnav.PreviewRow, 150)
	if got := m.animationPhaseDelay(); got != 3*time.Millisecond {
		t.Fatalf("capped phase delay = %s, want 3ms", got)
	}
}

func previewKey(value string) tui.KeyMsg {
	return tui.KeyMsg{Type: tui.KeyRunes, Runes: []rune(value)}
}

func completePreviewAnimation(m model) model {
	m.animationActive = false
	m.animationStep = m.animationTargetSteps()
	return m
}

func TestPreviewDepthKeysShowAndHideNesting(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(root, "z.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	if tm.previewDepth != 0 || len(tm.previewRows) != 2 {
		t.Fatalf("initial preview state: depth=%d rows=%#v", tm.previewDepth, tm.previewRows)
	}

	beforeGeneration := tm.animationGeneration
	next, cmd := tm.Update(previewKey("="))
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.previewDepth != 1 {
		t.Fatalf("depth after = is %d", tm.previewDepth)
	}
	if tm.animationGeneration == beforeGeneration || !tm.animationActive || cmd == nil {
		t.Fatalf("preview increase did not restart animation: generation=%d active=%v cmd=%v", tm.animationGeneration, tm.animationActive, cmd)
	}
	names := make([]string, 0, len(tm.previewRows))
	for _, row := range tm.previewRows {
		names = append(names, row.Entry.Name)
	}
	want := []string{"docs", "sub", "guide.md", "z.txt"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("depth-one names=%#v want %#v", names, want)
	}

	view := completePreviewAnimation(tm).View()
	if !strings.Contains(view, "    ├── sub/") || !strings.Contains(view, "    └── guide.md") {
		t.Fatalf("preview view missing tree connectors:\n%s", view)
	}

	next, _ = tm.Update(previewKey("="))
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.previewDepth != 2 || len(tm.previewRows) != 5 || tm.previewRows[2].Entry.Name != "deep.md" {
		t.Fatalf("depth-two preview: depth=%d rows=%#v", tm.previewDepth, tm.previewRows)
	}

	next, _ = tm.Update(previewKey("-"))
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.previewDepth != 1 || len(tm.previewRows) != 4 {
		t.Fatalf("depth decrease did not remove nested rows: depth=%d rows=%#v", tm.previewDepth, tm.previewRows)
	}

	next, _ = tm.Update(previewKey("-"))
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.previewDepth != 0 || len(tm.previewRows) != 2 {
		t.Fatalf("depth zero preview: depth=%d rows=%#v", tm.previewDepth, tm.previewRows)
	}

	next, _ = tm.Update(previewKey("-"))
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.previewDepth != 0 || tm.status != "preview depth: minimum" {
		t.Fatalf("minimum preview state: depth=%d status=%q", tm.previewDepth, tm.status)
	}
}

func TestPreviewRowsAreVisualizationOnly(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(docs, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "child.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "z.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	next, _ := tm.Update(previewKey("="))
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if len(tm.entries) != 2 {
		t.Fatalf("descendants entered selectable entries: %#v", tm.entries)
	}

	next, _ = tm.Update(previewKey("j"))
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.selected != 1 || tm.entries[tm.selected].Name != "z.txt" {
		t.Fatalf("selection landed on non-root row: selected=%d entries=%#v", tm.selected, tm.entries)
	}

	tm.selected = 0
	tm.toggleMark()
	if !tm.marks[docs] || len(tm.marks) != 1 {
		t.Fatalf("marks did not remain root-only: %#v", tm.marks)
	}
}

func TestPreviewDepthPersistsAcrossNavigationAndResetsOnLaunch(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	sub := filepath.Join(docs, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "deep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	next, _ := tm.Update(previewKey("="))
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	next, _ = tm.Update(previewKey("="))
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}

	next, _ = tm.Update(tui.KeyMsg{Type: tui.KeyRight})
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.currentDir != docs || tm.previewDepth != 2 {
		t.Fatalf("navigation did not preserve preview depth: dir=%q depth=%d", tm.currentDir, tm.previewDepth)
	}
	hasDeep := false
	for _, row := range tm.previewRows {
		if row.Entry.Name == "deep.txt" && row.Depth == 1 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Fatalf("navigated preview omitted deep row: %#v", tm.previewRows)
	}

	fresh := newModel(Config{StartDir: root, Mode: ModeBrowse})
	if fresh.previewDepth != 0 || len(fresh.previewRows) != 1 {
		t.Fatalf("new session inherited preview state: depth=%d rows=%#v", fresh.previewDepth, fresh.previewRows)
	}
}

func TestPreviewDepthIsCapped(t *testing.T) {
	root := t.TempDir()
	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	for i := 0; i < fsnav.MaxPreviewDepth+2; i++ {
		next, _ := tm.Update(previewKey("="))
		tm = next.(model)
	}
	if tm.previewDepth != fsnav.MaxPreviewDepth || tm.status != "preview depth: maximum" {
		t.Fatalf("maximum preview state: depth=%d status=%q", tm.previewDepth, tm.status)
	}
}

func TestTextLabelStyleUsesRelativeLineIntensity(t *testing.T) {
	neutral := textLabelStyle(fsnav.Entry{IsText: true, Lines: 200}, 200, 3)
	if neutral.GetForeground() != style.Color("#B4B400") {
		t.Fatalf("neutral foreground = %#v", neutral.GetForeground())
	}

	low := textLabelStyle(fsnav.Entry{IsText: true, Lines: 200}, 10125, 5)
	if low.GetForeground() != style.Color("#464600") {
		t.Fatalf("low foreground = %#v", low.GetForeground())
	}

	high := textLabelStyle(fsnav.Entry{IsText: true, Lines: 10125}, 10125, 5)
	if high.GetForeground() != style.Color("#FFFF00") {
		t.Fatalf("high foreground = %#v", high.GetForeground())
	}

	if directory := textLabelStyle((fsnav.Entry{IsDir: true}), 10125, 5); directory.GetForeground() != (style.NoColor{}) {
		t.Fatalf("directory foreground = %#v", directory.GetForeground())
	}
	if binary := textLabelStyle((fsnav.Entry{Lines: 0}), 10125, 5); binary.GetForeground() != (style.NoColor{}) {
		t.Fatalf("binary foreground = %#v", binary.GetForeground())
	}
}

func TestViewColorsRootNestedAndSelectedTextFiles(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	small := strings.Repeat("line\n", 200)
	large := strings.Repeat("line\n", 10125)
	if err := os.WriteFile(filepath.Join(root, "a-small.txt"), []byte(small), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "nested-small.txt"), []byte(small), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "z-large.txt"), []byte(large), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	next, _ := tm.Update(previewKey("="))
	tm = next.(model)
	tm = completePreviewAnimation(tm)
	maxLines, textCount := tm.textPreviewStats()
	if maxLines != 10125 || textCount != 3 {
		t.Fatalf("preview stats = max %d, count %d", maxLines, textCount)
	}
	view := tm.View()
	for _, name := range []string{"a-small.txt", "nested-small.txt", "z-large.txt"} {
		if !strings.Contains(view, name) {
			t.Fatalf("preview missing %s:\n%s", name, view)
		}
	}

	tm.selected = len(tm.entries) - 1
	largeEntry := tm.entries[tm.selected]
	selected := selectedStyle.Foreground(textLabelStyle(largeEntry, maxLines, textCount).GetForeground())
	if selected.GetForeground() != style.Color("#FFFF00") {
		t.Fatalf("selected foreground = %#v", selected.GetForeground())
	}
	if selected.GetBackground() != selectedStyle.GetBackground() {
		t.Fatalf("selected background changed: %#v", selected.GetBackground())
	}
}

func TestManualRReloadsImmediately(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "initial.txt"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	animationGeneration := tm.animationGeneration
	watchGeneration := tm.watchGeneration
	if err := os.WriteFile(filepath.Join(root, "agent-created.txt"), []byte("created\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	next, cmd := tm.Update(previewKey("r"))
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if names := entryNames(tm); !names["agent-created.txt"] {
		t.Fatalf("manual reload missed new entry: %#v", names)
	}
	if tm.status != "reloaded" {
		t.Fatalf("manual reload status = %q", tm.status)
	}
	if tm.animationGeneration == animationGeneration || !tm.animationActive {
		t.Fatalf("manual reload did not restart animation: generation=%d active=%v", tm.animationGeneration, tm.animationActive)
	}
	if tm.watchGeneration == watchGeneration || cmd == nil {
		t.Fatalf("manual reload did not restart watching: generation=%d cmd=%v", tm.watchGeneration, cmd)
	}
}

func TestAutomaticStructurePollReloadsAndPreservesSelection(t *testing.T) {
	root := t.TempDir()
	kept := filepath.Join(root, "kept.txt")
	if err := os.WriteFile(kept, []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	tm.selected = 0
	animationGeneration := tm.animationGeneration
	watchGeneration := tm.watchGeneration
	if err := os.WriteFile(filepath.Join(root, "agent-created.txt"), []byte("created\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fresh := fsnav.SnapshotStructure(root, "", false, tm.previewDepth)
	next, cmd := tm.Update(structurePollMsg{generation: tm.watchGeneration, snapshot: fresh})
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if names := entryNames(tm); !names["agent-created.txt"] {
		t.Fatalf("automatic poll missed new entry: %#v", names)
	}
	if tm.status != "listing updated" {
		t.Fatalf("automatic reload status = %q", tm.status)
	}
	if tm.selected < 0 || tm.selected >= len(tm.entries) || tm.entries[tm.selected].Path != kept {
		t.Fatalf("automatic reload lost selected path: selected=%d entries=%#v", tm.selected, tm.entries)
	}
	if tm.animationGeneration == animationGeneration || !tm.animationActive {
		t.Fatalf("automatic reload did not restart animation: generation=%d active=%v", tm.animationGeneration, tm.animationActive)
	}
	if tm.watchGeneration == watchGeneration || cmd == nil {
		t.Fatalf("automatic reload did not continue watching: generation=%d cmd=%v", tm.watchGeneration, cmd)
	}

	next, _ = tm.Update(listingAnimationTickMsg{generation: tm.animationGeneration})
	tm, ok = next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.animationStep == 0 {
		t.Fatal("automatic reload animation was not schedulable")
	}
}

func TestAutomaticStructurePollHandlesDeletion(t *testing.T) {
	root := t.TempDir()
	kept := filepath.Join(root, "a-kept.txt")
	removed := filepath.Join(root, "z-removed.txt")
	if err := os.WriteFile(kept, []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(removed, []byte("removed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	tm.selected = 1
	tm.toggleMark()
	if err := os.Remove(removed); err != nil {
		t.Fatal(err)
	}

	fresh := fsnav.SnapshotStructure(root, "", false, tm.previewDepth)
	next, _ := tm.Update(structurePollMsg{generation: tm.watchGeneration, snapshot: fresh})
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if names := entryNames(tm); names["z-removed.txt"] {
		t.Fatalf("automatic poll retained deleted entry: %#v", names)
	}
	if tm.selected < 0 || tm.selected >= len(tm.entries) || tm.entries[tm.selected].Path != kept {
		t.Fatalf("selection invalid after deletion: selected=%d entries=%#v", tm.selected, tm.entries)
	}
	if tm.marks[removed] || len(tm.marks) != 0 {
		t.Fatalf("deleted mark survived: %#v", tm.marks)
	}
}

func TestAutomaticStructurePollHandlesCurrentDirectoryDeletion(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child", "grandchild")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	tm := newModel(Config{StartDir: child, Mode: ModeBrowse})
	if err := os.RemoveAll(filepath.Join(root, "child")); err != nil {
		t.Fatal(err)
	}

	fresh := fsnav.SnapshotStructure(child, "", false, tm.previewDepth)
	next, _ := tm.Update(structurePollMsg{generation: tm.watchGeneration, snapshot: fresh})
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.currentDir != root {
		t.Fatalf("current directory after deletion = %q, want %q", tm.currentDir, root)
	}
	if !strings.Contains(tm.status, "directory removed:") {
		t.Fatalf("deletion status = %q", tm.status)
	}
}

func TestStructurePollIgnoresStaleGenerations(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "initial.txt"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	staleGeneration := tm.watchGeneration
	tm.reload()
	watchGeneration := tm.watchGeneration
	if err := os.WriteFile(filepath.Join(root, "stale-change.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fresh := fsnav.SnapshotStructure(root, "", false, tm.previewDepth)
	next, cmd := tm.Update(structurePollMsg{generation: staleGeneration, snapshot: fresh})
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if names := entryNames(tm); names["stale-change.txt"] {
		t.Fatalf("stale poll mutated newer listing: %#v", names)
	}
	if tm.watchGeneration != watchGeneration {
		t.Fatalf("stale poll changed watch generation: %d", tm.watchGeneration)
	}
	if cmd != nil {
		t.Fatalf("stale poll scheduled commands: %v", cmd)
	}
}

func TestUnchangedStructurePollDoesNotReload(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "stable.txt"), []byte("stable\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	tm.animationActive = false
	tm.animationStep = tm.animationTargetSteps()
	watchGeneration := tm.watchGeneration

	next, cmd := tm.Update(structurePollMsg{
		generation: tm.watchGeneration,
		snapshot:   tm.structureSnapshot,
	})
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.watchGeneration != watchGeneration || tm.status != "ready" {
		t.Fatalf("unchanged poll reloaded listing: generation=%d status=%q", tm.watchGeneration, tm.status)
	}
	if cmd == nil {
		t.Fatal("unchanged poll stopped watching")
	}
}

func TestFilterInputTreatsRArtText(t *testing.T) {
	root := t.TempDir()
	tm := newModel(Config{StartDir: root, Mode: ModeBrowse})
	tm.filtering = true
	tm.input = "ma"
	watchGeneration := tm.watchGeneration

	next, _ := tm.Update(previewKey("r"))
	tm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if tm.input != "mar" {
		t.Fatalf("filter input = %q, want mar", tm.input)
	}
	if tm.watchGeneration != watchGeneration {
		t.Fatalf("filter-text r reloaded listing: generation=%d", tm.watchGeneration)
	}
}

func TestInitSchedulesStructurePolling(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "watched.txt"), []byte("watched\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if newModel(Config{StartDir: root, Mode: ModeBrowse}).Init() == nil {
		t.Fatal("Init did not schedule structure polling")
	}
}
