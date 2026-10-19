package l

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAcceptsPathAndDepthInEitherOrder(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "src", "nested"))
	mustWrite(t, filepath.Join(root, "src", "app.go"), "one\n")

	for _, args := range [][]string{
		{root, "-d", "1"},
		{"-d", "1", root},
		{root, "--depth=1"},
	} {
		var stdout, stderr bytes.Buffer
		code := Run(args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(%v) code=%d stderr=%s", args, code, stderr.String())
		}
		output := stdout.String()
		if !strings.Contains(output, "src") {
			t.Fatalf("Run(%v) did not show src:\n%s", args, output)
		}
		if strings.Contains(output, "app.go") || strings.Contains(output, "nested") {
			t.Fatalf("Run(%v) exceeded depth:\n%s", args, output)
		}
	}
}

func TestRunAllIncludesHidden(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".env"), "x\n")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"-a", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), ".env") {
		t.Fatalf("expected hidden file in output:\n%s", stdout.String())
	}
}

func TestRunFilesFlagShowsTextFilesAndHidesBinaries(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "src"))
	mustMkdir(t, filepath.Join(root, "assets"))
	mustWrite(t, filepath.Join(root, "src", "app.go"), "one\n")
	mustWrite(t, filepath.Join(root, "assets", "model.bin"), "\x00bin")
	mustWrite(t, filepath.Join(root, "root.bin"), "\x00root")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"-f", root, "-d", "2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"src", "app.go"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in files-only output:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{"assets", "model.bin", "root.bin"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("did not expect %q in files-only output:\n%s", unwanted, output)
		}
	}
}

func TestRunRejectsBadDepth(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"-d", "nope"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "depth must be a non-negative integer or *") {
		t.Fatalf("missing depth error:\n%s", stderr.String())
	}
}

func TestRunRejectsBadAnimationMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--animate=warp"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--animate must be auto, always, or never") {
		t.Fatalf("missing animation mode error:\n%s", stderr.String())
	}
}

func TestRunCanForceColor(t *testing.T) {
	root := t.TempDir()
	var big strings.Builder
	for i := 0; i < 1200; i++ {
		big.WriteString("line\n")
	}
	mustWrite(t, filepath.Join(root, "big.go"), big.String())

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--color=always", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\x1b[38;2;255;255;0m") {
		t.Fatalf("expected forced color:\n%q", stdout.String())
	}
}

func TestRunForcedAnimationKeepsCapturedOutputPlain(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "app.go"), "one\n")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--animate=always", "--color=never", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	output := stdout.String()
	if strings.ContainsAny(output, "\r\x1b") {
		t.Fatalf("expected forced animation to keep captured output plain, got %q", output)
	}
	if !strings.Contains(output, "app.go 4B 1L") {
		t.Fatalf("missing file output:\n%s", output)
	}
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
