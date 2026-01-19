package agent

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

type SuperAgent struct {
	Model model.LLM
}

func NewSuperAgent(m model.LLM) *SuperAgent {
	return &SuperAgent{
		Model: m,
	}
}

func (sa *SuperAgent) Synthesize(ctx context.Context, query string, agentReports map[string]string, gnnResult string) (string, error) {
	var reportsBuilder strings.Builder
	for agentID, report := range agentReports {
		reportsBuilder.WriteString(fmt.Sprintf("\n--- Report from %s ---\n%s\n", agentID, report))
	}

	prompt := fmt.Sprintf(`
User Query: %s

## GNN Model Prediction
%s

## Research Agent Reports
%s

Synthesize these findings into a final report.
`, query, gnnResult, reportsBuilder.String())

	// Create a simple text content request
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Parts: []*genai.Part{
					{Text: prompt},
				},
			},
		},
	}

	// Consume the iterator
	var fullResponse strings.Builder
	for resp, err := range sa.Model.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", err
		}
		if resp.Content != nil {
			for _, part := range resp.Content.Parts {
				fullResponse.WriteString(part.Text)
			}
		}
	}

	return fullResponse.String(), nil
}
