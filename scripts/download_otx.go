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
	apiKey := os.Getenv("OTX_API_KEY")
	if apiKey == "" {
		fmt.Println("OTX_API_KEY not found in .env")
		return
	}

	outDir := "data/datasets/otx"
	os.MkdirAll(outDir, 0755)

	// OTX API - General pulses
	baseURL := "https://otx.alienvault.com/api/v1/pulses/subscribed"

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", baseURL, nil)
	req.Header.Set("X-OTX-API-KEY", apiKey)

	fmt.Println("Fetching data from AlienVault OTX...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching OTX: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("OTX API Error: %s - %s\n", resp.Status, string(body))
		return
	}

	outFile := outDir + "/pulses_recent.json"
	f, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		fmt.Printf("Error saving OTX data: %v\n", err)
		return
	}

	fmt.Printf("Successfully downloaded OTX data to %s\n", outFile)
}
