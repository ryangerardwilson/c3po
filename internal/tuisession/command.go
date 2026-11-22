package tuisession

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// placeJSON is a place as c3po __session reads and prints it.
type placeJSON struct {
	Screen string   `json:"screen"`
	CWD    string   `json:"cwd,omitempty"`
	Path   string   `json:"path,omitempty"`
	Cursor string   `json:"cursor,omitempty"`
	Args   []string `json:"args,omitempty"`
}

// ActiveRow is one running screen and the place its session holds.
type ActiveRow struct {
	SessionID string `json:"session_id"`
	PID       int    `json:"pid"`
	PIDStart  int64  `json:"pid_start"`
	Screen    string `json:"screen"`
	CWD       string `json:"cwd"`
}

// Command is c3po __session, the one interface through which a screen that
// runs outside c3po, and a reader such as tesseract, use screen sessions.
// It prints JSON on stdout.
//
//	open <screen> <pid> [<id>]     the session id and the restored place
//	note <id> <pid> <place json>   merge the place's non-empty fields
//	place <id> <pid> <path> <cursor>
//	close <id>                     drop the active row
//	active                         every running screen and its place
func Command(args []string, stdout, stderr io.Writer) int {
	out, err := command(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if out != nil {
		raw, err := json.Marshal(out)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, string(raw))
	}
	return 0
}

func command(args []string) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("c3po: __session needs open, note, place, close, or active")
	}
	switch {
	case args[0] == "open" && (len(args) == 3 || len(args) == 4):
		return commandOpen(args[1:])
	case args[0] == "note" && len(args) == 4:
		var note placeJSON
		if err := json.Unmarshal([]byte(args[3]), &note); err != nil {
			return nil, err
		}
		return nil, withSession(args[1], args[2], func(s *Session) error {
			return s.Note(Place{CWD: note.CWD, Path: note.Path, Cursor: note.Cursor, Args: note.Args})
		})
	case args[0] == "place" && len(args) == 5:
		return nil, withSession(args[1], args[2], func(s *Session) error { return s.Place(args[3], args[4]) })
	case args[0] == "close" && len(args) == 2:
		return nil, (&Session{id: args[1]}).Close()
	case args[0] == "active" && len(args) == 1:
		return Active()
	}
	return nil, errors.New("c3po: __session needs open, note, place, close, or active")
}

func commandOpen(args []string) (any, error) {
	pid, err := strconv.Atoi(args[1])
	if err != nil || pid <= 0 {
		return nil, errors.New("c3po: __session needs a process id")
	}
	id := ""
	if len(args) == 3 {
		id = args[2]
	}
	sess, err := open(args[0], id, pid)
	if err != nil {
		return nil, err
	}
	out := struct {
		ID       string     `json:"id"`
		Restored bool       `json:"restored"`
		Place    *placeJSON `json:"place,omitempty"`
	}{ID: sess.ID()}
	if body, ok := sess.Restored(); ok {
		out.Restored = true
		out.Place = &placeJSON{Screen: body.Screen, CWD: body.CWD, Path: body.Path, Cursor: body.Cursor, Args: body.Args}
	}
	return out, nil
}

// withSession runs change on the stored session id as process pid.
func withSession(id, pidText string, change func(*Session) error) error {
	pid, err := strconv.Atoi(pidText)
	if err != nil || pid <= 0 {
		return errors.New("c3po: __session needs a process id")
	}
	body, ok, err := load(id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("c3po: session %s was not found", id)
	}
	return change(&Session{id: id, screen: body.Screen, body: body, restored: true, pid: pid})
}

// Active lists every running screen with the place its session holds.
func Active() ([]ActiveRow, error) {
	rows := []ActiveRow{}
	err := sqlxdb.Query("SELECT a.session_id, a.pid, a.pid_start, s.screen, s.cwd FROM tui_active a JOIN tui_session s ON s.id = a.session_id ORDER BY a.session_id", &rows)
	return rows, err
}
