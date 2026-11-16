package idealake

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// Idea is one stored line. Embedding is kept in sqlite and is not printed.
type Idea struct {
	ID        string    `json:"id"`
	Topic     string    `json:"topic"`
	Text      string    `json:"text"`
	Created   string    `json:"created"`
	Model     string    `json:"model,omitempty"`
	Embedding []float64 `json:"embedding,omitempty"`
}

var clock = time.Now

func loadIdeas() ([]Idea, error) {
	rows, err := sqlxdb.List(recordApp, recordKind)
	if err != nil {
		return nil, fmt.Errorf("idealake: %s", err)
	}
	ideas := make([]Idea, 0, len(rows))
	for _, row := range rows {
		var idea Idea
		if err := json.Unmarshal(row.Body, &idea); err != nil {
			return nil, fmt.Errorf("idealake: record %s", row.Key)
		}
		if idea.ID == "" {
			idea.ID = row.Key
		}
		ideas = append(ideas, idea)
	}
	sort.Slice(ideas, func(i, j int) bool {
		return idLess(ideas[i].ID, ideas[j].ID)
	})
	return ideas, nil
}

func saveIdea(idea Idea) error {
	body, err := json.Marshal(idea)
	if err != nil {
		return err
	}
	if err := sqlxdb.Put(recordApp, recordKind, idea.ID, body); err != nil {
		return fmt.Errorf("idealake: %s", err)
	}
	return nil
}

func deleteIdea(id string) error {
	if err := sqlxdb.Delete(recordApp, recordKind, id); err != nil {
		return fmt.Errorf("idealake: %s", err)
	}
	return nil
}

func saveNew(ideas []Idea) error {
	saved := make([]string, 0, len(ideas))
	for _, idea := range ideas {
		if err := saveIdea(idea); err != nil {
			for _, id := range saved {
				_ = deleteIdea(id)
			}
			return err
		}
		saved = append(saved, idea.ID)
	}
	if len(saved) == 0 {
		return nil
	}
	if err := rememberNext(saved[len(saved)-1]); err != nil {
		for _, id := range saved {
			_ = deleteIdea(id)
		}
		return err
	}
	return nil
}

func assignIDs(existing []Idea, texts []string, topic string, vectors [][]float64) ([]Idea, error) {
	next, err := nextBase(existing)
	if err != nil {
		return nil, err
	}
	created := clock().UTC().Format(time.RFC3339)
	ideas := make([]Idea, len(texts))
	for i, text := range texts {
		next++
		idea := Idea{ID: strconv.Itoa(next), Topic: topic, Text: text, Created: created}
		if vectors != nil {
			idea.Model = embedModel
			idea.Embedding = vectors[i]
		}
		ideas[i] = idea
	}
	return ideas, nil
}

func nextBase(existing []Idea) (int, error) {
	base := maxID(existing)
	body, ok, err := sqlxdb.Get(recordApp, "meta", "next")
	if err != nil {
		return 0, fmt.Errorf("idealake: %s", err)
	}
	if !ok {
		return base, nil
	}
	n, conv := strconv.Atoi(string(body))
	if conv == nil && n > base {
		return n, nil
	}
	return base, nil
}

func rememberNext(id string) error {
	n, err := strconv.Atoi(id)
	if err != nil {
		return nil
	}
	if err := sqlxdb.Put(recordApp, "meta", "next", []byte(strconv.Itoa(n))); err != nil {
		return fmt.Errorf("idealake: %s", err)
	}
	return nil
}

func maxID(ideas []Idea) int {
	max := 0
	for _, idea := range ideas {
		n, err := strconv.Atoi(idea.ID)
		if err == nil && n > max {
			max = n
		}
	}
	return max
}

func findIdea(ideas []Idea, id string) (Idea, bool) {
	for _, idea := range ideas {
		if idea.ID == id {
			return idea, true
		}
	}
	return Idea{}, false
}

func filterTopic(ideas []Idea, topic string) []Idea {
	if topic == "" {
		return ideas
	}
	out := make([]Idea, 0)
	for _, idea := range ideas {
		if idea.Topic == topic {
			out = append(out, idea)
		}
	}
	return out
}

func idLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil && ai != bi {
		return ai < bi
	}
	return a < b
}
