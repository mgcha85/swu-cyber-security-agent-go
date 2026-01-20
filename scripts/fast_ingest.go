package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"swu-cyber-security-agent-go/internal/rag"
	"swu-cyber-security-agent-go/internal/vector"
)

func main() {
	ctx := context.Background()
	qdrantAddr := "localhost:6334"
	ollamaURL := "http://localhost:11434"
	model := "qwen3-embedding:4b"
	dim := 2560

	vc, err := vector.NewClient(qdrantAddr)
	if err != nil {
		log.Fatalf("Vector client error: %v", err)
	}
	defer vc.Close()

	ec := rag.NewEmbeddingClient(ollamaURL, model)

	fmt.Println("Starting targeted PDF ingestion for verification...")

	docEntries, _ := os.ReadDir("doc")
	for _, de := range docEntries {
		if de.IsDir() {
			collName := de.Name()
			subDir := filepath.Join("doc", collName)
			fmt.Printf("Indexing %s into collections: %s, cyber_intel_full\n", subDir, collName)

			// Index into specific collection
			specificIngestor := rag.NewIngestor(vc, ec, collName, dim)
			if err := specificIngestor.IngestPDFs(ctx, subDir); err != nil {
				fmt.Printf("  Warning: failed to ingest PDFs into %s: %v\n", collName, err)
			}

			// Also index into global collection
			globalIngestor := rag.NewIngestor(vc, ec, "cyber_intel_full", dim)
			if err := globalIngestor.IngestPDFs(ctx, subDir); err != nil {
				fmt.Printf("  Warning: failed to ingest PDFs into cyber_intel_full from %s: %v\n", subDir, err)
			}
		}
	}

	fmt.Println("Fast ingestion complete.")
}
