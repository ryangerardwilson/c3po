package gcal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/gcal/internal/calendar"
	"github.com/ryangerardwilson/c3po/gcal/internal/config"
	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
	"github.com/ryangerardwilson/c3po/gcal/internal/oauth"
	"github.com/ryangerardwilson/c3po/gcal/internal/pyfmt"
	"github.com/ryangerardwilson/c3po/gcal/internal/transcripts"
)

const (
	listShape       = "Use: c3po gcal <preset> list limit <n> [output json]"
	previewShape    = "Use: c3po gcal <preset> preview create event ... | preview reschedule event ..."
	previewCreate   = "Use: c3po gcal <preset> preview create event <title> from <start> to <end> invite <csv>"
	previewResched  = "Use: c3po gcal <preset> preview reschedule event <id> from <start> to <end>"
	exportShape     = "Use: c3po gcal <preset> export event <id>"
	deleteShape     = "Use: c3po gcal <preset> delete event <id>"
	rescheduleShape = "Use: c3po gcal <preset> reschedule event <id> from <start> to <end>"
	createShape     = "Use: c3po gcal <preset> create event <title> from <start> to <end> invite <csv>"
	actionShape     = "Use: c3po gcal <preset> list|create|delete|reschedule|export"
)

// presetCmd is one command against a configured preset.
type presetCmd struct {
	e        *env
	app      config.App
	account  config.Account
	args     []string
	calendar calendar.Client
}

func dispatch(args []string, e *env) error {
	switch args[0] {
	case "auth":
		return auth(args[1:], e)
	case "accounts":
		if len(args) < 2 || args[1] != "list" {
			return errs.Usage("Use: c3po gcal accounts list [output json]")
		}
		return accountsList(args[2:], e)
	case "setup":
		if len(args) < 2 || args[1] != "check" {
			return errs.Usage("Use: c3po gcal setup check [output json]")
		}
		return setupCheck(args[2:], e)
	}
	if !config.IsDigits(args[0]) {
		return errs.Usage("preset must be numeric, like `1` or `2`")
	}
	if len(args) == 1 {
		return errs.Usage("Use: c3po gcal <preset> list limit <n>")
	}
	return runPreset(args[0], args[1:], e)
}

func runPreset(preset string, args []string, e *env) error {
	app, err := loadConfig(e)
	if err != nil {
		return err
	}
	acct, err := config.Get(app, preset)
	if err != nil {
		return err
	}
	if args[0] == "preview" {
		return preview(args[1:], app.Timezone, e)
	}
	token, err := e.oauth().AccessToken(acct.Email, acct.ClientSecretFile, oauth.CalendarScopes)
	if err != nil {
		return err
	}
	c := presetCmd{e: e, app: app, account: acct, args: args,
		calendar: calendar.Client{HTTP: e.http, Base: e.calendarBase, Token: token}}
	return c.run()
}

func (c presetCmd) run() error {
	switch c.args[0] {
	case "list":
		return c.list()
	case "export":
		return c.export()
	case "delete":
		return c.delete()
	case "reschedule":
		return c.reschedule()
	case "create":
		return c.create()
	default:
		return errs.Usage(actionShape)
	}
}

func (c presetCmd) list() error {
	params, asJSON := c.args[1:], false
	if n := len(params); n >= 2 && params[n-2] == "output" && params[n-1] == "json" {
		params, asJSON = params[:n-2], true
	} else if contains(params, "output") {
		return errs.Usage(listShape)
	}
	count, err := parseLimit(params)
	if err != nil {
		return err
	}
	events, err := c.calendar.ListUpcoming(count, calendar.NonRecurring, c.e.now())
	if err != nil {
		return err
	}
	printEvents(c.e.stdout, events, c.app.Timezone, asJSON)
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func parseLimit(params []string) (int, error) {
	if len(params) != 2 || params[0] != "limit" {
		return 0, errs.Usage(listShape)
	}
	count, ok := pyInt(params[1])
	if !ok {
		return 0, errs.Usage("limit must be a positive integer")
	}
	if count <= 0 {
		return 0, errs.Usage("limit must be > 0")
	}
	return count, nil
}

func (c presetCmd) export() error {
	if len(c.args) != 3 || c.args[1] != "event" {
		return errs.Usage(exportShape)
	}
	driveToken, err := c.e.oauth().AccessToken(c.account.Email, c.account.ClientSecretFile, oauth.TranscriptScopes)
	if err != nil {
		return err
	}
	ev, err := c.calendar.Get(c.args[2])
	if err != nil {
		return err
	}
	att, err := transcripts.Find(ev)
	if err != nil {
		return err
	}
	drive := calendar.Client{HTTP: c.e.http, Base: c.e.driveBase, Token: driveToken}
	text, err := drive.ExportText(att.FileID)
	if err != nil {
		return err
	}
	dirs := c.e.dirs()
	if err := dirs.Ensure(); err != nil {
		return err
	}
	path, err := transcripts.Save(dirs.Transcripts(), text, ev.ID, ev.Title("(untitled)"), ev.Start.Value())
	if err != nil {
		return err
	}
	fmt.Fprintf(c.e.stdout, "saved transcript\tevent_id=%s\tfile_id=%s\n%s\n", ev.ID, att.FileID, path)
	return nil
}

func (c presetCmd) delete() error {
	if len(c.args) != 3 || c.args[1] != "event" {
		return errs.Usage(deleteShape)
	}
	if err := c.calendar.Delete(c.args[2]); err != nil {
		return err
	}
	fmt.Fprintf(c.e.stdout, "deleted\tevent_id=%s\n", c.args[2])
	return nil
}

func shapeIs(args []string, want map[int]string, n int) bool {
	if len(args) != n {
		return false
	}
	for i, word := range want {
		if args[i] != word {
			return false
		}
	}
	return true
}

func (c presetCmd) reschedule() error {
	if !shapeIs(c.args, map[int]string{1: "event", 3: "from", 5: "to"}, 7) {
		return errs.Usage(rescheduleShape)
	}
	start, end, err := parseWindow(c.args[4], c.args[6], c.app.Timezone)
	if err != nil {
		return err
	}
	ev, err := c.calendar.Reschedule(c.args[2], start, end, c.app.Timezone)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.e.stdout, "rescheduled\tevent_id=%s\n%s\n", ev.ID, ev.MeetingURL())
	return nil
}

func (c presetCmd) create() error {
	if !shapeIs(c.args, map[int]string{1: "event", 3: "from", 5: "to", 7: "invite"}, 9) {
		return errs.Usage(createShape)
	}
	start, end, err := parseWindow(c.args[4], c.args[6], c.app.Timezone)
	if err != nil {
		return err
	}
	invitees, err := parseInvitees(c.args[8])
	if err != nil {
		return err
	}
	ev, err := c.calendar.Create(c.args[2], start, end, c.app.Timezone, invitees, c.e.newID())
	if err != nil {
		return err
	}
	fmt.Fprintf(c.e.stdout, "created\tevent_id=%s\n%s\n", ev.ID, ev.MeetingURL())
	return nil
}

// preview validates a create or reschedule and prints what it would
// send, without credentials or network.
func preview(args []string, timezone string, e *env) error {
	if len(args) == 0 {
		return errs.Usage(previewShape)
	}
	switch args[0] {
	case "create":
		return previewCreateEvent(args[1:], timezone, e)
	case "reschedule":
		return previewRescheduleEvent(args[1:], timezone, e)
	default:
		return errs.Usage(previewShape)
	}
}

func previewCreateEvent(rest []string, timezone string, e *env) error {
	if !shapeIs(rest, map[int]string{0: "event", 2: "from", 4: "to", 6: "invite"}, 8) {
		return errs.Usage(previewCreate)
	}
	start, end, err := parseWindow(rest[3], rest[5], timezone)
	if err != nil {
		return err
	}
	invitees, err := parseInvitees(rest[7])
	if err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "preview_create_event title=%s from=%s to=%s timezone=%s invite=%s meet=yes side_effects=none\n",
		pyfmt.Repr(rest[1]), isoformat(start), isoformat(end), timezone, strings.Join(invitees, ","))
	return nil
}

func previewRescheduleEvent(rest []string, timezone string, e *env) error {
	if !shapeIs(rest, map[int]string{0: "event", 2: "from", 4: "to"}, 6) {
		return errs.Usage(previewResched)
	}
	start, end, err := parseWindow(rest[3], rest[5], timezone)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "preview_reschedule_event event_id=%s from=%s to=%s timezone=%s side_effects=none\n",
		rest[1], isoformat(start), isoformat(end), timezone)
	return nil
}

var intPattern = regexp.MustCompile(`^[+-]?\d+(_\d+)*$`)

// pyInt parses an integer the way Python int() reads a decimal string:
// surrounding whitespace, a sign, and single underscores between digits.
func pyInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if !intPattern.MatchString(s) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.ReplaceAll(s, "_", ""))
	return n, err == nil
}
