package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"swu-cyber-security-agent-go/internal/agent"
	"swu-cyber-security-agent-go/internal/db"
	"swu-cyber-security-agent-go/internal/gnn"
	"swu-cyber-security-agent-go/internal/logger"
	"swu-cyber-security-agent-go/internal/model"
	"swu-cyber-security-agent-go/internal/rag"
	"swu-cyber-security-agent-go/internal/vector"
	"swu-cyber-security-agent-go/pkg/config"

	"github.com/joho/godotenv"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

var (
	vlmCache   = make(map[string]string)
	vlmCacheMu sync.RWMutex
)

func main() {
	// 1. Initialization
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
		llm = &model.MockModel{FixedResponse: "Simulated analysis report content."}
	}

	gnnLoader := gnn.NewLoader("gnn_results")
	vlm := model.NewVLMClient(ollamaBase, cfg.System.VisionModel)
	sa := agent.NewSuperAgent(llm)

	// 2. Load all threats from GNN results
	entries, _ := os.ReadDir("gnn_results")
	var threats []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_gap.csv") {
			threats = append(threats, strings.TrimSuffix(e.Name(), "_gap.csv"))
		}
	}

	fmt.Printf("Starting batch analysis for %d threats...\n", len(threats))

	for _, threat := range threats {
		logger.Pipeline("Starting analysis for threat: %s", threat)
		ctx := context.Background()

		// A. GNN Context
		gnnData, _ := gnnLoader.GetThreatAnalysis(threat)
		if gnnData == nil {
			fmt.Printf("Skipping %s: No GNN data found\n", threat)
			continue
		}
		gnnContext := fmt.Sprintf("Trend: %s. Details: %s. Solutions: %v", gnnData.TrendSlope, gnnData.TrendDetails, gnnData.Solutions)

		// A-2. VLM Image Analysis (with Caching)
		vlmInterpretation := "No visual analysis available."
		if gnnData.ImageCheckPath != "" {
			vlmCacheMu.RLock()
			cached, ok := vlmCache[gnnData.ImageCheckPath]
			vlmCacheMu.RUnlock()

			if ok {
				logger.VLM("Using cached interpretation for: %s", gnnData.ImageCheckPath)
				vlmInterpretation = cached
			} else {
				logger.VLM("Analyzing GNN image: %s", gnnData.ImageCheckPath)
				interpretation, err := vlm.AnalyzeImage(ctx, "Analyze this GNN threat graph and describe the predicted trend and risk levels seen in the visualization. Focus on the slope and any anomalies.", gnnData.ImageCheckPath)
				if err != nil {
					logger.VLM("Analysis error: %v", err)
				} else {
					vlmInterpretation = interpretation
					logger.VLM("Interpretation: %s", vlmInterpretation)

					vlmCacheMu.Lock()
					vlmCache[gnnData.ImageCheckPath] = vlmInterpretation
					vlmCacheMu.Unlock()
				}
			}
		} else {
			logger.VLM("No graph image found for threat: %s", threat)
		}

		// B. Run Research Agents
		reports := make(map[string]string)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, ag := range cfg.Agents {
			wg.Add(1)
			go func(a config.AgentConfig) {
				defer wg.Done()
				kbName := "default"
				if len(a.KnowledgeBases) > 0 {
					kbName = a.KnowledgeBases[0]
				}
				retriever := rag.NewRetriever(vc, ec, kbName)
				ctxData, _ := retriever.RetrieveContext(ctx, threat)

				// B-2. Relational Intelligence (RDB Lookup)
				cveRegex := regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)
				foundCves := cveRegex.FindAllString(threat+" "+ctxData, -1)
				relationalContext := "No structured metadata found for detected vulnerabilities."
				if len(foundCves) > 0 {
					var rdbParts []string
					processed := make(map[string]bool)
					for _, cve := range foundCves {
						if processed[cve] {
							continue
						}
						processed[cve] = true
						meta, err := db.GetCveContext(cve)
						if err == nil && len(meta) > 0 {
							rdbParts = append(rdbParts, fmt.Sprintf("- %s: EPSS=%v, CVSS=%v, CISA_KEV=%v", cve, meta["epss_score"], meta["cvss_score"], meta["is_cisa_kev"]))
						}
					}
					if len(rdbParts) > 0 {
						relationalContext = strings.Join(rdbParts, "\n")
						logger.Agent(a.Name, "Enriched with RDB metadata for %d CVEs", len(rdbParts))
					}
				}

				prompt := fmt.Sprintf("System: %s\n\n[Relational Metadata (Factual Metrics)]\n%s\n\n[Visual Intelligence Context]\n%s\n\n[Technical/Knowledge Context]\n%s\n\nQuery: Analyze the threat: %s", a.Instruction, relationalContext, vlmInterpretation, ctxData, threat)

				logger.Agent(a.Name, "Researching threat with context length: %d", len(ctxData))
				logger.Agent(a.Name, "Prompt: %s", prompt)

				resp := runLLMSync(ctx, llm, prompt)

				logger.Agent(a.Name, "Response: %s", resp)

				mu.Lock()
				reports[a.Name] = resp
				mu.Unlock()
			}(ag)
		}
		wg.Wait()

		// C. Super Agent Synthesis (English)
		fmt.Printf("Synthesizing reports for %s (num_reports: %d)...\n", threat, len(reports))
		finalReport, err := sa.Synthesize(ctx, threat, reports, gnnContext)
		if err != nil {
			fmt.Printf("Synthesis error for %s: %v\n", threat, err)
		}
		fmt.Printf("Final report length: %d\n", len(finalReport))

		// D. Save English Report
		os.MkdirAll("reports", 0755)
		engPath := filepath.Join("reports", threat+".md")
		os.WriteFile(engPath, []byte(finalReport), 0644)
		db.SaveReport(threat+".md", threat, "Analysis with new KBs (English)")

		// E. Korean Translation
		fmt.Printf("Translating %s to Korean...\n", threat)
		koPrompt := fmt.Sprintf("Translate the following Cyber Security Threat Analysis Report into professional Korean. Keep original structure and technical terms (e.g. CVE, MITRE) as appropriate but translate descriptions and advice clearly.\n\n%s", finalReport)
		koReport := runLLMSync(ctx, llm, koPrompt)

		// F. Save Korean Report
		koPath := filepath.Join("reports", threat+"_ko.md")
		os.WriteFile(koPath, []byte(koReport), 0644)
		db.SaveReport(threat+"_ko.md", threat, "Analysis with new KBs (Korean)")

		logger.Pipeline("Completed analysis for threat: %s", threat)
	}

	fmt.Println("Batch analysis complete.")
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
