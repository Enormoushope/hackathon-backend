package main

import (
	"database/sql"
	"fmt"
	"log"
)

// seedData populates the database with initial test data
func seedData(db *sql.DB) error {
	// Check if basic seed already exists
	var userCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return err
	}
	if userCount > 100 {
		return nil // Sufficient data exists
	}

	if err := seedUsers(db); err != nil {
		return err
	}
	if err := seedItems(db); err != nil {
		return err
	}
	if err := seedPriceHistory(db); err != nil {
		return err
	}
	if err := seedTransactions(db); err != nil {
		return err
	}
	if err := seedFollows(db); err != nil {
		return err
	}
	if err := seedReactions(db); err != nil {
		return err
	}

	log.Println("Seed data generation complete!")
	return nil
}

// seedDataLite populates the database with a reduced set of demo data for fast startup
func seedDataLite(db *sql.DB) error {
	// If we already have some users, skip
	var userCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return err
	}
	if userCount > 20 {
		return nil
	}

	if err := seedUsersLite(db, 50); err != nil {
		return err
	}
	if err := seedItemsLite(db, 200); err != nil {
		return err
	}
	if err := seedPriceHistoryLite(db, 10, 6); err != nil { // fewer series, months
		return err
	}
	if err := seedTransactionsLite(db, 80); err != nil {
		return err
	}
	if err := seedFollowsLite(db, 300); err != nil {
		return err
	}
	if err := seedReactionsLite(db, 600); err != nil {
		return err
	}
	log.Println("Lite seed data generation complete!")
	return nil
}

func seedUsersLite(db *sql.DB, n int) error {
	log.Printf("Seeding %d users (lite)...", n)
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("user_%04d", i)
		name := fmt.Sprintf("User%04d", i)
		avatar := fmt.Sprintf("https://i.pravatar.cc/150?u=%s", id)
		bio := "デモユーザーです。"
		if _, err := db.Exec(`INSERT INTO users (id, name, avatar_url, bio, rating, selling_count, follower_count, review_count) VALUES (?, ?, ?, ?, NULL, 0, 0, 0)`,
			id, name, avatar, bio); err != nil {
			return err
		}
	}
	return nil
}

func seedItemsLite(db *sql.DB, n int) error {
	log.Printf("Seeding %d items (lite)...", n)
	categories := []struct {
		label     string
		group     string
		code      string
		condition string
	}{
		{"トレーディングカード", "hobby-tcg", "010", "新品・未使用"},
		{"フィギュア", "hobby-figure", "150", "新品・未使用"},
		{"技術書", "books-tech", "120", "目立った傷や汚れなし"},
		{"ゲーム機", "ent-game", "030", "未使用に近い"},
	}
	conditions := []string{"新品・未使用", "未使用に近い", "目立った傷や汚れなし"}
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("item_%04d", i)
		cat := categories[i%len(categories)]
		title := fmt.Sprintf("%s デモ%04d", cat.label, i)
		condition := conditions[i%len(conditions)]
		description := fmt.Sprintf("%sのデモ商品。状態: %s。", cat.label, condition)
		price := 1000 + (i%20)*500
		image := "https://picsum.photos/seed/" + id + "/300/200"
		soldOut := 0
		sellerID := fmt.Sprintf("user_%04d", ((i % 50) + 1))
		investItem := 0
		if _, err := db.Exec(`INSERT INTO items (id, title, price, description, condition, category, image_url, is_sold_out, seller_id, is_invest_item, product_group) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, title, price, description, condition, cat.code, image, soldOut, sellerID, investItem, cat.group); err != nil {
			return err
		}
	}
	return nil
}

func seedPriceHistoryLite(db *sql.DB, series, months int) error {
	log.Printf("Seeding price history (lite) series=%d months=%d...", series, months)
	for i := 1; i <= series; i++ {
		itemID := fmt.Sprintf("item_%04d", i)
		for w := 1; w <= months; w++ {
			ts := fmt.Sprintf("2024-%02d-15 12:00:00", w)
			price := 5000 + i*200 + w*300
			id := fmt.Sprintf("ph_%s_%d", itemID, w)
			if _, err := db.Exec(`INSERT INTO price_history (id, item_id, price, recorded_at) VALUES (?, ?, ?, ?)`,
				id, itemID, price, ts); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedTransactionsLite(db *sql.DB, n int) error {
	log.Printf("Seeding %d transactions (lite)...", n)
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("tx_%04d", i)
		itemID := fmt.Sprintf("item_%04d", ((i*3)%200)+1)
		buyerID := fmt.Sprintf("user_%04d", ((i*5)%50)+1)
		sellerID := fmt.Sprintf("user_%04d", ((i*7)%50)+1)
		price := 3000 + (i%40)*200
		quantity := 1
		ttype := "purchase"
		created := fmt.Sprintf("2024-12-%02d 09:00:00", (i%28)+1)
		if _, err := db.Exec(`INSERT INTO transactions (id, item_id, buyer_id, seller_id, price, quantity, transaction_type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, itemID, buyerID, sellerID, price, quantity, ttype, created); err != nil {
			return err
		}
	}
	return nil
}

func seedFollowsLite(db *sql.DB, n int) error {
	log.Printf("Seeding %d follow relationships (lite)...", n)
	for i := 1; i <= n; i++ {
		follower := fmt.Sprintf("user_%04d", ((i*11)%50)+1)
		followee := fmt.Sprintf("user_%04d", ((i*13)%50)+1)
		if follower == followee {
			continue
		}
		id := fmt.Sprintf("uf_%04d", i)
		if _, err := db.Exec(`INSERT OR IGNORE INTO user_follows (id, follower_id, followee_id, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			id, follower, followee); err != nil {
			return err
		}
	}
	return nil
}

func seedReactionsLite(db *sql.DB, n int) error {
	log.Printf("Seeding %d item reactions (lite)...", n)
	reactTypes := []string{"like", "bookmark", "watch"}
	for i := 1; i <= n; i++ {
		userID := fmt.Sprintf("user_%04d", ((i*9)%50)+1)
		itemID := fmt.Sprintf("item_%04d", ((i*5)%200)+1)
		rtype := reactTypes[i%len(reactTypes)]
		id := fmt.Sprintf("ir_%04d", i)
		if _, err := db.Exec(`INSERT OR IGNORE INTO item_reactions (id, item_id, user_id, reaction_type, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			id, itemID, userID, rtype); err != nil {
			return err
		}
	}
	return nil
}

func seedUsers(db *sql.DB) error {
	log.Println("Seeding 500 users...")
	for i := 1; i <= 500; i++ {
		id := fmt.Sprintf("user_%04d", i)
		name := fmt.Sprintf("User%04d", i)
		avatar := fmt.Sprintf("https://i.pravatar.cc/150?u=%s", id)
		bio := "様々なカテゴリを出品しています。よろしくお願いします。"

		if _, err := db.Exec(`INSERT INTO users (id, name, avatar_url, bio, rating, selling_count, follower_count, review_count) VALUES (?, ?, ?, ?, NULL, 0, 0, 0)`,
			id, name, avatar, bio); err != nil {
			return err
		}

		if i%100 == 0 {
			log.Printf("  %d users seeded...", i)
		}
	}
	return nil
}

func seedItems(db *sql.DB) error {
	log.Println("Seeding 2000 items...")

	categories := []struct {
		label     string
		group     string
		code      string
		condition string
	}{
		{"トレーディングカード", "hobby-tcg", "010", "新品・未使用"},
		{"フィルムカメラ", "camera", "020", "未使用に近い"},
		{"技術書", "books-tech", "120", "目立った傷や汚れなし"},
		{"ゲーム機", "ent-game", "030", "新品・未使用"},
		{"ヴィンテージ衣類", "vintage", "040", "やや傷や汚れあり"},
		{"オーディオ", "audio", "050", "目立った傷や汚れなし"},
		{"PC/ノート", "pc", "060", "新品・未使用"},
		{"漫画コミック", "books-manga", "110", "目立った傷や汚れなし"},
		{"腕時計", "watch", "070", "新品・未使用"},
		{"スニーカー", "sneakers", "080", "未使用に近い"},
		{"アクセサリー", "jewelry", "090", "新品・未使用"},
		{"楽器", "musical-inst", "100", "やや傷や汚れあり"},
		{"美術品", "art", "130", "新品・未使用"},
		{"アンティーク家具", "furniture", "140", "傷や汚れあり"},
		{"フィギュア", "hobby-figure", "150", "新品・未使用"},
		{"レコード", "vinyl", "160", "目立った傷や汚れなし"},
	}

	conditions := []string{"新品・未使用", "未使用に近い", "目立った傷や汚れなし", "やや傷や汚れあり", "傷や汚れあり"}

	for i := 1; i <= 2000; i++ {
		id := fmt.Sprintf("item_%04d", i)
		cat := categories[i%len(categories)]
		title := fmt.Sprintf("%s サンプル%04d", cat.label, i)
		condition := conditions[i%len(conditions)]
		description := fmt.Sprintf("%sのデモ商品です。状態: %s。サンプル説明%04d。", cat.label, condition, i)
		price := 1000 + (i%90)*1000
		image := "https://picsum.photos/seed/" + id + "/300/200"
		sold := (i%17 == 0)
		sellerID := fmt.Sprintf("user_%04d", ((i % 500) + 1))
		invest := (cat.group == "hobby-tcg" && i%13 == 0)

		soldOut := 0
		if sold {
			soldOut = 1
		}
		investItem := 0
		if invest {
			investItem = 1
		}

		if _, err := db.Exec(`INSERT INTO items (id, title, price, description, condition, category, image_url, is_sold_out, seller_id, is_invest_item, product_group) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, title, price, description, condition, cat.code, image, soldOut, sellerID, investItem, cat.group); err != nil {
			return err
		}

		if i%500 == 0 {
			log.Printf("  %d items seeded...", i)
		}
	}
	return nil
}

func seedPriceHistory(db *sql.DB) error {
	log.Println("Seeding price history...")

	for i := 1; i <= 30; i++ {
		itemID := fmt.Sprintf("item_%04d", i)
		for w := 1; w <= 12; w++ {
			ts := fmt.Sprintf("2024-%02d-15 12:00:00", w)
			price := 10000 + i*500 + w*800
			id := fmt.Sprintf("ph_%s_%d", itemID, w)

			if _, err := db.Exec(`INSERT INTO price_history (id, item_id, price, recorded_at) VALUES (?, ?, ?, ?)`,
				id, itemID, price, ts); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedTransactions(db *sql.DB) error {
	log.Println("Seeding 500 transactions...")

	for i := 1; i <= 500; i++ {
		id := fmt.Sprintf("tx_%04d", i)
		itemID := fmt.Sprintf("item_%04d", ((i*3)%2000)+1)
		buyerID := fmt.Sprintf("user_%04d", ((i*5)%500)+1)
		sellerID := fmt.Sprintf("user_%04d", ((i*7)%500)+1)
		price := 5000 + (i%100)*900
		quantity := 1
		ttype := "purchase"
		created := fmt.Sprintf("2024-12-%02d 09:00:00", (i%28)+1)

		if _, err := db.Exec(`INSERT INTO transactions (id, item_id, buyer_id, seller_id, price, quantity, transaction_type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, itemID, buyerID, sellerID, price, quantity, ttype, created); err != nil {
			return err
		}

		if i%100 == 0 {
			log.Printf("  %d transactions seeded...", i)
		}
	}
	return nil
}

func seedFollows(db *sql.DB) error {
	log.Println("Seeding 1500 follow relationships...")

	for i := 1; i <= 1500; i++ {
		follower := fmt.Sprintf("user_%04d", ((i*11)%500)+1)
		followee := fmt.Sprintf("user_%04d", ((i*13)%500)+1)
		if follower == followee {
			continue
		}

		id := fmt.Sprintf("uf_%04d", i)
		if _, err := db.Exec(`INSERT OR IGNORE INTO user_follows (id, follower_id, followee_id, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			id, follower, followee); err != nil {
			return err
		}

		if i%300 == 0 {
			log.Printf("  %d follows seeded...", i)
		}
	}
	return nil
}

func seedReactions(db *sql.DB) error {
	log.Println("Seeding 3000 item reactions...")

	reactTypes := []string{"like", "bookmark", "watch"}
	for i := 1; i <= 3000; i++ {
		userID := fmt.Sprintf("user_%04d", ((i*9)%500)+1)
		itemID := fmt.Sprintf("item_%04d", ((i*5)%2000)+1)
		rtype := reactTypes[i%len(reactTypes)]
		id := fmt.Sprintf("ir_%04d", i)

		if _, err := db.Exec(`INSERT OR IGNORE INTO item_reactions (id, item_id, user_id, reaction_type, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			id, itemID, userID, rtype); err != nil {
			return err
		}

		if i%600 == 0 {
			log.Printf("  %d reactions seeded...", i)
		}
	}
	return nil
}
