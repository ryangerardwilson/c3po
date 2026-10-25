package o

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/o/internal/app"
)

func stubApp(t *testing.T, fn func(app.Config) (app.Result, int)) {
	t.Helper()
	previous := runApp
	runApp = func(cfg app.Config, _ io.Writer) (app.Result, int) { return fn(cfg) }
	t.Cleanup(func() { runApp = previous })
}

func noApp(t *testing.T) {
	stubApp(t, func(app.Config) (app.Result, int) {
		t.Fatal("runApp should not be called")
		return app.Result{}, 1
	})
}

func TestHelpAndVersionDoNotStartApp(t *testing.T) {
	noApp(t)
	for _, arg := range []string{"help", "-h", "--help"} {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{arg}, &stdout, &stderr); code != 0 {
			t.Fatalf("%s exit = %d", arg, code)
		}
		if stdout.String() != Help {
			t.Fatalf("%s output is not Help: %s", arg, stdout.String())
		}
	}
	if !strings.Contains(Help, "c3po tui o pick from ~/src files py,md") {
		t.Fatalf("help missing picker example: %s", Help)
	}
	if !strings.Contains(Help, "c3po tui o ~/Downloads") {
		t.Fatalf("help missing path example: %s", Help)
	}
	if strings.Contains(Help, "c3po tui o open") {
		t.Fatal("help still requires the word open")
	}
	if strings.Contains(Help, "upgrade") {
		t.Fatal("help still lists upgrade")
	}
	if !strings.Contains(Help, "x                cut the current item or the visual range") || !strings.Contains(Help, "p                paste into the listed directory") {
		t.Fatal("help missing cut and paste")
	}
	if !strings.Contains(Help, "v                select consecutive items") || !strings.Contains(Help, "delete           delete the visual range") {
		t.Fatal("help missing visual mode")
	}
	if strings.Contains(Help, "wl-copy") {
		t.Fatal("help still copies paths with wl-copy")
	}
	if !strings.Contains(Help, "dd               delete the current item or the visual range") {
		t.Fatal("help missing delete")
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("version exit = %d", code)
	}
	if strings.TrimSpace(stdout.String()) != version.String() {
		t.Fatalf("version = %q, want %q", stdout.String(), version.String())
	}
}

func TestGlobalActionsMustStandAlone(t *testing.T) {
	noApp(t)
	for _, args := range [][]string{{"help", "x"}, {"version", "x"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 1 {
			t.Fatalf("%v exit = %d", args, code)
		}
		if strings.TrimSpace(stderr.String()) != "Use help or version by itself." {
			t.Fatalf("%v stderr = %q", args, stderr.String())
		}
	}
}

func TestUsageErrors(t *testing.T) {
	noApp(t)
	cases := map[string]struct {
		args []string
		want string
	}{
		"config extra":  {[]string{"config", "x"}, Usage},
		"reveal empty":  {[]string{"reveal"}, revealUsage},
		"reveal two":    {[]string{"reveal", "a", "b"}, revealUsage},
		"pick no from":  {[]string{"pick", "in", "/tmp"}, pickUsage},
		"pick bad kind": {[]string{"pick", "from", "/tmp", "links"}, pickUsage},
		"pick extra":    {[]string{"pick", "from", "/tmp", "directories", "x"}, pickUsage},
		"save no ext":   {[]string{"save", "from", "/tmp", "extension"}, saveUsage},
		"save extra":    {[]string{"save", "from", "/tmp", "extension", "txt", "x"}, saveUsage},
		"save bad word": {[]string{"save", "from", "/tmp", "ext", "txt"}, saveUsage},
	}
	for name, tc := range cases {
		var stdout, stderr bytes.Buffer
		if code := Run(tc.args, &stdout, &stderr); code != 1 {
			t.Fatalf("%s exit = %d", name, code)
		}
		if strings.TrimSpace(stderr.String()) != tc.want {
			t.Fatalf("%s stderr = %q, want %q", name, stderr.String(), tc.want)
		}
	}
}

func TestParsePickGrammar(t *testing.T) {
	cfg, err := parsePick([]string{"pick", "from", "/tmp", "files", ".PY;md", "multiple"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Allowed != "file" || !cfg.Multi || strings.Join(cfg.Extensions, ",") != "py,md" {
		t.Fatalf("cfg = %#v", cfg)
	}
	cfg, err = parsePick([]string{"pick", "from", "/tmp", "files", "multiple"})
	if err != nil || cfg.Allowed != "file" || !cfg.Multi || len(cfg.Extensions) != 0 {
		t.Fatalf("files multiple: cfg = %#v, err = %v", cfg, err)
	}
	cfg, err = parsePick([]string{"pick", "from", "/tmp", "multiple"})
	if err != nil || cfg.Allowed != "any" || !cfg.Multi {
		t.Fatalf("multiple: cfg = %#v, err = %v", cfg, err)
	}
	cfg, err = parsePick([]string{"pick", "from", "/tmp", "directories"})
	if err != nil || cfg.Allowed != "dir" || cfg.Multi || cfg.Mode != app.ModePick {
		t.Fatalf("directories: cfg = %#v, err = %v", cfg, err)
	}
	cfg, err = parseSave([]string{"save", "from", "/tmp", "extension", "TXT"})
	if err != nil || cfg.Mode != app.ModeSave || strings.Join(cfg.Extensions, ",") != "txt" {
		t.Fatalf("save: cfg = %#v, err = %v", cfg, err)
	}
}

func TestPickPrintsSelectionAndWritesCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	start := filepath.Join(tmp, "docs")
	if err := os.Mkdir(start, 0o755); err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(start, "note.md")
	stubApp(t, func(cfg app.Config) (app.Result, int) {
		if cfg.Mode != app.ModePick || cfg.Allowed != "file" || !cfg.Multi {
			t.Fatalf("unexpected picker cfg: %#v", cfg)
		}
		return app.Result{Selections: []string{selected}}, 0
	})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pick", "from", start, "files", "md,txt", "multiple"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pick exit = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != selected {
		t.Fatalf("stdout = %q", stdout.String())
	}
	body, err := os.ReadFile(filepath.Join(tmp, "cache", "c3po", "o", "picker-selection.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(body)) != selected {
		t.Fatalf("cache = %q", string(body))
	}
}

func TestRevealStartsInParentAndTargetsPath(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got app.Config
	stubApp(t, func(cfg app.Config) (app.Result, int) {
		got = cfg
		return app.Result{}, 0
	})
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"reveal", target}, &stdout, &stderr); code != 0 {
		t.Fatalf("reveal exit = %d", code)
	}
	realTmp, _ := filepath.EvalSymlinks(tmp)
	if got.StartDir != realTmp || got.RevealTarget != filepath.Join(realTmp, "file.txt") || got.Mode != app.ModeBrowse {
		t.Fatalf("reveal cfg = %#v", got)
	}
}

func TestAWordThatIsNotACommandIsAPath(t *testing.T) {
	noApp(t)
	for _, args := range [][]string{{"upgrade"}, {"open"}, {"open", "Apps/xl"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 1 {
			t.Fatalf("%v exit = %d", args, code)
		}
		if !strings.HasPrefix(strings.TrimSpace(stderr.String()), "Path not found: ") {
			t.Fatalf("%v stderr = %q", args, stderr.String())
		}
		if strings.Contains(stderr.String(), "Use:") {
			t.Fatalf("%v printed usage: %q", args, stderr.String())
		}
	}
}

func TestOpenDirectoryStartsNavigator(t *testing.T) {
	tmp := t.TempDir()
	var got app.Config
	stubApp(t, func(cfg app.Config) (app.Result, int) {
		got = cfg
		return app.Result{}, 3
	})
	var stdout, stderr bytes.Buffer
	if code := Run([]string{tmp}, &stdout, &stderr); code != 3 {
		t.Fatalf("open dir exit = %d", code)
	}
	realTmp, _ := filepath.EvalSymlinks(tmp)
	if got.StartDir != realTmp || got.Mode != app.ModeBrowse {
		t.Fatalf("open cfg = %#v", got)
	}
}

func TestOpenRejectsMissingAndMixedTargets(t *testing.T) {
	noApp(t)
	tmp := t.TempDir()
	file := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	missing := filepath.Join(tmp, "missing.txt")
	if code := Run([]string{missing}, &stdout, &stderr); code != 1 {
		t.Fatalf("missing exit = %d", code)
	}
	if !strings.Contains(stderr.String(), "Path not found: ") {
		t.Fatalf("missing stderr = %q", stderr.String())
	}
	stderr.Reset()
	if code := Run([]string{file, tmp}, &stdout, &stderr); code != 1 {
		t.Fatalf("mixed exit = %d", code)
	}
	if strings.TrimSpace(stderr.String()) != "Multiple open targets must all be files" {
		t.Fatalf("mixed stderr = %q", stderr.String())
	}
}

func TestRunNavigatorWritesShellHandoff(t *testing.T) {
	tmp := t.TempDir()
	handoff := filepath.Join(tmp, "shell-cd")
	target := filepath.Join(tmp, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("O_SHELL_CD_FILE", handoff)
	stubApp(t, func(app.Config) (app.Result, int) {
		return app.Result{ShellCD: target}, 0
	})

	var stdout, stderr bytes.Buffer
	if code := runNavigator(app.Config{Mode: app.ModeBrowse}, &stdout, &stderr); code != 0 {
		t.Fatalf("runNavigator exit = %d, stderr = %s", code, stderr.String())
	}
	body, err := os.ReadFile(handoff)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != target+"\n" {
		t.Fatalf("handoff = %q, want %q", string(body), target+"\n")
	}
}

func TestConfigCreatesRealConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("VISUAL", "true")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"config"}, &stdout, &stderr); code != 0 {
		t.Fatalf("config exit = %d, stderr = %s", code, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(tmp, "c3po", "o", "config.json"))
	if err != nil {
		t.Fatalf("config file missing: %v", err)
	}
	if string(body) != "{\n  \"matrix_mode\": false\n}\n" {
		t.Fatalf("config body = %q", string(body))
	}
}

func TestConfigReturnsEditorStatus(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("VISUAL", "false")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"config"}, &stdout, &stderr); code != 1 {
		t.Fatalf("config exit = %d", code)
	}
}

func TestOpenTextFileUsesEditor(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "note.md")
	if err := os.WriteFile(target, []byte("# note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", "true")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{target}, &stdout, &stderr); code != 0 {
		t.Fatalf("open exit = %d, stderr = %s", code, stderr.String())
	}
}

func TestPickReturnsOnePath(t *testing.T) {
	stubApp(t, func(cfg app.Config) (app.Result, int) {
		if cfg.Mode != app.ModePick || !cfg.Descend || cfg.Allowed != "any" {
			t.Fatalf("%#v", cfg)
		}
		return app.Result{Selections: []string{"/tmp/a.txt"}}, 0
	})
	path, code := Pick("/tmp", io.Discard)
	if code != 0 || path != "/tmp/a.txt" {
		t.Fatalf("%d %q", code, path)
	}
}

func TestPickQuitIsEmpty(t *testing.T) {
	stubApp(t, func(app.Config) (app.Result, int) { return app.Result{}, 0 })
	path, code := Pick("/tmp", io.Discard)
	if code != 0 || path != "" {
		t.Fatalf("%d %q", code, path)
	}
}
