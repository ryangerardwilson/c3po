package gmail

import (
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/gmail/internal/query"
)

const listShape = "Use: c3po gmail <preset> list [unread|read|sent|starred|external|thread <id>] [from <sender>] [containing <text>] [since <window>] [limit <count>] [with attachments] [open] [output json]"

const sendShape = "Use: c3po gmail <preset> send to <email|alias> subject <subject> body <body>|body from <draft_path> [cc <emails>] [bcc <emails>] [attach <path> ...]"

const replyShape = "Use: c3po gmail <preset> reply to <message_id|thread <thread_id>> [all] [in editor|body <body>|body from <draft_path>] [cc <emails>] [bcc <emails>] [attach <path> ...]"

type listQuery struct {
	Terms       []string
	Limit       *int
	Thread      string
	Open        bool
	Attachments bool
	JSON        bool
	Audit       string
	Sent        bool
}

type compose struct {
	Editor  bool
	To      string
	Subject string
	Body    string
	Draft   string
	CC      string
	BCC     string
	Attach  []string
	Thread  bool
	All     bool
	Target  string
}

func parseList(params []string) (listQuery, error) {
	var q listQuery
	i := 0
	if i < len(params) {
		switch params[i] {
		case "unread", "read", "starred", "external", "sent":
			q.Terms = append(q.Terms, params[i])
			q.Sent = params[i] == "sent"
			i++
		case "audit-unread":
			q.Audit = "unread"
			i++
		case "audit-read":
			q.Audit = "read"
			i++
		case "thread":
			if i+1 >= len(params) {
				return q, errs.Usage("%s", listShape)
			}
			q.Thread = params[i+1]
			i += 2
		}
	}
	for i < len(params) {
		switch params[i] {
		case "from", "containing", "since", "limit":
			if i+1 >= len(params) {
				return q, errs.Usage("%s requires: <%s>", params[i], listNeed(params[i]))
			}
			if err := applyListTerm(&q, params[i], params[i+1]); err != nil {
				return q, err
			}
			i += 2
		case "with":
			if i+1 >= len(params) || params[i+1] != "attachments" {
				return q, errs.Usage("%s", listShape)
			}
			q.Attachments = true
			i += 2
		case "open":
			q.Open = true
			i++
		case "output":
			if i+1 >= len(params) || params[i+1] != "json" {
				return q, errs.Usage("%s", listShape)
			}
			q.JSON = true
			i += 2
		default:
			return q, errs.Usage("%s", listShape)
		}
	}
	if q.Open && q.JSON {
		return q, errs.Usage("Use either open or output json, not both")
	}
	return q, nil
}

func listNeed(word string) string {
	switch word {
	case "from":
		return "sender"
	case "containing":
		return "text"
	case "since":
		return "window"
	default:
		return "count"
	}
}

func applyListTerm(q *listQuery, key, value string) error {
	switch key {
	case "from":
		q.Terms = append(q.Terms, "from:"+value)
	case "containing":
		q.Terms = append(q.Terms, value)
	case "since":
		term, err := query.TimeLimit(value)
		if err != nil {
			return err
		}
		q.Terms = append(q.Terms, term)
	case "limit":
		n, err := positive(value, "limit requires: <count>")
		if err != nil {
			return err
		}
		q.Limit = &n
	}
	return nil
}

func parseSend(params []string) (compose, error) {
	if len(params) == 2 && params[0] == "in" && params[1] == "editor" {
		return compose{Editor: true}, nil
	}
	c := cursor{args: params}
	var out compose
	var err error
	if out.To, err = c.word("to", sendShape); err != nil {
		return out, err
	}
	if out.Subject, err = c.word("subject", sendShape); err != nil {
		return out, err
	}
	if err = c.body(&out, sendShape); err != nil {
		return out, err
	}
	return out, c.tail(&out, sendShape)
}

func parseReply(params []string) (compose, error) {
	if len(params) < 2 || params[0] != "to" {
		return compose{}, errs.Usage("%s", replyShape)
	}
	c := cursor{args: params, i: 1}
	var out compose
	if c.peek() == "thread" {
		out.Thread = true
		c.i++
		if c.i >= len(c.args) {
			return out, errs.Usage("%s", replyShape)
		}
	}
	out.Target = c.args[c.i]
	c.i++
	if c.peek() == "all" {
		out.All = true
		c.i++
	}
	if c.i+1 < len(c.args) && c.args[c.i] == "in" && c.args[c.i+1] == "editor" {
		out.Editor = true
		c.i += 2
		return out, c.tail(&out, replyShape)
	}
	if err := c.body(&out, replyShape); err != nil {
		return out, err
	}
	return out, c.tail(&out, replyShape)
}

type cursor struct {
	args []string
	i    int
}

func (c *cursor) peek() string {
	if c.i >= len(c.args) {
		return ""
	}
	return c.args[c.i]
}

func (c *cursor) word(key, shape string) (string, error) {
	if c.i+1 >= len(c.args) || c.args[c.i] != key {
		return "", errs.Usage("%s", shape)
	}
	v := c.args[c.i+1]
	c.i += 2
	return v, nil
}

func (c *cursor) body(out *compose, shape string) error {
	if c.peek() != "body" || c.i+1 >= len(c.args) {
		return errs.Usage("%s", shape)
	}
	c.i++
	if c.peek() == "from" {
		if c.i+1 >= len(c.args) {
			return errs.Usage("body from requires: <draft_path>")
		}
		out.Draft = c.args[c.i+1]
		c.i += 2
		return nil
	}
	out.Body = c.args[c.i]
	c.i++
	return nil
}

func (c *cursor) tail(out *compose, shape string) error {
	for c.i < len(c.args) {
		switch c.args[c.i] {
		case "cc", "bcc":
			if c.i+1 >= len(c.args) {
				return errs.Usage("%s requires: <emails>", c.args[c.i])
			}
			if c.args[c.i] == "cc" {
				out.CC = c.args[c.i+1]
			} else {
				out.BCC = c.args[c.i+1]
			}
			c.i += 2
		case "attach":
			c.i++
			if c.i >= len(c.args) {
				return errs.Usage("attach requires at least one path")
			}
			for c.i < len(c.args) && !word(c.args[c.i], "cc", "bcc", "attach") {
				out.Attach = append(out.Attach, c.args[c.i])
				c.i++
			}
			if len(out.Attach) == 0 {
				return errs.Usage("attach requires at least one path")
			}
		default:
			return errs.Usage("%s", shape)
		}
	}
	return nil
}

func positive(value, shape string) (int, error) {
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, errs.Usage("%s", shape)
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 {
		return 0, errs.Usage("%s", shape)
	}
	return n, nil
}

func csv(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
