package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/AlexTLDR/evite/internal/utils"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./evite.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get all invitations
	rows, err := db.Query("SELECT id, phone FROM invitations")
	if err != nil {
		log.Fatalf("Failed to query invitations: %v", err)
	}
	defer rows.Close()

	type invitation struct {
		id    int64
		phone string
	}

	var invitations []invitation
	for rows.Next() {
		var inv invitation
		if err := rows.Scan(&inv.id, &inv.phone); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		invitations = append(invitations, inv)
	}

	fmt.Printf("Found %d invitations to process\n", len(invitations))

	// Normalize each phone number
	updated := 0
	failed := 0
	for _, inv := range invitations {
		normalized, err := utils.NormalizePhoneNumber(inv.phone)
		if err != nil {
			log.Printf("Failed to normalize phone %q (ID: %d): %v", inv.phone, inv.id, err)
			failed++
			continue
		}

		// Only update if the phone number changed
		if normalized != inv.phone {
			_, err := db.Exec("UPDATE invitations SET phone = ? WHERE id = ?", normalized, inv.id)
			if err != nil {
				log.Printf("Failed to update phone for ID %d: %v", inv.id, err)
				failed++
				continue
			}
			fmt.Printf("Updated ID %d: %q -> %q\n", inv.id, inv.phone, normalized)
			updated++
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total: %d\n", len(invitations))
	fmt.Printf("  Updated: %d\n", updated)
	fmt.Printf("  Failed: %d\n", failed)
	fmt.Printf("  Unchanged: %d\n", len(invitations)-updated-failed)
}
