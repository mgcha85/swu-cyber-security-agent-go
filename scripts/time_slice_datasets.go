package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// MITRE ATT&CK Simplified Structure
type StixObject struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Created  string      `json:"created"`
	Modified string      `json:"modified"`
	Data     interface{} `json:"-"` // Raw data catch-all
}

type StixBundle struct {
	Objects []json.RawMessage `json:"objects"`
}

// CISA KEV Structure
type KevVulnerability struct {
	CveID     string `json:"cveID"`
	DateAdded string `json:"dateAdded"`
}

type KevCatalog struct {
	Vulnerabilities []json.RawMessage `json:"vulnerabilities"`
}

func main() {
	cutoffDate, _ := time.Parse("2006-01-02", "2023-12-31")
	outputDir := "data/datasets/2023_slice"
	os.MkdirAll(outputDir, 0755)

	fmt.Println("--- Starting Time-Slicing for Backtesting (Cutoff: 2023-12-31) ---")

	// 1. Filter MITRE ATT&CK
	processMitre(cutoffDate, outputDir)

	// 2. Filter CISA KEV
	processCisaKev(cutoffDate, outputDir)

	// 3. Filter EPSS (by CVE year)
	processEpss("2023", outputDir)

	fmt.Println("\n--- Time-slicing complete. Files saved in " + outputDir + " ---")
}

func processEpss(cutoffYear string, outDir string) {
	fmt.Print("Processing EPSS Scores... ")
	inputPath := "data/datasets/epss_scores.csv"
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	lines := strings.Split(string(data), "\n")
	var filteredLines []string
	if len(lines) > 0 {
		filteredLines = append(filteredLines, lines[0]) // Keep header (metadata)
	}
	if len(lines) > 1 {
		filteredLines = append(filteredLines, lines[1]) // Keep CSV header (cve,epss,percentile)
	}

	count := 0
	for i := 2; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 1 {
			continue
		}
		cve := parts[0] // e.g., CVE-2023-1234
		cveParts := strings.Split(cve, "-")
		if len(cveParts) >= 2 {
			year := cveParts[1]
			if year <= cutoffYear {
				filteredLines = append(filteredLines, line)
				count++
			}
		}
	}

	outPath := outDir + "/epss_scores-2023.csv"
	os.WriteFile(outPath, []byte(strings.Join(filteredLines, "\n")), 0644)
	fmt.Printf("Done. (Kept %d CVEs)\n", count)
}

func processMitre(cutoff time.Time, outDir string) {
	fmt.Print("Processing MITRE ATT&CK... ")
	inputPath := "data/datasets/enterprise-attack.json"
	data, _ := os.ReadFile(inputPath)

	var bundle StixBundle
	json.Unmarshal(data, &bundle)

	var filteredObjects []json.RawMessage
	for _, raw := range bundle.Objects {
		var obj struct {
			Created  string `json:"created"`
			Modified string `json:"modified"`
		}
		json.Unmarshal(raw, &obj)

		// Parse dates (STIX dates are ISO8601)
		created, _ := time.Parse(time.RFC3339Nano, obj.Created)
		modified, _ := time.Parse(time.RFC3339Nano, obj.Modified)

		if created.Before(cutoff) || modified.Before(cutoff) {
			filteredObjects = append(filteredObjects, raw)
		}
	}

	outPath := outDir + "/enterprise-attack-2023.json"
	outBundle := map[string]interface{}{
		"type":    "bundle",
		"objects": filteredObjects,
	}
	outData, _ := json.MarshalIndent(outBundle, "", "  ")
	os.WriteFile(outPath, outData, 0644)
	fmt.Printf("Done. (Kept %d/%d objects)\n", len(filteredObjects), len(bundle.Objects))
}

func processCisaKev(cutoff time.Time, outDir string) {
	fmt.Print("Processing CISA KEV... ")
	inputPath := "data/datasets/cisa_kev.json"
	data, _ := os.ReadFile(inputPath)

	var catalog struct {
		Vulnerabilities []json.RawMessage `json:"vulnerabilities"`
	}
	json.Unmarshal(data, &catalog)

	var filteredVuls []json.RawMessage
	var groundTruth []json.RawMessage

	for _, raw := range catalog.Vulnerabilities {
		var v struct {
			DateAdded string `json:"dateAdded"`
		}
		json.Unmarshal(raw, &v)

		dateAdded, _ := time.Parse("2006-01-02", v.DateAdded)

		if dateAdded.Before(cutoff) || dateAdded.Equal(cutoff) {
			filteredVuls = append(filteredVuls, raw)
		} else {
			groundTruth = append(groundTruth, raw)
		}
	}

	// Knowledge Base (2023)
	os.WriteFile(outDir+"/cisa_kev-2023.json", marshalKev(filteredVuls), 0644)
	// Ground Truth (2024-2026) for validation
	os.WriteFile(outDir+"/cisa_kev-actual-2024-2026.json", marshalKev(groundTruth), 0644)

	fmt.Printf("Done. (Knowledge: %d, Ground Truth: %d)\n", len(filteredVuls), len(groundTruth))
}

func marshalKev(vuls []json.RawMessage) []byte {
	out := map[string]interface{}{
		"title":           "CISA KEV Sliced",
		"vulnerabilities": vuls,
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return data
}
