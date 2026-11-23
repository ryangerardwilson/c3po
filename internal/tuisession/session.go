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
	"sync"
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
//
// Note and Place only change the place in memory. A writer goroutine stores
// the latest place, and only when it differs from the last one stored, so a
// key never waits on the database. The place is overwrite state: writing
// the latest one is enough. Close stores the last place before it returns.
type Session struct {
	id       string
	screen   string
	restored bool
	pid      int

	mu      sync.Mutex
	body    Place
	written string // key of the place last stored
	err     error  // last failed background write, reported once
	wake    chan struct{}
	done    chan struct{}
}

// closeWait bounds how long Close waits for a background write.
const closeWait = 2 * time.Second

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
		if err := sess.store(); err != nil {
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
	if err := sess.store(); err != nil {
		return nil, err
	}
	return sess, nil
}

// Restored returns the saved body when this session was opened by id.
func (s *Session) Restored() (Place, bool) {
	if s == nil || !s.restored {
		return Place{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.body, true
}

// ID is the session id a later --resume uses.
func (s *Session) ID() string {
	if s == nil {
		return ""
	}
	return s.id
}

// Note merges non-empty fields into the session and queues it for the
// writer. A non-nil Args slice replaces the saved args, including an empty
// slice. The error is a background write that failed since the last call.
func (s *Session) Note(note Place) error {
	if s == nil {
		return nil
	}
	return s.change(func(body *Place) {
		if note.CWD != "" {
			body.CWD = note.CWD
		}
		if note.Path != "" {
			body.Path = note.Path
		}
		if note.Cursor != "" {
			body.Cursor = note.Cursor
		}
		if note.Args != nil {
			body.Args = append([]string(nil), note.Args...)
		}
	})
}

// Place sets the path and cursor even when they are empty, then queues the
// place for the writer.
func (s *Session) Place(path, cursor string) error {
	if s == nil {
		return nil
	}
	return s.change(func(body *Place) {
		body.Path = path
		body.Cursor = cursor
	})
}

// change edits the place under the lock and wakes the writer when the
// place now differs from the one last stored.
func (s *Session) change(edit func(*Place)) error {
	s.mu.Lock()
	edit(&s.body)
	s.body.CWD = workingDir(s.body.CWD)
	changed := placeKey(s.body) != s.written
	err := s.err
	s.err = nil
	if changed && s.wake == nil {
		s.wake = make(chan struct{}, 1)
		s.done = make(chan struct{})
		go s.writer(s.wake, s.done)
	}
	wake := s.wake
	s.mu.Unlock()
	if changed {
		select {
		case wake <- struct{}{}:
		default: // a wake is already queued; it stores the latest place
		}
	}
	return err
}

func (s *Session) writer(wake <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for range wake {
		s.flush()
	}
}

// flush stores the current place unless it is already stored.
func (s *Session) flush() {
	s.mu.Lock()
	body := s.body
	body.Args = append([]string(nil), s.body.Args...)
	if s.body.Args == nil {
		body.Args = nil
	}
	key := placeKey(body)
	if key == s.written {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	err := s.write(body)
	s.mu.Lock()
	if err == nil {
		s.written = key
	} else {
		s.err = err
	}
	s.mu.Unlock()
}

// store writes the place now. Open uses it, so the session exists before the
// screen paints.
func (s *Session) store() error {
	s.mu.Lock()
	s.body.CWD = workingDir(s.body.CWD)
	body := s.body
	s.mu.Unlock()
	if err := s.write(body); err != nil {
		return err
	}
	s.mu.Lock()
	s.written = placeKey(body)
	s.mu.Unlock()
	return nil
}

func placeKey(body Place) string {
	raw, _ := json.Marshal(body)
	return string(raw)
}

func workingDir(fallback string) string {
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	return fallback
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

// Close stores the last place, waiting up to closeWait for a write already
// running, then removes the active row and keeps the session body.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	err := s.sync()
	if derr := sqlxdb.Exec("DELETE FROM tui_active WHERE session_id = " + sqlxdb.Quote(s.id)); derr != nil {
		return derr
	}
	return err
}

// sync stops the writer and stores the last changed place now. A session
// with no change since Open stores nothing.
func (s *Session) sync() error {
	s.mu.Lock()
	wake, done := s.wake, s.done
	s.wake = nil
	s.mu.Unlock()
	if wake != nil {
		close(wake)
		select {
		case <-done:
		case <-time.After(closeWait):
		}
		s.flush()
	}
	s.mu.Lock()
	err := s.err
	s.err = nil
	s.mu.Unlock()
	return err
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

// write stores one place and the active row in one transaction.
func (s *Session) write(body Place) error {
	body.Screen = s.screen
	args := "NULL"
	if body.Args != nil {
		raw, err := json.Marshal(body.Args)
		if err != nil {
			return err
		}
		args = sqlxdb.Quote(string(raw))
	}
	ticks, _ := processStartTicks(s.pid)
	id := sqlxdb.Quote(s.id)
	return sqlxdb.Exec("INSERT INTO tui_session(id, screen, cwd, path, cursor, args) VALUES (" +
		id + ", " + sqlxdb.Quote(s.screen) + ", " + sqlxdb.Quote(body.CWD) + ", " + sqlxdb.Quote(body.Path) + ", " +
		sqlxdb.Quote(body.Cursor) + ", " + args + ") ON CONFLICT(id) DO UPDATE SET screen = excluded.screen, " +
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
