package tui

import (
	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

// Run opens the screen. A non-terminal returns before a session is written.
func Run(id string) error {
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	sess, err := tuisession.Open("slack", id)
	if err != nil {
		return err
	}
	defer sess.Close()
	cfg, err := app.LoadScreenConfig()
	if err != nil {
		return err
	}
	m := newModel(presetsFrom(app.ScreenPresets(cfg)), newLive(cfg), sess)
	return m.loop(term)
}

func presetsFrom(in []app.Preset) []Preset {
	out := make([]Preset, len(in))
	for i, preset := range in {
		out[i] = Preset{ID: preset.ID, Name: preset.Name}
	}
	return out
}

type live struct {
	cfg app.Config
}

func newLive(cfg app.Config) *live {
	return &live{cfg: cfg}
}

func (l *live) client(preset string) (app.SlackClient, error) {
	account, err := app.AccountFor(l.cfg, preset)
	if err != nil {
		return app.SlackClient{}, err
	}
	return app.ScreenClient(account)
}

func (l *live) Conversations(preset string) ([]Conversation, error) {
	client, err := l.client(preset)
	if err != nil {
		return nil, err
	}
	rows, err := app.ScreenConversations(client)
	if err != nil {
		return nil, err
	}
	return conversationsFrom(rows), nil
}

func (l *live) History(preset, channel string) ([]Message, error) {
	client, err := l.client(preset)
	if err != nil {
		return nil, err
	}
	rows, err := app.ScreenHistory(client, channel)
	if err != nil {
		return nil, err
	}
	return messagesFrom(rows), nil
}

func (l *live) Send(preset, channel, text, thread string) error {
	client, err := l.client(preset)
	if err != nil {
		return err
	}
	return app.ScreenSend(client, channel, text, thread)
}

func (l *live) Mark(preset, channel, ts string) error {
	client, err := l.client(preset)
	if err != nil {
		return err
	}
	return app.ScreenMark(client, channel, ts)
}

func conversationsFrom(in []app.Conversation) []Conversation {
	out := make([]Conversation, len(in))
	for i, row := range in {
		out[i] = Conversation{
			ID:      row.ID,
			Name:    row.Name,
			Surface: row.Surface,
			Unread:  row.Unread,
			Latest:  row.Latest,
		}
	}
	return out
}

func messagesFrom(in []app.ChatMessage) []Message {
	out := make([]Message, len(in))
	for i, row := range in {
		out[i] = Message{TS: row.TS, ThreadTS: row.ThreadTS, User: row.User, Text: row.Text}
	}
	return out
}
