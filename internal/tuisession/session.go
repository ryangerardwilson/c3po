// Package tuisession stores one sqlx session for each c3po terminal screen.
// The session body stays after the screen closes so a later --resume can
// open the same screen. The active row is only present while the process runs.
package tuisession

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

const (
	appName     = "tui"
	kindSession = "session"
	kindActive  = "active"
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Body is the durable screen state. Cursor is a stable row key. Path is a
// directory, a file, or a parent id. Args are the flags a one-shot screen
// such as l should replay. Secrets are not stored.
type Body struct {
	ID     string   `json:"id"`
	Screen string   `json:"screen"`
	CWD    string   `json:"cwd,omitempty"`
	Path   string   `json:"path,omitempty"`
	Cursor string   `json:"cursor,omitempty"`
	Args   []string `json:"args,omitempty"`
}

type activeBody struct {
	SessionID string `json:"session_id"`
	PID       int    `json:"pid"`
	PIDStart  int64  `json:"pid_start"`
	Screen    string `json:"screen"`
	CWD       string `json:"cwd,omitempty"`
}

// Session is one open screen. A nil Session ignores Note, Place, and Close
// so model tests that never open a session stay quiet.
type Session struct {
	id       string
	screen   string
	body     Body
	restored bool
}

// TakeResume removes one --resume <id> or --resume=<id> from args.
// A bare --resume is an error. The id must be a session UUID.
func TakeResume(args []string) (string, []string, error) {
	var id string
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--resume":
			if id != "" {
				return "", nil, errors.New("c3po: --resume was given twice")
			}
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", nil, errors.New("c3po: --resume needs a session id")
			}
			i++
			id = args[i]
			if !idPattern.MatchString(id) {
				return "", nil, errors.New("c3po: --resume needs a session id")
			}
		case strings.HasPrefix(arg, "--resume="):
			if id != "" {
				return "", nil, errors.New("c3po: --resume was given twice")
			}
			id = strings.TrimPrefix(arg, "--resume=")
			if !idPattern.MatchString(id) {
				return "", nil, errors.New("c3po: --resume needs a session id")
			}
		default:
			rest = append(rest, arg)
		}
	}
	return id, rest, nil
}

// Open loads id or creates a session. A missing id returns an error so restore
// does not start an empty screen. The active row records this process.
func Open(screen, id string) (*Session, error) {
	if screen == "" {
		return nil, errors.New("c3po: session screen is empty")
	}
	if id == "" {
		next, err := newID()
		if err != nil {
			return nil, err
		}
		sess := &Session{id: next, screen: screen, body: Body{ID: next, Screen: screen}}
		if err := sess.write(); err != nil {
			return nil, err
		}
		return sess, nil
	}
	if !idPattern.MatchString(id) {
		return nil, errors.New("c3po: --resume needs a session id")
	}
	raw, ok, err := sqlxdb.Get(appName, kindSession, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("c3po: session %s was not found", id)
	}
	var body Body
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if body.Screen != "" && body.Screen != screen {
		return nil, fmt.Errorf("c3po: session %s is a %s screen", id, body.Screen)
	}
	body.ID = id
	body.Screen = screen
	sess := &Session{id: id, screen: screen, body: body, restored: true}
	if err := sess.write(); err != nil {
		return nil, err
	}
	return sess, nil
}

// Restored returns the saved body when this session was opened by id.
func (s *Session) Restored() (Body, bool) {
	if s == nil || !s.restored {
		return Body{}, false
	}
	return s.body, true
}

// ID is the session id a later --resume uses.
func (s *Session) ID() string {
	if s == nil {
		return ""
	}
	return s.id
}

// Note merges non-empty fields into the session and writes it.
// A non-nil Args slice replaces the saved args, including an empty slice.
func (s *Session) Note(note Body) error {
	if s == nil {
		return nil
	}
	if note.CWD != "" {
		s.body.CWD = note.CWD
	}
	if note.Path != "" {
		s.body.Path = note.Path
	}
	if note.Cursor != "" {
		s.body.Cursor = note.Cursor
	}
	if note.Args != nil {
		s.body.Args = append([]string(nil), note.Args...)
	}
	return s.write()
}

// Place sets the path and cursor even when they are empty, then writes.
func (s *Session) Place(path, cursor string) error {
	if s == nil {
		return nil
	}
	s.body.Path = path
	s.body.Cursor = cursor
	return s.write()
}

// Index returns the row whose key matches the restored cursor.
func (s *Session) Index(keys []string) (int, bool) {
	if s == nil || !s.restored || s.body.Cursor == "" {
		return 0, false
	}
	for i, key := range keys {
		if key == s.body.Cursor {
			return i, true
		}
	}
	return 0, false
}

// Close removes the active row and keeps the session body.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	return sqlxdb.Delete(appName, kindActive, s.id)
}

func (s *Session) write() error {
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		s.body.CWD = cwd
	}
	s.body.ID = s.id
	s.body.Screen = s.screen
	raw, err := json.Marshal(s.body)
	if err != nil {
		return err
	}
	if err := sqlxdb.Put(appName, kindSession, s.id, raw); err != nil {
		return err
	}
	return s.writeActive()
}

func (s *Session) writeActive() error {
	ticks, _ := processStartTicks(os.Getpid())
	raw, err := json.Marshal(activeBody{
		SessionID: s.id,
		PID:       os.Getpid(),
		PIDStart:  ticks,
		Screen:    s.screen,
		CWD:       s.body.CWD,
	})
	if err != nil {
		return err
	}
	return sqlxdb.Put(appName, kindActive, s.id, raw)
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func processStartTicks(pid int) (int64, bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	text := string(data)
	end := strings.LastIndex(text, ")")
	if end < 0 || end+2 >= len(text) {
		return 0, false
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) <= 19 {
		return 0, false
	}
	ticks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return 0, false
	}
	return ticks, true
}

// Interactive reports whether w is a terminal. Tests that pass a buffer
// are not interactive, so a screen can skip opening a session for them.
func Interactive(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
