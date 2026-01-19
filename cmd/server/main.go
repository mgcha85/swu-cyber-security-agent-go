package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"swu-cyber-security-agent-go/internal/agent"
	"swu-cyber-security-agent-go/internal/db"
	"swu-cyber-security-agent-go/internal/gnn"
	"swu-cyber-security-agent-go/internal/model"
	"swu-cyber-security-agent-go/internal/rag"
	"swu-cyber-security-agent-go/internal/tool"
	"swu-cyber-security-agent-go/internal/vector"
	"swu-cyber-security-agent-go/internal/vlm"
	"swu-cyber-security-agent-go/pkg/config"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

var (
	llmRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "Time spent processing LLM requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"model"},
	)
)

func init() {
	prometheus.MustRegister(llmRequestDuration)
}

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

	// Init DB & Apply Overrides
	if err := db.InitDB(); err != nil {
		log.Printf("Failed to init DB, proceeding without persistence: %v", err)
	} else {
		overrides, err := db.GetAllConfigs()
		if err == nil {
			cfg.ApplyOverrides(overrides)
		}
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

	searchTool := tool.NewGoogleSearch()
	gnnLoader := gnn.NewLoader("gnn_results")

	// Metrics Endpoint
	// Metrics Endpoint
	r.Handle("/metrics", promhttp.Handler())

	// 0. Config & History Endpoints
	r.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			configs, _ := db.GetAllConfigs()
			// Mask secrets
			for k, v := range configs {
				if strings.Contains(k, "KEY") {
					if len(v) > 4 {
						configs[k] = "..." + v[len(v)-4:]
					} else {
						configs[k] = "***"
					}
				}
			}
			json.NewEncoder(w).Encode(configs)
		} else if r.Method == "POST" {
			var req struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if err := db.UpsertConfig(req.Key, req.Value); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			// Apply immediately
			cfg.ApplyOverrides(map[string]string{req.Key: req.Value})
			json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		}
	}).Methods("GET", "POST")

	r.HandleFunc("/api/reports", func(w http.ResponseWriter, r *http.Request) {
		reports, err := db.GetReports(20) // Limit 20
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(reports)
	}).Methods("GET")

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
			Image string `json:"image"` // User provided image (optional)
		}
		json.NewDecoder(r.Body).Decode(&req)

		// 0. GNN Data Loading
		// Try to identify threat from query. For simplicity, check if query contains known threat names
		// or if query IS the threat name.
		// Detailed matching could be LLM based, but let's try direct map first or fuzzy.
		// For now, let's assume the query might BE the threat name, or we iterate known files.
		// Or we pass it to the GNN Loader to find best match?
		// Let's rely on simple heuristic: Check if query matches a file prefix in gnn_results.
		// Actually, let's look for known keywords from query.
		threatName := "Account_Hijacking" // Default for testing if verification fails?
		// Better: Iterate known threats to find match in query
		knownThreats := []string{"Account_Hijacking", "Advanced_Persistent_Threat", "Adversarial_Attack"}
		for _, t := range knownThreats {
			if strings.Contains(strings.ToLower(req.Query), strings.ToLower(strings.ReplaceAll(t, "_", " "))) ||
				strings.Contains(strings.ToLower(req.Query), strings.ToLower(t)) {
				threatName = t
				break
			}
		}

		gnnData, err := gnnLoader.GetThreatAnalysis(threatName)
		gnnContext := "No GNN Data Found for this query."

		if err == nil {
			gnnContext = fmt.Sprintf("Trend: %s. Details: %s. Solutions: %v", gnnData.TrendSlope, gnnData.TrendDetails, gnnData.Solutions)
			if gnnData.ImageCheckPath != "" {
				// Image path is available for VLM to use later
			}
		}

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

				// Manual Search (Supplement)
				searchData, _ := searchTool.SearchAndFormat(ctx, req.Query)

				combinedContext := fmt.Sprintf("KB Context:\n%s\n\nGoogle Search Context:\n%s", ctxData, searchData)

				prompt := fmt.Sprintf("System: %s\nContext: %s\nQuery: %s", a.Instruction, combinedContext, req.Query)
				resp := runLLM(ctx, llm, prompt)

				mu.Lock()
				reports[a.Name] = resp
				mu.Unlock()
			}(ag)
		}

		// B. Run VLM
		// Priority: User Image > GNN Image > Nothing
		targetImage := req.Image
		// If req.Image is empty, and we have GNN Image (need to base64 encode), use it.
		// For now, let's just use a placeholder text if we can't encode, or I'll add the import.
		// I will create a helper for file->base64 in a moment, but since I can't easily add import inside function...
		// I'll skip effective base64 here and just pass the path if VLM supports it? No it supports base64.
		// I'll add `encoding/base64` to imports.

		var vlmResult string
		if targetImage != "" { // User provided
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := vlmClient.AnalyzeImage(ctx, targetImage, "Analyze this cyber security figure: "+req.Query)
				if err == nil {
					mu.Lock()
					vlmResult = res
					mu.Unlock()
				}
			}()
		} else if gnnData != nil && gnnData.ImageCheckPath != "" {
			// Automate VLM on GNN Image
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Read and Encode
				imgBytes, err := os.ReadFile(gnnData.ImageCheckPath)
				if err == nil {
					encoded := base64.StdEncoding.EncodeToString(imgBytes)
					res, err := vlmClient.AnalyzeImage(ctx, encoded, "Analyze this trend graph for "+threatName)
					if err == nil {
						mu.Lock()
						vlmResult = "GNN Image Analysis: " + res
						mu.Unlock()
					}
				}
			}()
		}

		wg.Wait()

		// C. Super Agent Synthesis
		sa := agent.NewSuperAgent(llm)
		// Pass GNN Data text to Synthesis
		finalReport, err := sa.Synthesize(ctx, req.Query, reports, fmt.Sprintf("%s\n\n(VLM on Graph: %s)", gnnContext, vlmResult))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Save Report to DB
		summary := finalReport
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		db.SaveReport(threatName+".md", threatName, summary)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"final_report":  finalReport,
			"agent_reports": reports,
			"vlm_analysis":  vlmResult,
			"gnn_data":      gnnContext,
		})
	}).Methods("POST")

	// 5. Metadata Endpoints
	// List Agents
	r.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		type AgentSummary struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		var summaries []AgentSummary
		for _, a := range cfg.Agents {
			summaries = append(summaries, AgentSummary{ID: a.ID, Name: a.Name})
		}
		json.NewEncoder(w).Encode(summaries)
	}).Methods("GET")

	// List Knowledge Bases (Collections)
	r.HandleFunc("/api/kb", func(w http.ResponseWriter, r *http.Request) {
		cols, err := vc.ListCollections(ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string][]string{"knowledge_bases": cols})
	}).Methods("GET")

	// List Agent to KB Mapping
	r.HandleFunc("/api/agents/map", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(cfg.Agents)
	}).Methods("GET")

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Helpers
func runLLM(ctx context.Context, api adkmodel.LLM, prompt string) string {
	start := time.Now()
	defer func() {
		// Verify if we can get model name easily, for now use "default" or from env
		modelName := os.Getenv("OPENAI_MODEL_NAME")
		if modelName == "" {
			modelName = "unknown"
		}
		llmRequestDuration.WithLabelValues(modelName).Observe(time.Since(start).Seconds())
	}()

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
