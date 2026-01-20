package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	apiKey := os.Getenv("NIS_API_KEY")
	if apiKey == "" {
		fmt.Println("NIS_API_KEY not found in .env")
		return
	}

	outDir := "data/datasets/nvd"
	os.MkdirAll(outDir, 0755)

	// NIST NVD API v2
	// We'll fetch the most recent vulnerabilities (last 30 days) as a start
	// For a full database, one would need to iterate with offsets.
	baseURL := "https://services.nvd.nist.gov/rest/json/cves/2.0"

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", baseURL, nil)
	req.Header.Set("apiKey", apiKey)

	fmt.Println("Fetching data from NVD API...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching NVD: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("NVD API Error: %s - %s\n", resp.Status, string(body))
		return
	}

	outFile := outDir + "/cves_recent.json"
	f, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		fmt.Printf("Error saving NVD data: %v\n", err)
		return
	}

	fmt.Printf("Successfully downloaded NVD data to %s\n", outFile)
}
