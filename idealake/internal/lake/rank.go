package lake

import (
	"math"
	"sort"
	"strings"
)

// Hit is an idea and its similarity to a query.
type Hit struct {
	Idea
	Score float64
}

// Rank orders the embedded ideas by similarity to vectors, best first, and
// keeps at most SearchLimit.
func Rank(ideas []Idea, vectors []float64) []Hit {
	hits := vectorRank(ideas, vectors)
	if len(hits) > SearchLimit {
		return hits[:SearchLimit]
	}
	return hits
}

func vectorRank(ideas []Idea, vectors []float64) []Hit {
	hits := make([]Hit, 0)
	for _, idea := range ideas {
		if idea.Model != EmbedModel || len(idea.Embedding) != len(vectors) || len(vectors) == 0 {
			continue
		}
		hits = append(hits, Hit{Idea: idea, Score: Cosine(vectors, idea.Embedding)})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return IDLess(hits[i].ID, hits[j].ID)
		}
		return hits[i].Score > hits[j].Score
	})
	return hits
}

// Contains keeps the ideas whose text holds query, ignoring case.
func Contains(ideas []Idea, query string) []Idea {
	needle := strings.ToLower(query)
	out := make([]Idea, 0)
	for _, idea := range ideas {
		if strings.Contains(strings.ToLower(idea.Text), needle) {
			out = append(out, idea)
		}
	}
	return out
}

// Unembedded counts the ideas without a current embedding.
func Unembedded(ideas []Idea) int {
	n := 0
	for _, idea := range ideas {
		if idea.Model != EmbedModel || len(idea.Embedding) == 0 {
			n++
		}
	}
	return n
}

// Cosine is the cosine similarity of a and b, or 0 when either is zero.
func Cosine(a, b []float64) float64 {
	var dot, left, right float64
	for i := range a {
		dot += a[i] * b[i]
		left += a[i] * a[i]
		right += b[i] * b[i]
	}
	if left == 0 || right == 0 {
		return 0
	}
	return dot / (math.Sqrt(left) * math.Sqrt(right))
}
