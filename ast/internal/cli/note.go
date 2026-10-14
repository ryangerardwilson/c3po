package cli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/ast/internal/files"
)

func (r Runner) noteWritten(paths []string) {
	paths = sourcePaths(paths)
	if len(paths) == 0 {
		return
	}
	bin, err := exec.LookPath("cx")
	if err != nil {
		return
	}
	out := cxNote(bin, paths)
	if len(bytes.TrimSpace(out)) == 0 {
		return
	}
	_, _ = fmt.Fprintln(r.Out)
	_, _ = fmt.Fprint(r.Out, string(bytes.TrimRight(out, "\n")))
	_, _ = fmt.Fprintln(r.Out)
}

func cxNote(bin string, paths []string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, noteArgs(paths)...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return out
}

func noteArgs(paths []string) []string {
	args := []string{"note", "--format", "text"}
	for _, p := range paths {
		args = append(args, "--path", p)
	}
	return args
}

func sourcePaths(paths []string) []string {
	var out []string
	for _, p := range paths {
		if sourceExt(p) {
			out = append(out, p)
		}
	}
	return out
}

func sourceExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".py", ".rs", ".ts", ".tsx", ".js", ".jsx", ".prisma", ".sql":
		return true
	default:
		return false
	}
}

func applyAbs(root string, results []files.ApplyResult) []string {
	var out []string
	for _, res := range results {
		if skipNote(res.Op) {
			continue
		}
		out = append(out, absJoin(root, res.Path))
	}
	return out
}

func skipNote(op string) bool {
	return op == "delete" || op == "move"
}

func absJoin(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func (r Runner) notedSymbol(root string, results []files.ApplyResult) error {
	for _, res := range results {
		_, _ = fmt.Fprintf(r.Out, "ok  %s  %s  formatter %s\n", res.Hash, res.Path, formatterName(res.Formatter))
	}
	r.noteWritten(applyAbs(root, results))
	return nil
}

func (r Runner) notedHash(root, abs, hash string) error {
	rel, _ := filepath.Rel(root, abs)
	_, _ = fmt.Fprintf(r.Out, "ok  %s  %s\n", hash, filepath.ToSlash(rel))
	r.noteWritten([]string{abs})
	return nil
}
