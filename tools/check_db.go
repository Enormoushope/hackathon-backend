package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "hackathon.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check items count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total items in database: %d\n", count)

	// Check items details
	rows, err := db.Query("SELECT id, title, price, is_sold_out FROM items LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\nFirst 5 items:")
	fmt.Println("ID | Title | Price | IsSoldOut")
	for rows.Next() {
		var id, title string
		var price int
		var isSoldOut int
		err := rows.Scan(&id, &title, &price, &isSoldOut)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s | %s | %d | %d\n", id, title, price, isSoldOut)
	}

	// Check users count
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nTotal users in database: %d\n", userCount)
}
