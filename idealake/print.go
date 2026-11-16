package idealake

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type ideaJSON struct {
	ID       string   `json:"id"`
	Topic    string   `json:"topic"`
	Text     string   `json:"text"`
	Created  string   `json:"created"`
	Embedded bool     `json:"embedded"`
	Score    *float64 `json:"score,omitempty"`
}

type ideasDoc struct {
	Ideas []ideaJSON `json:"ideas"`
}

type topicsDoc struct {
	Topics []topicCount `json:"topics"`
}

type topicCount struct {
	Topic string `json:"topic"`
	N     int    `json:"n"`
}

type removedDoc struct {
	Removed string `json:"removed"`
}

func (d *desk) printIdeas(ideas []Idea) int {
	if d.json {
		views := make([]ideaJSON, 0, len(ideas))
		for _, idea := range ideas {
			views = append(views, viewIdea(idea))
		}
		return d.emit(ideasDoc{Ideas: views})
	}
	if len(ideas) == 0 {
		fmt.Fprintln(d.stdout, "no ideas")
		return 0
	}
	for _, idea := range ideas {
		fmt.Fprintf(d.stdout, "%s  %s  %s\n", idea.ID, idea.Topic, idea.Text)
	}
	return 0
}

func (d *desk) printHits(hits []hit) int {
	if d.json {
		views := make([]ideaJSON, 0, len(hits))
		for _, item := range hits {
			view := viewIdea(item.Idea)
			score := item.Score
			view.Score = &score
			views = append(views, view)
		}
		return d.emit(ideasDoc{Ideas: views})
	}
	if len(hits) == 0 {
		fmt.Fprintln(d.stdout, "no ideas")
		return 0
	}
	for _, item := range hits {
		score := strconv.FormatFloat(item.Score, 'f', 3, 64)
		fmt.Fprintf(d.stdout, "%s  %s  %s  %s\n", score, item.ID, item.Topic, item.Text)
	}
	return 0
}

func (d *desk) printTopics(counts []topicCount) int {
	if d.json {
		if counts == nil {
			counts = []topicCount{}
		}
		return d.emit(topicsDoc{Topics: counts})
	}
	if len(counts) == 0 {
		fmt.Fprintln(d.stdout, "no topics")
		return 0
	}
	for _, count := range counts {
		fmt.Fprintf(d.stdout, "%d  %s\n", count.N, count.Topic)
	}
	return 0
}

func viewIdea(idea Idea) ideaJSON {
	return ideaJSON{
		ID:       idea.ID,
		Topic:    idea.Topic,
		Text:     idea.Text,
		Created:  idea.Created,
		Embedded: idea.Model == embedModel && len(idea.Embedding) > 0,
	}
}

func (d *desk) emit(v any) int {
	enc := json.NewEncoder(d.stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(d.stderr, "idealake: %s\n", err.Error())
		return 1
	}
	return 0
}
