package idealake

import (
	"math"
	"sort"
	"strings"
)

type hit struct {
	Idea
	Score float64
}

func rankIdeas(ideas []Idea, vectors []float64) []hit {
	hits := vectorRank(ideas, vectors)
	if len(hits) > searchLimit {
		return hits[:searchLimit]
	}
	return hits
}

func vectorRank(ideas []Idea, vectors []float64) []hit {
	hits := make([]hit, 0)
	for _, idea := range ideas {
		if idea.Model != embedModel || len(idea.Embedding) != len(vectors) || len(vectors) == 0 {
			continue
		}
		hits = append(hits, hit{Idea: idea, Score: cosine(vectors, idea.Embedding)})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return idLess(hits[i].ID, hits[j].ID)
		}
		return hits[i].Score > hits[j].Score
	})
	return hits
}

func containsIdeas(ideas []Idea, query string) []Idea {
	needle := strings.ToLower(query)
	out := make([]Idea, 0)
	for _, idea := range ideas {
		if strings.Contains(strings.ToLower(idea.Text), needle) {
			out = append(out, idea)
		}
	}
	return out
}

func unembedded(ideas []Idea) int {
	n := 0
	for _, idea := range ideas {
		if idea.Model != embedModel || len(idea.Embedding) == 0 {
			n++
		}
	}
	return n
}

func cosine(a, b []float64) float64 {
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
