package lake

// Embedder turns texts into vectors.
type Embedder interface {
	Embed(inputs []string) ([][]float64, error)
}

// EmbedPassages embeds new ideas as passages. ok is false when any vector
// is missing.
func EmbedPassages(e Embedder, texts []string) ([][]float64, bool) {
	if len(texts) == 0 {
		return nil, false
	}
	inputs := make([]string, len(texts))
	for i, text := range texts {
		inputs[i] = PassagePrefix + text
	}
	vectors, err := e.Embed(inputs)
	if err != nil || !vectorsReady(vectors, len(texts)) {
		return nil, false
	}
	return vectors, true
}

// EmbedQuery embeds a search query.
func EmbedQuery(e Embedder, query string) ([]float64, bool) {
	vectors, err := e.Embed([]string{QueryPrefix + query})
	if err != nil || !vectorsReady(vectors, 1) {
		return nil, false
	}
	return vectors[0], true
}

func vectorsReady(vectors [][]float64, n int) bool {
	if len(vectors) != n {
		return false
	}
	for _, vector := range vectors {
		if len(vector) == 0 {
			return false
		}
	}
	return true
}
