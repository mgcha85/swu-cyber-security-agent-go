package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// STIX Object structure for MITRE ATT&CK
type StixObject struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	ExternalRefs []struct {
		SourceName string `json:"source_name"`
		ExternalID string `json:"external_id"`
		URL        string `json:"url"`
	} `json:"external_references,omitempty"`
}

type StixBundle struct {
	Objects []StixObject `json:"objects"`
}

func main() {
	filePath := "data/datasets/enterprise-attack.json"
	file, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var bundle StixBundle
	if err := json.Unmarshal(file, &bundle); err != nil {
		fmt.Printf("Error unmarshaling JSON: %v\n", err)
		return
	}

	fmt.Println("=== MITRE ATT&CK Techniques (Sample) ===")
	count := 0
	for _, obj := range bundle.Objects {
		// Only focus on Techniques (attack-pattern)
		if obj.Type == "attack-pattern" && !obj.Name == "" {
			var attackID string
			for _, ref := range obj.ExternalRefs {
				if ref.SourceName == "mitre-attack" {
					attackID = ref.ExternalID
					break
				}
			}

			if attackID != "" {
				fmt.Printf("[%s] %s\n", attackID, obj.Name)
				count++
			}
		}
		if count >= 10 { // Just show 10 as a sample
			break
		}
	}
	fmt.Printf("\n... Total objects in file: %d\n", len(bundle.Objects))
	fmt.Println("To ingest this into Qdrant, you would create embeddings from the Name + Description.")
}
