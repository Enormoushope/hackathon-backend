package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	userId := flag.String("user", "", "User ID to set as admin (e.g., user_001)")
	isAdmin := flag.Bool("admin", true, "Set to admin (true/false)")
	flag.Parse()

	if *userId == "" {
		fmt.Println("Usage: go run main.go -user=user_001 [-admin=true]")
		return
	}

	// Open database
	db, err := sql.Open("sqlite", "./hackathon.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Check if user exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", *userId).Scan(&count)
	if err != nil {
		log.Fatal("Query failed:", err)
	}

	if count == 0 {
		log.Fatalf("User %s not found", *userId)
	}

	// Update user admin status
	adminVal := 0
	if *isAdmin {
		adminVal = 1
	}

	result, err := db.Exec("UPDATE users SET is_admin = ? WHERE id = ?", adminVal, *userId)
	if err != nil {
		log.Fatal("Update failed:", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal("Failed to get rows affected:", err)
	}

	if rowsAffected > 0 {
		status := "admin"
		if !*isAdmin {
			status = "regular user"
		}
		fmt.Printf("✓ User %s updated to %s\n", *userId, status)
	} else {
		fmt.Println("No rows updated")
	}
}
