package tui

import (
	"strings"
	"unicode/utf8"
)

const (
	whenWidth = 30
	whoWidth  = 18
)

// Item is one desk row.
type Item struct {
	MessageID    string
	Conversation string
	Sender       string
	Text         string
	Date         string
	Why          string
}

// Workspace is one Slack account on the desk.
type Workspace struct {
	Preset string
	Name   string
	Error  string
	DMs    []Item
	Tagged []Item
	Aware  []Item
}

// Board is the desk the screen paints.
type Board struct {
	RefreshedAt string
	Since       string
	Error       string
	Workspaces  []Workspace
}

type bucket struct {
	label string
	items []Item
}

func (ws Workspace) buckets() []bucket {
	return []bucket{
		{label: "DMS", items: ws.DMs},
		{label: "TAGGED", items: ws.Tagged},
		{label: "SHOULD KNOW", items: ws.Aware},
	}
}

func (ws Workspace) items() []Item {
	out := make([]Item, 0, len(ws.DMs)+len(ws.Tagged)+len(ws.Aware))
	for _, part := range ws.buckets() {
		out = append(out, part.items...)
	}
	return out
}

func clean(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func whoOf(item Item) string {
	if name := clean(item.Sender); name != "" {
		return name
	}
	return clean(item.Conversation)
}

func bodyOf(item Item) string {
	if why := clean(item.Why); why != "" {
		return why
	}
	return clean(item.Text)
}

func pad(value string, width int) string {
	n := utf8.RuneCountInString(value)
	if n >= width {
		return string([]rune(value)[:width])
	}
	return value + strings.Repeat(" ", width-n)
}

func columnHead() string {
	return pad("WHEN", whenWidth) + "  " + pad("WHO", whoWidth) + "  WHAT"
}

func itemLine(item Item) string {
	return pad(clean(item.Date), whenWidth) + "  " + pad(whoOf(item), whoWidth) + "  " + bodyOf(item)
}
