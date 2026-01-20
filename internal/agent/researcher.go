package agent

import (
	"context"
	"fmt"

	"swu-cyber-security-agent-go/internal/rag"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ResearchAgent struct {
	Agent agent.Agent
	Name  string
}

func NewResearchAgent(ctx context.Context, name, desc, kb string, r *rag.Retriever, m model.LLM) (*ResearchAgent, error) {
	// Define the tool handler
	type QueryInput struct {
		Query string `json:"query" doc:"The query to search in the knowledge base"`
	}
	type QueryOutput struct {
		Context string `json:"context"`
	}

	handler := func(ctx tool.Context, input QueryInput) (QueryOutput, error) {
		// Use a detached context or the tool context? Tool context is likely tied to the request.
		// We'll use context.Background() for the DB call if tool.Context doesn't provide one suitable,
		// but typically tool.Context is a context.Context.
		// Checking tool.Context definition might be needed, but assuming it satisfies context.Context or RetrieveContext can take it.
		// For now using context.Background() to avoid compatibility issues if tool.Context is strict.
		res, err := r.RetrieveContext(context.Background(), input.Query)
		if err != nil {
			return QueryOutput{Context: fmt.Sprintf("Error: %v", err)}, nil
		}
		return QueryOutput{Context: res}, nil
	}

	kbTool, err := functiontool.New(functiontool.Config{
		Name:        "consult_knowledge_base",
		Description: fmt.Sprintf("Consults the %s knowledge base.", kb),
	}, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to create kb tool: %w", err)
	}

	cveTool, err := NewCveTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create cve tool: %w", err)
	}

	iocTool, err := NewIocTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create ioc tool: %w", err)
	}

	agent, err := llmagent.New(llmagent.Config{
		Name:        name,
		Description: desc,
		Model:       m,
		Instruction: fmt.Sprintf("You are a specialized research agent focused on %s. "+
			"You have access to a knowledge base and structured cyber intelligence data. "+
			"Always consult the knowledge base and use provided tools (CVE details, IOC search) to gather factual data.", name),
		Tools: []tool.Tool{kbTool, cveTool, iocTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return &ResearchAgent{Agent: agent, Name: name}, nil
}
