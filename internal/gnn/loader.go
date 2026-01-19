package gnn

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type GNNResult struct {
	TrendSlope     string // "Increasing", "Decreasing", "Stable"
	TrendDetails   string // e.g. "Year1: 0.1 -> Year3: 0.5"
	Solutions      []string
	ImageCheckPath string // Path to image for VLM
}

type Loader struct {
	BaseDir string
}

func NewLoader(baseDir string) *Loader {
	return &Loader{BaseDir: baseDir}
}

// GetThreatAnalysis loads Graph connections and Trend data for a specific threat
func (l *Loader) GetThreatAnalysis(threatName string) (*GNNResult, error) {
	// 1. Get Solutions from graph_new.csv
	solutions, err := l.getSolutions(threatName)
	if err != nil {
		// Log error but maybe proceed? No, graph is essential.
		// Try fuzzy matching? For now exact match.
		return nil, err
	}

	// 2. Get Trend from {threat_name}_gap.csv
	// Filename pattern: [Threat]_gap.csv
	// Example: Account_Hijacking_gap.csv
	csvPath := filepath.Join(l.BaseDir, threatName+"_gap.csv")
	slope, details, err := l.parseTrendCSV(csvPath)
	if err != nil {
		// If file missing, maybe return just solutions
		slope = "Unknown"
		details = "Trend data unavailable"
	}

	// 3. Image Path
	imagePath := filepath.Join(l.BaseDir, threatName+".png")
	// Verify existence?
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		imagePath = ""
	}

	return &GNNResult{
		TrendSlope:     slope,
		TrendDetails:   details,
		Solutions:      solutions,
		ImageCheckPath: imagePath,
	}, nil
}

func (l *Loader) getSolutions(threat string) ([]string, error) {
	f, err := os.Open(filepath.Join(l.BaseDir, "graph_new.csv"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	for _, row := range records {
		if len(row) > 0 && row[0] == threat {
			// Found threat. Solutions are in subsequent columns
			var sols []string
			for i := 1; i < len(row); i++ {
				if row[i] != "" && row[i] != "-" {
					sols = append(sols, row[i])
				}
			}
			return sols, nil
		}
	}
	return nil, fmt.Errorf("threat '%s' not found in graph definition", threat)
}

func (l *Loader) parseTrendCSV(path string) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return "", "", err
	}

	// Expected: Solution, Year1, Year2, Year3
	// e.g. Solution,2025,2026,2027
	//      Session_management,0.0002,0.0003,0.0004

	if len(records) < 2 {
		return "Insufficient Data", "", nil
	}

	header := records[0] // Years
	// Assume we want the average trend of all solutions listed in this CSV?
	// Or usually just one row per solution.
	// Let's aggregate.

	var totalSlope float64
	var count int

	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 3 {
			continue
		}

		// Parse first and last numeric value
		firstVal, _ := strconv.ParseFloat(row[1], 64)
		lastVal, _ := strconv.ParseFloat(row[len(row)-1], 64)

		totalSlope += (lastVal - firstVal)
		count++
	}

	if count == 0 {
		return "Unknown", "No numeric data", nil
	}

	avgSlope := totalSlope / float64(count)
	direction := "Stable"
	if avgSlope > 0.00001 {
		direction = "Increasing"
	} else if avgSlope < -0.00001 {
		direction = "Decreasing"
	}

	years := fmt.Sprintf("%s to %s", header[1], header[len(header)-1])
	return direction, fmt.Sprintf("Trend (%s): %s (Avg Delta: %.5f)", years, direction, avgSlope), nil
}
