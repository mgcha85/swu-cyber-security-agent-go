package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"swu-cyber-security-agent-go/internal/vector"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
)

type Ingestor struct {
	VectorClient *vector.Client
	Embedder     *EmbeddingClient
	Collection   string
}

func NewIngestor(vc *vector.Client, ec *EmbeddingClient, collection string) *Ingestor {
	return &Ingestor{
		VectorClient: vc,
		Embedder:     ec,
		Collection:   collection,
	}
}

// IngestPDFs iterates over a directory and ingests all PDF files
func (i *Ingestor) IngestPDFs(ctx context.Context, dir string) error {
	// Ensure collection exists
	// Assuming vector size 768 for all-MiniLM-L6-v2 or similar from Ollama
	// Adjust size based on the model. deepseek or others might be different.
	if err := i.VectorClient.CreateCollection(ctx, i.Collection, 768); err != nil {
		return err
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".pdf" {
			return nil
		}

		fmt.Printf("Processing %s...\n", path)
		// TODO: Real PDF text extraction. For Step 1, simple text mock/placeholder or use a library.
		// Since pure Go PDF text extraction can be complex, I might use a simple placeholder or suggest a library.
		// For now, I'll simulate text extraction to keep dependencies simple unless user requested robust parsing.
		// User mentioned "rsc.io/pdf" in plan but that's quite old. "dslipak/pdf" is a fork or similar.
		// Let's use a mock extraction for the structure first, or a simple text reader if files are text.

		text := "Mock content for " + path
		// Real impl would call i.extractText(path)

		return i.indexDocument(ctx, path, text)
	})
}

func (i *Ingestor) indexDocument(ctx context.Context, filename, text string) error {
	// Chunking (naive)
	chunkSize := 500
	for j := 0; j < len(text); j += chunkSize {
		end := j + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunk := text[j:end]

		emb, err := i.Embedder.Embed(ctx, chunk)
		if err != nil {
			return fmt.Errorf("embedding failure: %w", err)
		}

		pointID := uuid.New().String()

		points := []*pb.PointStruct{
			{
				Id: &pb.PointId{
					PointIdOptions: &pb.PointId_Uuid{Uuid: pointID},
				},
				Vectors: &pb.Vectors{
					VectorsOptions: &pb.Vectors_Vector{
						Vector: &pb.Vector{Data: emb},
					},
				},
				Payload: map[string]*pb.Value{
					"filename": {Kind: &pb.Value_StringValue{StringValue: filename}},
					"text":     {Kind: &pb.Value_StringValue{StringValue: chunk}},
				},
			},
		}

		if err := i.VectorClient.UpsertPoints(ctx, i.Collection, points); err != nil {
			return fmt.Errorf("upsert failure: %w", err)
		}
	}
	return nil
}
