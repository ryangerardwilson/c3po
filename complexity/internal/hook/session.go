package hook

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/config"
)

type session struct {
	Head  string   `json:"head"`
	Files []string `json:"files"`
}

func (ev event) sessionID() string {
	id := ev.SessionID
	if id == "" {
		id = ev.SessionID2
	}
	return sanitizeID(id)
}

func (ev event) stopActive() bool {
	return ev.StopHookActive || ev.StopHookActive2
}

func remember(cwd string, ev event) {
	id := ev.sessionID()
	if id == "" {
		return
	}
	root, ok := config.GitRoot(cwd)
	if !ok {
		return
	}
	s := loadSession(id)
	if s.Head == "" {
		s.Head = headSHA(root)
	}
	if file := eventFile(cwd, ev); file != "" && under(root, file) {
		s.Files = addFile(s.Files, file)
	}
	_ = saveSession(id, s)
}

func stopScope(cwd, root string, ev event) (string, []string) {
	id := ev.sessionID()
	if id == "" {
		return "HEAD", []string{root}
	}
	s := loadSession(id)
	if s.Head == "" {
		return "HEAD", []string{root}
	}
	paths := underRoot(root, s.Files)
	if len(paths) == 0 {
		return s.Head, []string{root}
	}
	return s.Head, paths
}

func skipStop(cfg config.Config, ev event) bool {
	return !cfg.Enforcement.OnStop || closed(ev.Reason) || ev.stopActive()
}

func sanitizeID(id string) string {
	var b strings.Builder
	for _, c := range id {
		if idChar(c) {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func idChar(c rune) bool {
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
		return true
	}
	return c == '_' || c == '-'
}

func sessionPath(id string) (string, error) {
	base, err := config.GlobalPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(base), "sessions", id+".json"), nil
}

func loadSession(id string) session {
	path, err := sessionPath(id)
	if err != nil {
		return session{}
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		return session{}
	}
	var s session
	if json.Unmarshal(buf, &s) != nil {
		return session{}
	}
	return s
}

func saveSession(id string, s session) error {
	path, err := sessionPath(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(buf, '\n'), 0o644)
}

func headSHA(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func eventFile(cwd string, ev event) string {
	in := ev.input()
	if in == nil {
		return ""
	}
	path := field(in, "file_path", "path", "target_file", "filePath")
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	return path
}

func under(root, file string) bool {
	rel, err := filepath.Rel(root, filepath.Clean(file))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}

func underRoot(root string, files []string) []string {
	var out []string
	for _, file := range files {
		if under(root, file) {
			out = append(out, file)
		}
	}
	return out
}

func addFile(files []string, file string) []string {
	for _, old := range files {
		if old == file {
			return files
		}
	}
	return append(files, file)
}
