package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"swu-cyber-security-agent-go/internal/agent"
	"swu-cyber-security-agent-go/internal/model"
	"swu-cyber-security-agent-go/internal/rag"
	"swu-cyber-security-agent-go/internal/vector"
	"swu-cyber-security-agent-go/pkg/config"

	"github.com/joho/godotenv"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	ingestDir := flag.String("ingest", "", "Directory containing PDFs to ingest")
	chatQuery := flag.String("chat", "", "Query to ask the RAG agents")
	serverMode := flag.Bool("server", false, "Run as HTTP API/WEB Server")
	configPath := flag.String("config", "config.yaml", "Path to config.yaml")
	flag.Parse()

	// Load Config
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

	// 1. Init Vector Client
	vc, err := vector.NewClient(qdrantAddr)
	if err != nil {
		log.Fatalf("Failed to init Qdrant client: %v", err)
	}
	defer vc.Close()

	// 2. Init Embedding Client
	ec := rag.NewEmbeddingClient(ollamaBase, "all-minilm")

	if *ingestDir != "" {
		collectionName := "cve_database"
		if len(cfg.Agents) > 0 && len(cfg.Agents[0].KnowledgeBases) > 0 {
			collectionName = cfg.Agents[0].KnowledgeBases[0]
		}

		fmt.Printf("Ingesting into collection: %s\n", collectionName)

		ingestor := rag.NewIngestor(vc, ec, collectionName)
		if err := ingestor.IngestPDFs(ctx, *ingestDir); err != nil {
			log.Fatalf("Ingestion failed: %v", err)
		}
		fmt.Println("Ingestion complete.")
		return
	}

	// Determine Model Provider
	var llm adkmodel.LLM
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "openai" {
		baseURL := os.Getenv("OPENAI_BASE_URL")
		apiKey := os.Getenv("OPENAI_API_KEY")
		modelName := os.Getenv("OPENAI_MODEL_NAME")

		fmt.Printf("Using OpenAI Compatible Model: %s at %s\n", modelName, baseURL)
		llm = model.NewOpenAIModel(baseURL, apiKey, modelName)
	} else {
		fmt.Println("Using Mock Model (Set LLM_PROVIDER=openai in .env to use real model)")
		llm = &model.MockModel{FixedResponse: "This is a mock response from the Research Agent."}
	}

	if *chatQuery != "" {
		if len(cfg.Agents) == 0 {
			log.Fatal("No agents defined in config")
		}
		targetAgent := cfg.Agents[0]
		runCLI(ctx, *chatQuery, vc, ec, targetAgent, llm)
		return
	}

	if *serverMode {
		runServer(ctx, vc, ec, cfg, llm)
		return
	}

	flag.Usage()
}

func runCLI(ctx context.Context, query string, vc *vector.Client, ec *rag.EmbeddingClient, cfg config.AgentConfig, m adkmodel.LLM) {
	kbName := "cve_database"
	if len(cfg.KnowledgeBases) > 0 {
		kbName = cfg.KnowledgeBases[0]
	}
	retriever := rag.NewRetriever(vc, ec, kbName)

	fmt.Printf("Initializing Agent: %s\nInstruction: %s\n", cfg.Name, cfg.Instruction)

	researcher, err := agent.NewResearchAgent(
		ctx,
		cfg.Name,
		cfg.Description,
		kbName,
		retriever,
		m,
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	fmt.Printf("Agent %s is thinking...\n", researcher.Name)

	// Add instruction to the query since our adapter is simple
	fullPrompt := fmt.Sprintf("System Instruction: %s\n\nUser Query: %s", cfg.Instruction, query)

	req := &adkmodel.LLMRequest{
		Contents: []*genai.Content{
			{
				Parts: []*genai.Part{
					{Text: fullPrompt},
				},
			},
		},
	}

	for resp, err := range m.GenerateContent(ctx, req, false) {
		if err != nil {
			log.Printf("Error: %v", err)
		}
		if resp.Content != nil {
			for _, p := range resp.Content.Parts {
				fmt.Print(p.Text)
			}
		}
	}
	fmt.Println()
}

func runServer(ctx context.Context, vc *vector.Client, ec *rag.EmbeddingClient, cfg *config.Config, m adkmodel.LLM) {
	http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query   string `json:"query"`
			AgentID string `json:"agent_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var targetAgent config.AgentConfig
		if req.AgentID != "" {
			found := false
			for _, a := range cfg.Agents {
				if a.ID == req.AgentID {
					targetAgent = a
					found = true
					break
				}
			}
			if !found {
				http.Error(w, "Agent ID not found", http.StatusNotFound)
				return
			}
		} else {
			if len(cfg.Agents) > 0 {
				targetAgent = cfg.Agents[0]
			}
		}

		kbName := "cve_database"
		if len(targetAgent.KnowledgeBases) > 0 {
			kbName = targetAgent.KnowledgeBases[0]
		}
		retriever := rag.NewRetriever(vc, ec, kbName)

		_, err := agent.NewResearchAgent(
			ctx,
			targetAgent.Name,
			targetAgent.Description,
			kbName,
			retriever,
			m, // Pass the shared model instance (or create new if needed per request context?)
			// Ideally NewResearchAgent uses the model instance.
			// NOTE: sharing model instance is fine if it's stateless per request.
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Agent init failed: %v", err), http.StatusInternalServerError)
			return
		}

		fullPrompt := fmt.Sprintf("System Instruction: %s\n\nUser Query: %s", targetAgent.Instruction, req.Query)

		llmReq := &adkmodel.LLMRequest{
			Contents: []*genai.Content{{Parts: []*genai.Part{{Text: fullPrompt}}}},
		}

		var fullResponse string
		for resp, err := range m.GenerateContent(ctx, llmReq, false) {
			if err != nil {
				continue
			}
			if resp.Content != nil {
				for _, p := range resp.Content.Parts {
					fullResponse += p.Text
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]string{
			"agent":    targetAgent.Name,
			"response": fullResponse,
		})
	})

	// Ingest endpoint remains same...
	http.HandleFunc("/api/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Dir            string `json:"dir"`
			CollectionName string `json:"collection_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		targetCollection := req.CollectionName
		if targetCollection == "" {
			targetCollection = "cve_database"
		}

		ingestor := rag.NewIngestor(vc, ec, targetCollection)
		if err := ingestor.IngestPDFs(ctx, req.Dir); err != nil {
			http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Ingestion complete into %s", targetCollection),
		})
	})

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
