package rag

import (
	"context"
	"fmt"
	"strings"

	"swu-cyber-security-agent-go/internal/vector"
)

type Retriever struct {
	VectorClient *vector.Client
	Embedder     *EmbeddingClient
	Collection   string
}

func NewRetriever(vc *vector.Client, ec *EmbeddingClient, collection string) *Retriever {
	return &Retriever{
		VectorClient: vc,
		Embedder:     ec,
		Collection:   collection,
	}
}

type SearchResult struct {
	Content  string
	Filename string
	Score    float32
}

func (r *Retriever) Search(ctx context.Context, query string, limit uint64) ([]SearchResult, error) {
	emb, err := r.Embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	points, err := r.VectorClient.Search(ctx, r.Collection, emb, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]SearchResult, 0, len(points))
	for _, p := range points {
		content := ""
		filename := ""

		if p.Payload != nil {
			if v, ok := p.Payload["text"]; ok {
				content = v.GetStringValue()
			}
			if v, ok := p.Payload["filename"]; ok {
				filename = v.GetStringValue()
			}
		}

		results = append(results, SearchResult{
			Content:  content,
			Filename: filename,
			Score:    p.Score,
		})
	}

	return results, nil
}

func (r *Retriever) RetrieveContext(ctx context.Context, query string) (string, error) {
	results, err := r.Search(ctx, query, 3) // Default top 3
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, res := range results {
		sb.WriteString(fmt.Sprintf("-- Document %d (%s) --\n%s\n\n", i+1, res.Filename, res.Content))
	}

	if sb.Len() == 0 {
		return "No relevant context found.", nil
	}

	return sb.String(), nil
}
