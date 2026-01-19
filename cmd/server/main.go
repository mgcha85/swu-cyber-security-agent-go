package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"swu-cyber-security-agent-go/internal/agent"
	"swu-cyber-security-agent-go/internal/model"
	"swu-cyber-security-agent-go/internal/rag"
	"swu-cyber-security-agent-go/internal/vector"
	"swu-cyber-security-agent-go/internal/vlm"
	"swu-cyber-security-agent-go/pkg/config"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	ingestDir := flag.String("ingest", "", "Directory containing PDFs to ingest (single collection)")
	batchIngestDir := flag.String("batch-ingest", "", "Root directory for batch ingestion")
	chatQuery := flag.String("chat", "", "Query to ask the RAG agents")
	serverMode := flag.Bool("server", false, "Run as HTTP API/WEB Server")
	configPath := flag.String("config", "config.yaml", "Path to config.yaml")
	agentID := flag.String("agent", "", "Specific Agent ID for chat")

	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	qdrantAddr := os.Getenv("QDRANT_ADDR")
	if qdrantAddr == "" {
		qdrantAddr = "localhost:6334"
	}
	ollamaBase := os.Getenv("OLLAMA_BASE")
	if ollamaBase == "" {
		ollamaBase = "http://localhost:11434"
	}

	ctx := context.Background()
	vc, err := vector.NewClient(qdrantAddr)
	if err != nil {
		log.Fatalf("Failed to init Qdrant client: %v", err)
	}
	defer vc.Close()

	ec := rag.NewEmbeddingClient(ollamaBase, cfg.System.EmbeddingModel)

	// Check ingest flags (batch or single)
	if *batchIngestDir != "" {
		handleBatchIngest(ctx, vc, ec, *batchIngestDir, cfg.System.EmbeddingDim)
		return
	}
	if *ingestDir != "" {
		handleSingleIngest(ctx, vc, ec, *ingestDir, cfg)
		return
	}

	// Init LLM (DeepSeek)
	var llm adkmodel.LLM
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "openai" {
		llm = model.NewOpenAIModel(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL_NAME"))
	} else {
		llm = &model.MockModel{FixedResponse: "Mock response."}
	}

	// Init VLM
	vlmClient := vlm.NewClient(ollamaBase, cfg.System.VisionModel)

	if *chatQuery != "" {
		// CLI Chat Logic
		handleCLIChat(ctx, vc, ec, cfg, llm, *chatQuery, *agentID)
		return
	}

	if *serverMode {
		runAPIServer(ctx, vc, ec, cfg, llm, vlmClient)
		return
	}

	flag.Usage()
}

func runAPIServer(ctx context.Context, vc *vector.Client, ec *rag.EmbeddingClient, cfg *config.Config, llm adkmodel.LLM, vlmClient *vlm.Client) {
	r := mux.NewRouter()

	// 1. Ingest
	r.HandleFunc("/api/ingest", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Dir            string `json:"dir"`
			CollectionName string `json:"collection_name"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.CollectionName == "" {
			req.CollectionName = "cve_database"
		}
		ingestor := rag.NewIngestor(vc, ec, req.CollectionName, cfg.System.EmbeddingDim)
		if err := ingestor.IngestPDFs(ctx, req.Dir); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}).Methods("POST")

	// 2. Individual Agent Chat
	r.HandleFunc("/api/agent/{id}/chat", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		agentID := vars["id"]
		var req struct {
			Query string `json:"query"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		targetAgent, found := findAgent(cfg, agentID)
		if !found {
			http.Error(w, "Agent not found", 404)
			return
		}

		// Start Agent
		kbName := "default"
		if len(targetAgent.KnowledgeBases) > 0 {
			kbName = targetAgent.KnowledgeBases[0]
		}
		retriever := rag.NewRetriever(vc, ec, kbName)
		// Manual RAG: Retrieve context first
		ctxData, _ := retriever.RetrieveContext(ctx, req.Query)

		fullPrompt := fmt.Sprintf("System: %s\n\nContext:\n%s\n\nUser Query: %s", targetAgent.Instruction, ctxData, req.Query)
		resp := runLLM(ctx, llm, fullPrompt)

		json.NewEncoder(w).Encode(map[string]string{"agent": targetAgent.Name, "response": resp})
	}).Methods("POST")

	// 3. VLM Analyze
	r.HandleFunc("/api/vlm/analyze", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Image  string `json:"image"`
			Prompt string `json:"prompt"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		resp, err := vlmClient.AnalyzeImage(ctx, req.Image, req.Prompt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"response": resp})
	}).Methods("POST")

	// 4. Super Agent Orchestration
	r.HandleFunc("/api/super/chat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
			Image string `json:"image"`
		} // Image is optional Base64
		json.NewDecoder(r.Body).Decode(&req)

		// A. Run Research Agents concurrently
		var wg sync.WaitGroup
		reports := make(map[string]string)
		var mu sync.Mutex

		for _, ag := range cfg.Agents {
			wg.Add(1)
			go func(a config.AgentConfig) {
				defer wg.Done()
				// Configure Retriever
				kbName := "default"
				if len(a.KnowledgeBases) > 0 {
					kbName = a.KnowledgeBases[0]
				}
				retriever := rag.NewRetriever(vc, ec, kbName)
				// Manual RAG
				ctxData, _ := retriever.RetrieveContext(ctx, req.Query)

				prompt := fmt.Sprintf("System: %s\nContext: %s\nQuery: %s", a.Instruction, ctxData, req.Query)
				resp := runLLM(ctx, llm, prompt)

				mu.Lock()
				reports[a.Name] = resp
				mu.Unlock()
			}(ag)
		}

		// B. Run VLM if image provided
		var vlmResult string
		if req.Image != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := vlmClient.AnalyzeImage(ctx, req.Image, "Analyze this figure in the context of cyber security: "+req.Query)
				if err == nil {
					mu.Lock()
					vlmResult = res
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		// C. Super Agent Synthesis
		sa := agent.NewSuperAgent(llm)
		finalReport, err := sa.Synthesize(ctx, req.Query, reports, vlmResult)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"final_report":  finalReport,
			"agent_reports": reports,
			"vlm_analysis":  vlmResult,
		})
	}).Methods("POST")

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Helpers
func runLLM(ctx context.Context, api adkmodel.LLM, prompt string) string {
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

func findAgent(cfg *config.Config, id string) (config.AgentConfig, bool) {
	for _, a := range cfg.Agents {
		if a.ID == id {
			return a, true
		}
	}
	return config.AgentConfig{}, false
}

func handleBatchIngest(ctx context.Context, vc *vector.Client, ec *rag.EmbeddingClient, root string, dim int) {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() {
			col := e.Name()
			path := filepath.Join(root, col)
			fmt.Printf("Ingesting %s...\n", col)
			rag.NewIngestor(vc, ec, col, dim).IngestPDFs(ctx, path)
		}
	}
}

func handleSingleIngest(ctx context.Context, vc *vector.Client, ec *rag.EmbeddingClient, dir string, cfg *config.Config) {
	// defaults
	rag.NewIngestor(vc, ec, "cve_database", cfg.System.EmbeddingDim).IngestPDFs(ctx, dir)
}

func handleCLIChat(ctx context.Context, vc *vector.Client, ec *rag.EmbeddingClient, cfg *config.Config, llm adkmodel.LLM, query, agentID string) {
	// ... Simplified CLI logic reusing findAgent and runLLM
	// (Omitted for brevity in this plan representation, but included in file)
	target, found := findAgent(cfg, agentID)
	if !found && len(cfg.Agents) > 0 {
		target = cfg.Agents[0]
	}

	kb := "default"
	if len(target.KnowledgeBases) > 0 {
		kb = target.KnowledgeBases[0]
	}
	retriever := rag.NewRetriever(vc, ec, kb)
	// Manual RAG
	ctxData, _ := retriever.RetrieveContext(ctx, query)

	fmt.Printf("Agent: %s\n", target.Name)
	resp := runLLM(ctx, llm, fmt.Sprintf("System: %s\nContext: %s\nQuery: %s", target.Instruction, ctxData, query))
	fmt.Println(resp)
}
