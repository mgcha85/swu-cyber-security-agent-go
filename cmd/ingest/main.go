package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"swu-cyber-security-agent-go/internal/db"
	"swu-cyber-security-agent-go/internal/rag"
	"swu-cyber-security-agent-go/internal/vector"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
)

func main() {
	ctx := context.Background()

	// Config (Hardcoded for script simplicity or pull from env if needed)
	qdrantAddr := "localhost:6334"
	ollamaURL := "http://localhost:11434"
	model := "qwen3-embedding:4b"
	dim := 2560 // Matching config.yaml

	vc, err := vector.NewClient(qdrantAddr)
	if err != nil {
		log.Fatalf("Vector client error: %v", err)
	}
	defer vc.Close()

	ec := rag.NewEmbeddingClient(ollamaURL, model)

	// Initialize SQLite
	db.InitDB()

	// 1. EPSS (SQLite Only) - FAST
	fmt.Println("Ingesting EPSS Scores into SQLite...")
	if err := IngestEpss("data/datasets/epss_scores.csv"); err != nil {
		fmt.Printf("EPSS ingestion skip/fail: %v\n", err)
	}

	// 2. CISA KEV (VDB + RDB)
	fmt.Println("Ingesting CISA KEV...")
	kevIngestor := &MetadataIngestor{vc: vc, ec: ec, collection: "cyber_intel_full", dim: dim}
	if err := kevIngestor.IngestCisaKev(ctx, "data/datasets/cisa_kev.json"); err != nil {
		log.Fatalf("CISA KEV ingestion failed: %v", err)
	}

	// 3. NVD (NIST) (VDB + RDB)
	fmt.Println("Ingesting NVD CVEs...")
	nvdIngestor := &MetadataIngestor{vc: vc, ec: ec, collection: "cyber_intel_full", dim: dim}
	if err := nvdIngestor.IngestNvd(ctx, "data/datasets/nvd/cves_recent.json"); err != nil {
		fmt.Printf("NVD ingestion skip/fail: %v\n", err)
	}

	// 4. MITRE ATT&CK (VDB Only) - SLOW
	fmt.Println("Ingesting MITRE ATT&CK...")
	mitreIngestor := &MetadataIngestor{vc: vc, ec: ec, collection: "mitre_attack", dim: dim}
	if err := mitreIngestor.IngestMitre(ctx, "data/datasets/enterprise-attack.json"); err != nil {
		log.Fatalf("MITRE ingestion failed: %v", err)
	}

	// 4. AlienVault OTX
	fmt.Println("Ingesting OTX Pulses...")
	otxIngestor := &MetadataIngestor{vc: vc, ec: ec, collection: "cyber_intel_full", dim: dim}
	if err := otxIngestor.IngestOtx(ctx, "data/datasets/otx/pulses_recent.json"); err != nil {
		fmt.Printf("OTX ingestion skip/fail: %v\n", err)
	}

	// 5. EPSS (SQLite Only)
	fmt.Println("Ingesting EPSS Scores into SQLite...")
	if err := IngestEpss("data/datasets/epss_scores.csv"); err != nil {
		fmt.Printf("EPSS ingestion skip/fail: %v\n", err)
	}

	// 6. Existing PDF Reports (Multi-collection)
	fmt.Println("Ingesting existing PDF reports from 'doc/' into multiple collections...")
	docEntries, _ := os.ReadDir("doc")
	for _, de := range docEntries {
		if de.IsDir() {
			collName := de.Name()
			subDir := filepath.Join("doc", collName)
			fmt.Printf("Indexing %s into collections: %s, cyber_intel_full\n", subDir, collName)

			// Index into specific collection
			specificIngestor := rag.NewIngestor(vc, ec, collName, dim)
			specificIngestor.IngestPDFs(ctx, subDir)

			// Also index into global collection
			globalIngestor := rag.NewIngestor(vc, ec, "cyber_intel_full", dim)
			globalIngestor.IngestPDFs(ctx, subDir)
		}
	}

	fmt.Println("All datasets ingested successfully.")
}

type MetadataIngestor struct {
	vc         *vector.Client
	ec         *rag.EmbeddingClient
	collection string
	dim        int
}

func (mi *MetadataIngestor) IngestMitre(ctx context.Context, path string) error {
	mi.vc.CreateCollection(ctx, mi.collection, uint64(mi.dim))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var bundle struct {
		Objects []json.RawMessage `json:"objects"`
	}
	json.Unmarshal(data, &bundle)

	for _, raw := range bundle.Objects {
		var obj struct {
			Type        string `json:"type"`
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		json.Unmarshal(raw, &obj)

		if obj.Type == "attack-pattern" && obj.Name != "" {
			text := fmt.Sprintf("Technique: %s\nDescription: %s", obj.Name, obj.Description)
			mi.index(ctx, text, map[string]string{
				"type": "mitre_technique",
				"name": obj.Name,
			})
		}
	}
	return nil
}

func (mi *MetadataIngestor) IngestCisaKev(ctx context.Context, path string) error {
	mi.vc.CreateCollection(ctx, mi.collection, uint64(mi.dim))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var catalog struct {
		Vulnerabilities []struct {
			CveID             string `json:"cveID"`
			VendorProject     string `json:"vendorProject"`
			VulnerabilityName string `json:"vulnerabilityName"`
			ShortDescription  string `json:"shortDescription"`
			RequiredAction    string `json:"requiredAction"`
			DateAdded         string `json:"dateAdded"`
		} `json:"vulnerabilities"`
	}
	json.Unmarshal(data, &catalog)

	for _, v := range catalog.Vulnerabilities {
		// VDB
		text := fmt.Sprintf("CVE: %s\nName: %s\nDescription: %s\nAction: %s", v.CveID, v.VulnerabilityName, v.ShortDescription, v.RequiredAction)
		mi.index(ctx, text, map[string]string{
			"type": "cisa_kev",
			"cve":  v.CveID,
		})

		// RDB
		db.SaveCisaKev(v.CveID, v.VendorProject, v.DateAdded)
	}
	return nil
}

func (mi *MetadataIngestor) IngestNvd(ctx context.Context, path string) error {
	mi.vc.CreateCollection(ctx, mi.collection, uint64(mi.dim))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var nvdRes struct {
		Vulnerabilities []struct {
			Cve struct {
				ID           string `json:"id"`
				Published    string `json:"published"`
				LastModified string `json:"lastModified"`
				Metrics      struct {
					CvssMetricV31 []struct {
						CvssData struct {
							BaseScore float64 `json:"baseScore"`
						} `json:"cvssData"`
					} `json:"cvssMetricV31"`
				} `json:"metrics"`
				Descriptions []struct {
					Value string `json:"value"`
				} `json:"descriptions"`
			} `json:"cve"`
		} `json:"vulnerabilities"`
	}
	json.Unmarshal(data, &nvdRes)

	for _, v := range nvdRes.Vulnerabilities {
		// VDB
		desc := ""
		if len(v.Cve.Descriptions) > 0 {
			desc = v.Cve.Descriptions[0].Value
		}
		text := fmt.Sprintf("CVE: %s\nDescription: %s", v.Cve.ID, desc)
		mi.index(ctx, text, map[string]string{
			"type": "nvd_cve",
			"cve":  v.Cve.ID,
		})

		// RDB
		score := 0.0
		if len(v.Cve.Metrics.CvssMetricV31) > 0 {
			score = v.Cve.Metrics.CvssMetricV31[0].CvssData.BaseScore
		}
		db.SaveNvdMeta(v.Cve.ID, score, v.Cve.Published, v.Cve.LastModified)
	}
	return nil
}

func (mi *MetadataIngestor) IngestOtx(ctx context.Context, path string) error {
	mi.vc.CreateCollection(ctx, mi.collection, uint64(mi.dim))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var otxRes struct {
		Results []struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Created     string   `json:"created"`
			Tags        []string `json:"tags"`
			Indicators  []struct {
				Indicator string `json:"indicator"`
				Type      string `json:"type"`
				Created   string `json:"created"`
			} `json:"indicators"`
		} `json:"results"`
	}
	json.Unmarshal(data, &otxRes)

	for _, p := range otxRes.Results {
		// VDB
		text := fmt.Sprintf("OTX Pulse: %s\nDescription: %s\nTags: %s", p.Name, p.Description, strings.Join(p.Tags, ", "))
		mi.index(ctx, text, map[string]string{
			"type": "otx_pulse",
			"name": p.Name,
		})

		// RDB (Indicators)
		for _, i := range p.Indicators {
			db.SaveOtxIoc(i.Indicator, i.Type, p.ID, i.Created)
		}
	}
	return nil
}

func IngestEpss(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // Allow variable field counts for metadata
	// Skip the first metadata line
	header1, _ := reader.Read()
	scoreDate := ""
	for _, part := range header1 {
		if strings.Contains(part, "score_date:") {
			p := strings.Split(part, "score_date:")
			if len(p) > 1 {
				scoreDate = p[1]
			}
		}
	}

	// Skip the header line (cve,epss,percentile)
	reader.Read()

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if len(record) < 3 {
			continue
		}
		cve := record[0]
		score, _ := strconv.ParseFloat(record[1], 64)
		percentile, _ := strconv.ParseFloat(record[2], 64)

		db.SaveEpss(cve, score, percentile, scoreDate)
	}
	return nil
}

func (mi *MetadataIngestor) index(ctx context.Context, text string, metadata map[string]string) {
	emb, _ := mi.ec.Embed(ctx, text)
	payload := make(map[string]*pb.Value)
	for k, v := range metadata {
		payload[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: v}}
	}
	payload["text"] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: text}}

	points := []*pb.PointStruct{
		{
			Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: uuid.New().String()}},
			Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: emb}}},
			Payload: payload,
		},
	}
	mi.vc.UpsertPoints(ctx, mi.collection, points)
}
