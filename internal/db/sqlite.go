package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() error {
	dbPath := "data/agent.db"
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return createTables()
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS configs (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT,
			threat_name TEXT,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}
	return nil
}

// Config Operations
func UpsertConfig(key, value string) error {
	query := `INSERT INTO configs (key, value) VALUES (?, ?) 
			  ON CONFLICT(key) DO UPDATE SET value = excluded.value;`
	_, err := DB.Exec(query, key, value)
	return err
}

func GetConfig(key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM configs WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func GetAllConfigs() (map[string]string, error) {
	rows, err := DB.Query("SELECT key, value FROM configs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		configs[key] = value
	}
	return configs, nil
}

// Report Operations
type Report struct {
	ID         int       `json:"id"`
	Filename   string    `json:"filename"`
	ThreatName string    `json:"threat_name"`
	Summary    string    `json:"summary"`
	CreatedAt  time.Time `json:"created_at"`
}

func SaveReport(filename, threatName, summary string) error {
	query := "INSERT INTO reports (filename, threat_name, summary) VALUES (?, ?, ?)"
	_, err := DB.Exec(query, filename, threatName, summary)
	return err
}

func GetReports(limit int) ([]Report, error) {
	query := "SELECT id, filename, threat_name, summary, created_at FROM reports ORDER BY created_at DESC LIMIT ?"
	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var r Report
		if err := rows.Scan(&r.ID, &r.Filename, &r.ThreatName, &r.Summary, &r.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	return reports, nil
}
