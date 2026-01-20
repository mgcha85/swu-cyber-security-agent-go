package main

import (
	"fmt"
	"log"
	"swu-cyber-security-agent-go/internal/db"
)

func main() {
	if err := db.InitDB(); err != nil {
		log.Fatalf("InitDB failed: %v", err)
	}

	tables := []string{"epss", "cisa_kev", "nvd_meta", "otx_ioc"}
	for _, table := range tables {
		var count int
		err := db.DB.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("Table %s: Error %v\n", table, err)
		} else {
			fmt.Printf("Table %s: %d records\n", table, count)
		}
	}

	// Sample CVE test
	cve := "CVE-2023-38831" // Common ransomware CVE
	ctx, err := db.GetCveContext(cve)
	if err != nil {
		fmt.Printf("GetCveContext error: %v\n", err)
	} else {
		fmt.Printf("Context for %s: %v\n", cve, ctx)
	}
}
