// Package lake is idealake's domain: ideas in topics, their rules, their ids,
// and how they rank. It reaches storage and embeddings only through Store and
// Embedder, which the idealake command implements.
package lake

import (
	"sort"
	"strconv"
	"time"
)

const (
	MaxIdea       = 300
	MaxTopic      = 80
	SearchLimit   = 20
	EmbedModel    = "v1"
	PassagePrefix = "passage: "
	QueryPrefix   = "query: "
)

// Idea is one stored line in a topic. Embedding is never printed.
type Idea struct {
	ID        string
	Topic     string
	Text      string
	Created   string
	Model     string
	Embedding []float64
}

// TopicCount is a topic and how many ideas it holds.
type TopicCount struct {
	Topic string `json:"topic"`
	N     int    `json:"n"`
}

// Store keeps topics and ideas.
type Store interface {
	Ideas() ([]Idea, error)
	TopicNames() ([]string, error)
	// LastIssued is the highest idea id ever issued, including removed ones.
	LastIssued() (int, error)
	// SaveNew stores new ideas and their topic together.
	SaveNew(ideas []Idea) error
	EnsureTopic(topic string) error
	Delete(id string) error
}

// AssignIDs gives each new text the next id after every id ever issued.
// lastIssued is the highest id the store ever issued; a removed id is never
// issued again.
func AssignIDs(lastIssued int, existing []Idea, texts []string, topic string, vectors [][]float64, now time.Time) []Idea {
	next := lastIssued
	if n := MaxID(existing); n > next {
		next = n
	}
	created := now.UTC().Format(time.RFC3339)
	ideas := make([]Idea, len(texts))
	for i, text := range texts {
		next++
		idea := Idea{ID: strconv.Itoa(next), Topic: topic, Text: text, Created: created}
		if vectors != nil {
			idea.Model = EmbedModel
			idea.Embedding = vectors[i]
		}
		ideas[i] = idea
	}
	return ideas
}

// MaxID is the highest numeric id among ideas.
func MaxID(ideas []Idea) int {
	max := 0
	for _, idea := range ideas {
		n, err := strconv.Atoi(idea.ID)
		if err == nil && n > max {
			max = n
		}
	}
	return max
}

// FindIdea returns the idea with id.
func FindIdea(ideas []Idea, id string) (Idea, bool) {
	for _, idea := range ideas {
		if idea.ID == id {
			return idea, true
		}
	}
	return Idea{}, false
}

// FilterTopic keeps the ideas in topic; an empty topic keeps all.
func FilterTopic(ideas []Idea, topic string) []Idea {
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

// IDLess orders ids numerically, then as text.
func IDLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil && ai != bi {
		return ai < bi
	}
	return a < b
}

// CountTopic counts the ideas in topic.
func CountTopic(ideas []Idea, topic string) int {
	n := 0
	for _, idea := range ideas {
		if idea.Topic == topic {
			n++
		}
	}
	return n
}

// MergeTopics adds every named topic with no ideas to counts, sorted by topic.
func MergeTopics(counts []TopicCount, names []string) []TopicCount {
	seen := map[string]bool{}
	for _, count := range counts {
		seen[count.Topic] = true
	}
	for _, name := range names {
		if seen[name] {
			continue
		}
		counts = append(counts, TopicCount{Topic: name, N: 0})
		seen[name] = true
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Topic < counts[j].Topic
	})
	return counts
}

// TopicCounts counts the ideas in each topic, sorted by topic.
func TopicCounts(ideas []Idea) []TopicCount {
	counts := map[string]int{}
	var names []string
	for _, idea := range ideas {
		if _, ok := counts[idea.Topic]; !ok {
			names = append(names, idea.Topic)
		}
		counts[idea.Topic]++
	}
	sort.Strings(names)
	out := make([]TopicCount, 0, len(names))
	for _, name := range names {
		out = append(out, TopicCount{Topic: name, N: counts[name]})
	}
	return out
}
