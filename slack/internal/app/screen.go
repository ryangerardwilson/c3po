package app

import "sort"

// Preset is one Slack workspace the screen can open. It has no token.
type Preset struct {
	ID   string
	Name string
}

// Conversation is one channel or direct message.
type Conversation struct {
	ID      string
	Name    string
	Surface string
	Unread  int
	Latest  string
}

// ChatMessage is one history row, oldest first when returned by ScreenHistory.
type ChatMessage struct {
	TS       string
	ThreadTS string
	User     string
	Text     string
}

// LoadScreenConfig reads the c3po slack account file.
func LoadScreenConfig() (Config, error) {
	return loadConfig(configPath(""))
}

// ScreenPresets lists workspaces in preset order. Names come from the account.
func ScreenPresets(cfg Config) []Preset {
	accts := accounts(cfg)
	keys := sortedPresetKeys(accts)
	out := make([]Preset, 0, len(keys))
	for _, key := range keys {
		name := str(accts[key]["name"])
		if name == "" {
			name = key
		}
		out = append(out, Preset{ID: key, Name: name})
	}
	return out
}

// AccountFor returns the account for one preset.
func AccountFor(cfg Config, preset string) (Account, error) {
	_, account, err := selectAccount(cfg, preset)
	return account, err
}

// ScreenClient is a Slack client for one account. The token stays on the client.
func ScreenClient(slack Connector, account Account) (Session, error) {
	token, err := resolveToken(account)
	if err != nil {
		return Session{}, err
	}
	return Session{
		Token:   token,
		Gateway: slack.Connect(ConnectOptions{Token: token, Timeout: defaultListHTTPTimeout}),
	}, nil
}

// ScreenConversations lists direct messages and channels the user is in.
func ScreenConversations(client Session) ([]Conversation, error) {
	dms, err := listMemberDMs(client)
	if err != nil {
		return nil, err
	}
	channels, err := listMemberChannels(client)
	if err != nil {
		return nil, err
	}
	rows := make([]Conversation, 0, len(dms)+len(channels))
	rows = append(rows, mapConversations(client, dms)...)
	rows = append(rows, mapConversations(client, channels)...)
	sortConversations(rows)
	return rows, nil
}

func mapConversations(client Session, channels []map[string]any) []Conversation {
	rows := make([]Conversation, 0, len(channels))
	for _, channel := range channels {
		id := str(channel["id"])
		if id == "" {
			continue
		}
		rows = append(rows, Conversation{
			ID:      id,
			Name:    conversationLabel(client, channel, id),
			Surface: conversationSurface(channel, id),
			Unread:  unreadOf(channel),
			Latest:  extractTS(channel),
		})
	}
	return rows
}

func conversationLabel(client Session, channel map[string]any, id string) string {
	name := channelName(channel, id)
	if !userIDRE.MatchString(name) {
		return name
	}
	user, err := getUserInfo(client, name)
	if err != nil {
		return name
	}
	return displayUser(user, name)
}

func unreadOf(channel map[string]any) int {
	if _, ok := channel["unread_count_display"]; ok {
		return intValue(channel["unread_count_display"], 0)
	}
	if _, ok := channel["unread_count"]; ok {
		return intValue(channel["unread_count"], 0)
	}
	latest := extractTS(channel)
	last := str(channel["last_read"])
	if latest != "" && tsFloat(latest) > tsFloat(last) {
		return 1
	}
	return 0
}

func sortConversations(rows []Conversation) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Unread != rows[j].Unread {
			return rows[i].Unread > rows[j].Unread
		}
		if surfaceRank(rows[i].Surface) != surfaceRank(rows[j].Surface) {
			return surfaceRank(rows[i].Surface) < surfaceRank(rows[j].Surface)
		}
		return rows[i].Name < rows[j].Name
	})
}

func surfaceRank(surface string) int {
	switch surface {
	case "dm":
		return 0
	case "group_dm":
		return 1
	default:
		return 2
	}
}

// ScreenHistory returns the newest messages, oldest first.
func ScreenHistory(client Session, channelID string) ([]ChatMessage, error) {
	page, err := client.History(HistoryQuery{ChannelID: channelID, Limit: 40})
	if err != nil {
		return nil, err
	}
	raw := page.Messages
	rows := make([]ChatMessage, 0, len(raw))
	cache := map[string]map[string]any{}
	for i := len(raw) - 1; i >= 0; i-- {
		rows = append(rows, chatMessage(client, raw[i], cache))
	}
	return rows, nil
}

func chatMessage(client Session, message map[string]any, cache map[string]map[string]any) ChatMessage {
	sender := senderInfo(client, message, cache)
	text := messageText(message)
	if text == "" || text == "-" {
		text = "(no text)"
	}
	return ChatMessage{
		TS:       str(message["ts"]),
		ThreadTS: str(message["thread_ts"]),
		User:     firstNonEmpty(str(sender["name"]), str(message["user"]), "unknown"),
		Text:     text,
	}
}

// ScreenSend posts text. An empty threadTS is a new message, not a reply.
func ScreenSend(client Session, channelID, text, threadTS string) error {
	_, err := client.PostMessage(channelID, text, threadTS)
	return err
}

// ScreenMark marks one conversation read through the given timestamp.
func ScreenMark(client Session, channelID, ts string) error {
	return client.MarkRead(channelID, ts)
}
