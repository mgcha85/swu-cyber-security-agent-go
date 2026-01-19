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

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	ingestDir := flag.String("ingest", "", "Directory containing PDFs to ingest")
	chatQuery := flag.String("chat", "", "Query to ask the RAG agents")
	serverMode := flag.Bool("server", false, "Run as HTTP API/WEB Server")
	flag.Parse()

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
		ingestor := rag.NewIngestor(vc, ec, collectionName)
		if err := ingestor.IngestPDFs(ctx, *ingestDir); err != nil {
			log.Fatalf("Ingestion failed: %v", err)
		}
		fmt.Println("Ingestion complete.")
		return
	}

	if *chatQuery != "" {
		runCLI(ctx, *chatQuery, vc, ec)
		return
	}

	if *serverMode {
		runServer(ctx, vc, ec)
		return
	}

	flag.Usage()
}

func runCLI(ctx context.Context, query string, vc *vector.Client, ec *rag.EmbeddingClient) {
	// Init Mock Model
	m := &model.MockModel{FixedResponse: "This is a mock response from the Research Agent."}

	retriever := rag.NewRetriever(vc, ec, "cve_database")

	researcher, err := agent.NewResearchAgent(
		ctx,
		"Attacker Feasibility",
		"Analyzes attack feasibility.",
		"cve_database",
		retriever,
		m,
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	fmt.Printf("Agent %s is thinking...\n", researcher.Name)

	req := &adkmodel.LLMRequest{
		Contents: []*genai.Content{
			{
				Parts: []*genai.Part{
					{Text: fmt.Sprintf("Question: %s", query)},
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

func runServer(ctx context.Context, vc *vector.Client, ec *rag.EmbeddingClient) {
	// Re-using dependencies for handlers
	http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Logic similar to CLI
		m := &model.MockModel{FixedResponse: "This is a mock API response from the RAG System."}
		retriever := rag.NewRetriever(vc, ec, "cve_database")
		_, err := agent.NewResearchAgent(
			ctx,
			"Attacker Feasibility",
			"Analyzes attack feasibility.",
			"cve_database",
			retriever,
			m,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Agent init failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Simulating response generation (using MockModel direct call for POC)
		llmReq := &adkmodel.LLMRequest{
			Contents: []*genai.Content{{Parts: []*genai.Part{{Text: req.Query}}}},
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

		json.NewEncoder(w).Encode(map[string]string{"response": fullResponse})
	})

	http.HandleFunc("/api/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Dir string `json:"dir"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ingestor := rag.NewIngestor(vc, ec, "cve_database")
		if err := ingestor.IngestPDFs(ctx, req.Dir); err != nil {
			http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Ingestion complete"})
	})

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
