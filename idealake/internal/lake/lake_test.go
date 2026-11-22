package lake

import (
	"testing"
	"time"
)

func TestCosine(t *testing.T) {
	if Cosine([]float64{1, 0}, []float64{1, 0}) != 1 {
		t.Fatal("same")
	}
	if Cosine([]float64{1, 0}, []float64{0, 1}) != 0 {
		t.Fatal("orthogonal")
	}
	if Cosine([]float64{0, 0}, []float64{1, 0}) != 0 {
		t.Fatal("zero")
	}
}

func TestAssignIDsSkipsEveryIssuedID(t *testing.T) {
	existing := []Idea{{ID: "3"}, {ID: "x"}}
	ideas := AssignIDs(7, existing, []string{"a", "b"}, "t", [][]float64{{1}, {2}}, time.Unix(0, 0))
	if len(ideas) != 2 || ideas[0].ID != "8" || ideas[1].ID != "9" || ideas[0].Model != EmbedModel || ideas[1].Created != "1970-01-01T00:00:00Z" {
		t.Fatalf("ideas %+v", ideas)
	}
	if ideas := AssignIDs(0, existing, []string{"a"}, "t", nil, time.Unix(0, 0)); ideas[0].ID != "4" || ideas[0].Embedding != nil {
		t.Fatalf("ideas %+v", ideas)
	}
}

func TestRankAndRules(t *testing.T) {
	ideas := []Idea{
		{ID: "1", Topic: "t", Text: "Quiet", Model: EmbedModel, Embedding: []float64{1, 0}},
		{ID: "2", Topic: "t", Text: "loud", Model: EmbedModel, Embedding: []float64{0, 1}},
		{ID: "3", Topic: "u", Text: "quiet too"},
	}
	hits := Rank(FilterTopic(ideas, "t"), []float64{1, 0})
	if len(hits) != 2 || hits[0].ID != "1" || Unembedded(ideas) != 1 || len(Contains(ideas, "quiet")) != 2 {
		t.Fatalf("hits %+v", hits)
	}
	if _, err := CleanTopic("a\nb"); err == nil {
		t.Fatal("two-line topic")
	}
	counts := MergeTopics(TopicCounts(ideas), []string{"v"})
	if len(counts) != 3 || counts[2].Topic != "v" || counts[2].N != 0 {
		t.Fatalf("counts %+v", counts)
	}
}
