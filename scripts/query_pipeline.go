package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"swu-cyber-security-agent-go/internal/agent"
	"swu-cyber-security-agent-go/internal/db"
	"swu-cyber-security-agent-go/internal/logger"
	"swu-cyber-security-agent-go/internal/model"
	"swu-cyber-security-agent-go/internal/rag"
	"swu-cyber-security-agent-go/internal/vector"
	"swu-cyber-security-agent-go/pkg/config"

	"github.com/joho/godotenv"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	godotenv.Load()
	cfg, _ := config.LoadConfig("config.yaml")
	db.InitDB()

	qdrantAddr := "localhost:6334"
	ollamaBase := "http://localhost:11434"

	vc, _ := vector.NewClient(qdrantAddr)
	defer vc.Close()
	ec := rag.NewEmbeddingClient(ollamaBase, cfg.System.EmbeddingModel)

	var llm adkmodel.LLM
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "openai" {
		llm = model.NewOpenAIModel(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL_NAME"))
	} else {
		llm = &model.MockModel{FixedResponse: "Simulated research result."}
	}

	threat := "Ransomware"
	logger.Pipeline("Starting manual query for threat: %s", threat)
	ctx := context.Background()

	// 1. Run a single agent for demonstration
	agentCfg := cfg.Agents[0] // e.g., Attacker Feasibility
	retriever := rag.NewRetriever(vc, ec, "cyber_intel_full")

	ctxData, _ := retriever.RetrieveContext(ctx, threat)
	prompt := fmt.Sprintf("System: %s\nContext: %s\nQuery: Analyze the threat: %s", agentCfg.Instruction, ctxData, threat)

	logger.Agent(agentCfg.Name, "Prompt: %s", prompt)
	resp := runLLMSync(ctx, llm, prompt)
	logger.Agent(agentCfg.Name, "Response: %s", resp)

	// 2. Super Agent Synthesis
	sa := agent.NewSuperAgent(llm)
	reports := map[string]string{agentCfg.Name: resp}
	gnnResult := "Trend: Increasing. Details: Ransomware activities are spiked in 2025."

	finalReport, _ := sa.Synthesize(ctx, threat, reports, gnnResult)
	logger.Pipeline("Final Report Generated (first 100 chars): %s...", finalReport[:100])
}

func runLLMSync(ctx context.Context, api adkmodel.LLM, prompt string) string {
	llmReq := &adkmodel.LLMRequest{Contents: []*genai.Content{{Parts: []*genai.Part{{Text: prompt}}}}}
	var sb strings.Builder
	for resp, err := range api.GenerateContent(ctx, llmReq, false) {
		if err == nil && resp.Content != nil {
			for _, p := range resp.Content.Parts {
				sb.WriteString(p.Text)
			}
		}
	}
	return sb.String()
}
