package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"swu-cyber-security-agent-go/internal/db"
)

func main() {
	db.InitDB()
	path := "data/datasets/epss_scores.csv"
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	// Skip the first metadata line
	header1, err := reader.Read()
	if err != nil {
		log.Fatalf("header1 error: %v", err)
	}
	fmt.Printf("Header1: %v\n", header1)

	scoreDate := ""
	if len(header1) > 0 && strings.Contains(header1[0], "model_version") {
		parts := strings.Split(header1[0], "score_date:")
		if len(parts) > 1 {
			scoreDate = parts[1]
		}
	}
	fmt.Printf("ScoreDate: %s\n", scoreDate)

	header2, err := reader.Read()
	if err != nil {
		log.Fatalf("header2 error: %v", err)
	}
	fmt.Printf("Header2: %v\n", header2)

	count := 0
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

		if count < 5 {
			fmt.Printf("Ingesting %s: %f, %f\n", cve, score, percentile)
		}
		db.SaveEpss(cve, score, percentile, scoreDate)
		count++
	}
	fmt.Printf("Total ingested: %d\n", count)
}
