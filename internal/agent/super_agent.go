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

## GNN Data (Quantitative & Trend)
%s

## Research Agent Findings (Qualitative)
%s

## Instructions for Synthesis
1. **Comparison Table**: Create a markdown table comparing GNN Predictions vs. Agent Opinions.
   | Feature/Threat | GNN Trend | Agent Consensus | Agreement |
   | :--- | :--- | :--- | :--- |
   | ... | ... | ... | ... |

2. **Agent Stance Summary**: Briefly summarize each agent's stance and their basis (Success/Fail/Neutral).
   - **Attacker Feasibility**: [Reason] -> (Agree/Disagree with GNN)
   - ...

3. **Final Conclusion (Vote)**:
   - Tally the "votes" (e.g., Agents supporting GNN vs. opposing).
   - Format: "Result: X vs Y - [Winner] Dominates".
   - providing a final strategic recommendation.

Synthesize these findings into the final report following the structure above.
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
