// Package xai embeds idealake texts with the xAI embeddings API.
package xai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/idealake/internal/lake"
)

// URL is the xAI embeddings endpoint.
const URL = "https://api.x.ai/v1/embeddings"

// Client is a lake.Embedder that posts to URL.
type Client struct {
	URL string
}

type embedRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format"`
}

type embedItem struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type embedResponse struct {
	Data []embedItem `json:"data"`
}

// Embed asks xAI for one vector per input. XAI_API_KEY is the key; it never
// appears in an error.
func (c Client) Embed(inputs []string) ([][]float64, error) {
	key := strings.TrimSpace(os.Getenv("XAI_API_KEY"))
	if key == "" {
		return nil, errors.New("idealake: XAI_API_KEY is not set")
	}
	body, err := json.Marshal(embedRequest{
		Model:          lake.EmbedModel,
		Input:          inputs,
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, embedError(err, key)
	}
	req, err := http.NewRequest(http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, embedError(err, key)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, embedError(err, key)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, embedError(fmt.Errorf("idealake: embeddings returned %d", resp.StatusCode), key)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, embedError(err, key)
	}
	var parsed embedResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, embedError(errors.New("idealake: embeddings response"), key)
	}
	vectors, err := takeVectors(parsed.Data, len(inputs))
	if err != nil {
		return nil, embedError(err, key)
	}
	return vectors, nil
}

func takeVectors(items []embedItem, n int) ([][]float64, error) {
	out := make([][]float64, n)
	seen := make([]bool, n)
	for _, item := range items {
		if item.Index < 0 || item.Index >= n || seen[item.Index] || len(item.Embedding) == 0 {
			return nil, errors.New("idealake: embeddings response")
		}
		seen[item.Index] = true
		out[item.Index] = item.Embedding
	}
	for _, ok := range seen {
		if !ok {
			return nil, errors.New("idealake: embeddings response")
		}
	}
	return out, nil
}

func embedError(err error, key string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if key != "" {
		msg = strings.ReplaceAll(msg, key, "redacted")
	}
	return errors.New(msg)
}
