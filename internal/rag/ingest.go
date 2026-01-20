package rag

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"swu-cyber-security-agent-go/internal/vector"

	"github.com/dslipak/pdf"
	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
)

type Ingestor struct {
	VectorClient *vector.Client
	Embedder     *EmbeddingClient
	Collection   string
	EmbeddingDim int
}

func NewIngestor(vc *vector.Client, ec *EmbeddingClient, collection string, dim int) *Ingestor {
	return &Ingestor{
		VectorClient: vc,
		Embedder:     ec,
		Collection:   collection,
		EmbeddingDim: dim,
	}
}

func (i *Ingestor) IngestPDFs(ctx context.Context, dir string) error {
	// Ensure collection exists
	if err := i.VectorClient.CreateCollection(ctx, i.Collection, uint64(i.EmbeddingDim)); err != nil {
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
		text, err := i.extractText(path)
		if err != nil {
			fmt.Printf("Warning: Failed to extract text from %s: %v\n", path, err)
			return nil // Skip failed files but continue walking
		}

		return i.indexDocument(ctx, path, text)
	})
}

func (i *Ingestor) extractText(path string) (string, error) {
	r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	_, err = buf.ReadFrom(b)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
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
