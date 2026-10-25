package gmail

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/gmail/internal/spam"
)

const unreadQuery = "is:unread -in:sent"

// tidy trashes mail that matches each preset's spam list.
func (e Env) tidy() (int, error) {
	app, err := e.load()
	if err != nil {
		return 0, err
	}
	if len(app.Presets) == 0 {
		return 0, errs.Usage("No configured presets found.")
	}
	totalGroups := 0
	for _, preset := range app.Presets {
		totalGroups += len(senderGroups(app.Accounts[preset]))
	}
	bar := newCleanBar(e.Stdout, "", totalGroups)
	if totalGroups > 0 {
		bar.start()
	}
	done := 0
	trashed := 0
	var rows []tidyRow
	for _, preset := range app.Presets {
		acct := app.Accounts[preset]
		groups := len(senderGroups(acct))
		svc, err := e.service(acct)
		if err != nil {
			if totalGroups > 0 {
				bar.fail()
			}
			return 0, err
		}
		n, err := cleanSenders(svc, acct, &bar, done, false)
		if err != nil {
			return 0, err
		}
		done += groups
		trashed += n
		rows = append(rows, tidyRow{acct.Preset, acct.Email, n})
	}
	if totalGroups > 0 {
		bar.hold()
		fmt.Fprint(e.Stdout, tidyTable(rows))
	}
	fmt.Fprintf(e.Stdout, "gmail tidy complete: presets=%d trashed=%d\n", len(app.Presets), trashed)
	return 0, nil
}

type tidyRow struct {
	preset string
	email  string
	n      int
}

func tidyTable(rows []tidyRow) string {
	presetW, emailW, trashedW := len("preset"), len("email"), len("trashed")
	cells := make([][3]string, len(rows))
	for i, row := range rows {
		cells[i] = [3]string{row.preset, row.email, fmt.Sprintf("%d", row.n)}
		if len(cells[i][0]) > presetW {
			presetW = len(cells[i][0])
		}
		if len(cells[i][1]) > emailW {
			emailW = len(cells[i][1])
		}
		if len(cells[i][2]) > trashedW {
			trashedW = len(cells[i][2])
		}
	}
	rule := func(left, mid, right string) string {
		return left + strings.Repeat("─", presetW+2) + mid + strings.Repeat("─", emailW+2) + mid + strings.Repeat("─", trashedW+2) + right
	}
	cell := func(a, b, c string) string {
		return "│ " + padRight(a, presetW) + " │ " + padRight(b, emailW) + " │ " + padLeft(c, trashedW) + " │"
	}
	var b strings.Builder
	fmt.Fprintln(&b, rule("┌", "┬", "┐"))
	fmt.Fprintln(&b, cell("preset", "email", "trashed"))
	fmt.Fprintln(&b, rule("├", "┼", "┤"))
	for _, row := range cells {
		fmt.Fprintln(&b, cell(row[0], row[1], row[2]))
	}
	fmt.Fprintln(&b, rule("└", "┴", "┘"))
	return b.String()
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func padLeft(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat(" ", n-len(s)) + s
}

// tidyAudit lists unread senders that are not already spam, busiest first.
func (e Env) tidyAudit() (int, error) {
	app, err := e.load()
	if err != nil {
		return 0, err
	}
	if len(app.Presets) == 0 {
		return 0, errs.Usage("No configured presets found.")
	}
	senders := 0
	for _, preset := range app.Presets {
		acct := app.Accounts[preset]
		svc, err := e.service(acct)
		if err != nil {
			return 0, err
		}
		msgs, err := svc.List(unreadQuery, 0)
		if err != nil {
			return 0, err
		}
		counts := map[string]int{}
		for _, msg := range msgs {
			from := strings.ToLower(strings.TrimSpace(summarize(msg, app.TimezoneOffset, true).FromEmail))
			if from == "" {
				continue
			}
			counts[from]++
		}
		for _, item := range spam.Rank(counts, acct.SpamSenders, acct.SpamExcludes) {
			fmt.Fprintf(e.Stdout, "%s\t%s\tunread=%d\n", preset, item.Sender, item.Unread)
			senders++
		}
	}
	fmt.Fprintf(e.Stdout, "gmail tidy audit complete: presets=%d senders=%d\n", len(app.Presets), senders)
	return 0, nil
}
