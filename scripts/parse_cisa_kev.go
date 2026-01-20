package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type KevVulnerability struct {
	CveID                      string `json:"cveID"`
	VendorProject              string `json:"vendorProject"`
	Product                    string `json:"product"`
	VulnerabilityName          string `json:"vulnerabilityName"`
	ShortDescription           string `json:"shortDescription"`
	KnownRansomwareCampaignUse string `json:"knownRansomwareCampaignUse"`
}

type KevCatalog struct {
	Vulnerabilities []KevVulnerability `json:"vulnerabilities"`
}

func main() {
	filePath := "data/datasets/cisa_kev.json"
	file, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var catalog KevCatalog
	if err := json.Unmarshal(file, &catalog); err != nil {
		fmt.Printf("Error unmarshaling JSON: %v\n", err)
		return
	}

	fmt.Printf("=== CISA KEV Summary ===\n")
	fmt.Printf("Total vulnerabilities: %d\n\n", len(catalog.Vulnerabilities))

	fmt.Println("=== Ransomware-Related Vulnerabilities (Sample) ===")
	count := 0
	for _, v := range catalog.Vulnerabilities {
		if v.KnownRansomwareCampaignUse == "Known" {
			fmt.Printf("[%s] %s (%s)\n", v.CveID, v.VulnerabilityName, v.Product)
			count++
		}
		if count >= 10 {
			break
		}
	}

	fmt.Printf("\n... and more. Using this data, you can prioritize remediation for CVEs that are already being exploited.\n")
}
