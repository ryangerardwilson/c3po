// Package store keeps idealake topics and ideas in the c3po database tables
// idealake_topic and idealake_idea. It is the lake.Store the command uses.
package store

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/idealake/internal/lake"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// SQL is a lake.Store. Now stamps a new topic.
type SQL struct {
	Now func() time.Time
}

type ideaRow struct {
	ID        int64   `json:"id"`
	Topic     string  `json:"topic"`
	Text      string  `json:"text"`
	Created   int64   `json:"created_at"`
	Model     *string `json:"model"`
	Embedding *string `json:"embedding"`
}

// Ideas loads every idea with its topic name, in id order.
func (s SQL) Ideas() ([]lake.Idea, error) {
	var rows []ideaRow
	sql := "SELECT i.id, t.name AS topic, i.text, i.created_at, i.model, i.embedding " +
		"FROM idealake_idea i JOIN idealake_topic t ON t.id = i.topic_id ORDER BY i.id"
	if err := sqlxdb.Query(sql, &rows); err != nil {
		return nil, fmt.Errorf("idealake: %s", err)
	}
	ideas := make([]lake.Idea, 0, len(rows))
	for _, row := range rows {
		idea := lake.Idea{
			ID:      strconv.FormatInt(row.ID, 10),
			Topic:   row.Topic,
			Text:    row.Text,
			Created: time.Unix(row.Created, 0).UTC().Format(time.RFC3339),
		}
		if row.Model != nil && row.Embedding != nil {
			idea.Model = *row.Model
			if err := json.Unmarshal([]byte(*row.Embedding), &idea.Embedding); err != nil {
				return nil, fmt.Errorf("idealake: idea %d", row.ID)
			}
		}
		ideas = append(ideas, idea)
	}
	return ideas, nil
}

// Delete removes one idea. A non-numeric id removes nothing.
func (s SQL) Delete(id string) error {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil
	}
	if err := sqlxdb.Exec("DELETE FROM idealake_idea WHERE id = " + strconv.FormatInt(n, 10)); err != nil {
		return fmt.Errorf("idealake: %s", err)
	}
	return nil
}

// SaveNew stores new ideas and their topic in one transaction. An id that is
// already taken fails the whole write.
func (s SQL) SaveNew(ideas []lake.Idea) error {
	if len(ideas) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(s.topicInsert(ideas[0].Topic))
	for _, idea := range ideas {
		created, err := time.Parse(time.RFC3339, idea.Created)
		if err != nil {
			return err
		}
		model, embedding := "NULL", "NULL"
		if idea.Embedding != nil {
			raw, err := json.Marshal(idea.Embedding)
			if err != nil {
				return err
			}
			model, embedding = sqlxdb.Quote(idea.Model), sqlxdb.Quote(string(raw))
		}
		b.WriteString(";\nINSERT INTO idealake_idea(id, topic_id, text, created_at, model, embedding) VALUES (" +
			idea.ID + ", (SELECT id FROM idealake_topic WHERE name = " + sqlxdb.Quote(idea.Topic) + "), " +
			sqlxdb.Quote(idea.Text) + ", " + strconv.FormatInt(created.Unix(), 10) + ", " + model + ", " + embedding + ")")
	}
	if err := sqlxdb.Exec(b.String()); err != nil {
		return fmt.Errorf("idealake: %s", err)
	}
	return nil
}

// LastIssued is the idea id sequence, which keeps the highest id ever issued.
func (s SQL) LastIssued() (int, error) {
	var rows []struct {
		Seq int `json:"seq"`
	}
	if err := sqlxdb.Query("SELECT seq FROM sqlite_sequence WHERE name = 'idealake_idea'", &rows); err != nil {
		return 0, fmt.Errorf("idealake: %s", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].Seq, nil
}

func (s SQL) topicInsert(topic string) string {
	return "INSERT OR IGNORE INTO idealake_topic(name, created_at) VALUES (" +
		sqlxdb.Quote(topic) + ", " + strconv.FormatInt(s.Now().UTC().Unix(), 10) + ")"
}

// EnsureTopic stores topic when it is missing.
func (s SQL) EnsureTopic(topic string) error {
	if err := sqlxdb.Exec(s.topicInsert(topic)); err != nil {
		return fmt.Errorf("idealake: %s", err)
	}
	return nil
}

// TopicNames lists every topic name.
func (s SQL) TopicNames() ([]string, error) {
	var rows []struct {
		Name string `json:"name"`
	}
	if err := sqlxdb.Query("SELECT name FROM idealake_topic ORDER BY name", &rows); err != nil {
		return nil, fmt.Errorf("idealake: %s", err)
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return names, nil
}
