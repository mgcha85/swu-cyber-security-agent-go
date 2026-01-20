package main

import (
	"context"
	"fmt"
	"log"
	"swu-cyber-security-agent-go/internal/vector"
)

func main() {
	qdrantAddr := "localhost:6334"
	client, err := vector.NewClient(qdrantAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// All collections mentioned in config.yaml + standard ones
	collections := []string{
		"cyber_intel_full",
		"mitre_attack",
		"DBIR",
		"DFIR",
		"Mandiant-M-Trends",
		"default",
	}

	for _, coll := range collections {
		fmt.Printf("Deleting collection if exists: %s\n", coll)
		// We implement a check or just try deletes
		err := client.DeleteCollection(ctx, coll)
		if err != nil {
			fmt.Printf("  Skip/Error for %s: %v\n", coll, err)
		} else {
			fmt.Printf("  Deleted %s\n", coll)
		}
	}
	fmt.Println("Full purge complete. The Vector DB is now clean.")
}
