// Package tuisession stores one session for each c3po terminal screen in the
// tui_session table. The session stays after the screen closes so a later
// --resume can open the same screen. Its tui_active row is the running
// process and is only present while that process runs.
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

var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Place is where a screen was: the durable screen state. Cursor is a stable row key. Path is a
// directory, a file, or a parent id. Args are the flags a one-shot screen
// such as l should replay; nil means none were noted. Secrets are not stored.
type Place struct {
	Screen string
	CWD    string
	Path   string
	Cursor string
	Args   []string
}

// Session is one open screen. A nil Session ignores Note, Place, and Close
// so model tests that never open a session stay quiet.
type Session struct {
	id       string
	screen   string
	body     Place
	restored bool
	pid      int
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
	return open(screen, id, os.Getpid())
}

// open is Open for the screen running as process pid.
func open(screen, id string, pid int) (*Session, error) {
	if screen == "" {
		return nil, errors.New("c3po: session screen is empty")
	}
	if id == "" {
		next, err := newID()
		if err != nil {
			return nil, err
		}
		sess := &Session{id: next, screen: screen, body: Place{Screen: screen}, pid: pid}
		if err := sess.write(); err != nil {
			return nil, err
		}
		return sess, nil
	}
	if !idPattern.MatchString(id) {
		return nil, errors.New("c3po: --resume needs a session id")
	}
	body, ok, err := load(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("c3po: session %s was not found", id)
	}
	if body.Screen != screen {
		return nil, fmt.Errorf("c3po: session %s is a %s screen", id, body.Screen)
	}
	body.Screen = screen
	sess := &Session{id: id, screen: screen, body: body, restored: true, pid: pid}
	if err := sess.write(); err != nil {
		return nil, err
	}
	return sess, nil
}

// Restored returns the saved body when this session was opened by id.
func (s *Session) Restored() (Place, bool) {
	if s == nil || !s.restored {
		return Place{}, false
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
func (s *Session) Note(note Place) error {
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
	return sqlxdb.ExecLocal("DELETE FROM tui_active WHERE session_id = " + sqlxdb.Quote(s.id))
}

func load(id string) (Place, bool, error) {
	var rows []struct {
		Screen string  `json:"screen"`
		CWD    string  `json:"cwd"`
		Path   string  `json:"path"`
		Cursor string  `json:"cursor"`
		Args   *string `json:"args"`
	}
	if err := sqlxdb.Query("SELECT screen, cwd, path, cursor, args FROM tui_session WHERE id = "+sqlxdb.Quote(id), &rows); err != nil {
		return Place{}, false, err
	}
	if len(rows) == 0 {
		return Place{}, false, nil
	}
	row := rows[0]
	body := Place{Screen: row.Screen, CWD: row.CWD, Path: row.Path, Cursor: row.Cursor}
	if row.Args != nil {
		if err := json.Unmarshal([]byte(*row.Args), &body.Args); err != nil {
			return Place{}, false, err
		}
	}
	return body, true, nil
}

// write stores the session and its active row in one transaction.
func (s *Session) write() error {
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		s.body.CWD = cwd
	}
	s.body.Screen = s.screen
	args := "NULL"
	if s.body.Args != nil {
		raw, err := json.Marshal(s.body.Args)
		if err != nil {
			return err
		}
		args = sqlxdb.Quote(string(raw))
	}
	ticks, _ := processStartTicks(s.pid)
	id := sqlxdb.Quote(s.id)
	return sqlxdb.ExecLocal("INSERT INTO tui_session(id, screen, cwd, path, cursor, args) VALUES (" +
		id + ", " + sqlxdb.Quote(s.screen) + ", " + sqlxdb.Quote(s.body.CWD) + ", " + sqlxdb.Quote(s.body.Path) + ", " +
		sqlxdb.Quote(s.body.Cursor) + ", " + args + ") ON CONFLICT(id) DO UPDATE SET screen = excluded.screen, " +
		"cwd = excluded.cwd, path = excluded.path, cursor = excluded.cursor, args = excluded.args;\n" +
		"INSERT INTO tui_active(session_id, pid, pid_start) VALUES (" + id + ", " + strconv.Itoa(s.pid) + ", " +
		strconv.FormatInt(ticks, 10) + ") ON CONFLICT(session_id) DO UPDATE SET pid = excluded.pid, pid_start = excluded.pid_start")
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
