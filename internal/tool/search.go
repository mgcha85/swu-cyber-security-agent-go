package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

type GoogleSearch struct {
	ApiKey string
	Cx     string
}

func NewGoogleSearch() *GoogleSearch {
	return &GoogleSearch{
		ApiKey: os.Getenv("GOOGLE_API_KEY"),
		Cx:     os.Getenv("GOOGLE_CX"),
	}
}

func (g *GoogleSearch) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if g.ApiKey == "" || g.Cx == "" {
		// Fallback or Mock if keys are missing?
		// For now, return empty or error.
		return nil, fmt.Errorf("GOOGLE_API_KEY or GOOGLE_CX not set")
	}

	endpoint := "https://www.googleapis.com/customsearch/v1"
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("key", g.ApiKey)
	q.Set("cx", g.Cx)
	q.Set("q", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("google search api failed: %s", resp.Status)
	}

	var data struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, item := range data.Items {
		results = append(results, SearchResult{
			Title:   item.Title,
			Link:    item.Link,
			Snippet: item.Snippet,
		})
	}
	return results, nil
}

func (g *GoogleSearch) SearchAndFormat(ctx context.Context, query string) (string, error) {
	results, err := g.Search(ctx, query)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("Search Results:\n")
	for i, res := range results {
		if i >= 3 {
			break
		} // Limit to top 3
		sb.WriteString(fmt.Sprintf("- %s: %s\n  %s\n", res.Title, res.Link, res.Snippet))
	}
	return sb.String(), nil
}
