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
		`CREATE TABLE IF NOT EXISTS epss (
			cve TEXT PRIMARY KEY,
			epss_score REAL,
			percentile REAL,
			score_date TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS cisa_kev (
			cve_id TEXT PRIMARY KEY,
			vendor_project TEXT,
			date_added TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS nvd_meta (
			cve_id TEXT PRIMARY KEY,
			cvss_score REAL,
			published_date TEXT,
			last_modified TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS otx_ioc (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			indicator TEXT,
			type TEXT,
			pulse_id TEXT,
			created TEXT
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

// Cyber Intelligence Operations

func SaveEpss(cve string, score, percentile float64, date string) error {
	query := `INSERT INTO epss (cve, epss_score, percentile, score_date) VALUES (?, ?, ?, ?)
			  ON CONFLICT(cve) DO UPDATE SET epss_score=excluded.epss_score, percentile=excluded.percentile, score_date=excluded.score_date;`
	_, err := DB.Exec(query, cve, score, percentile, date)
	return err
}

func SaveCisaKev(cveID, vendor, date string) error {
	query := `INSERT INTO cisa_kev (cve_id, vendor_project, date_added) VALUES (?, ?, ?)
			  ON CONFLICT(cve_id) DO UPDATE SET vendor_project=excluded.vendor_project, date_added=excluded.date_added;`
	_, err := DB.Exec(query, cveID, vendor, date)
	return err
}

func SaveNvdMeta(cveID string, cvss float64, pubDate, modDate string) error {
	query := `INSERT INTO nvd_meta (cve_id, cvss_score, published_date, last_modified) VALUES (?, ?, ?, ?)
			  ON CONFLICT(cve_id) DO UPDATE SET cvss_score=excluded.cvss_score, published_date=excluded.published_date, last_modified=excluded.last_modified;`
	_, err := DB.Exec(query, cveID, cvss, pubDate, modDate)
	return err
}

func SaveOtxIoc(indicator, iocType, pulseID, created string) error {
	query := "INSERT INTO otx_ioc (indicator, type, pulse_id, created) VALUES (?, ?, ?, ?)"
	_, err := DB.Exec(query, indicator, iocType, pulseID, created)
	return err
}

// Query Helpers for Tools
func GetCveContext(cveID string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	// EPSS
	var epssScore, percentile float64
	err := DB.QueryRow("SELECT epss_score, percentile FROM epss WHERE cve = ?", cveID).Scan(&epssScore, &percentile)
	if err == nil {
		res["epss_score"] = epssScore
		res["epss_percentile"] = percentile
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	// CISA KEV
	var kevID string
	err = DB.QueryRow("SELECT cve_id FROM cisa_kev WHERE cve_id = ?", cveID).Scan(&kevID)
	res["is_cisa_kev"] = (err == nil)

	// NVD Meta
	var cvssScore float64
	err = DB.QueryRow("SELECT cvss_score FROM nvd_meta WHERE cve_id = ?", cveID).Scan(&cvssScore)
	if err == nil {
		res["cvss_score"] = cvssScore
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	return res, nil
}

func SearchIoc(indicator string) ([]map[string]interface{}, error) {
	query := "SELECT type, pulse_id, created FROM otx_ioc WHERE indicator = ?"
	rows, err := DB.Query(query, indicator)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var iocType, pulseID, created string
		if err := rows.Scan(&iocType, &pulseID, &created); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"type":     iocType,
			"pulse_id": pulseID,
			"created":  created,
		})
	}
	return results, nil
}
