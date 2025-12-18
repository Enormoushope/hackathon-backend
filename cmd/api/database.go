package main

import (
	"database/sql"
	"fmt"
)

// createTables creates all required database tables
func createTables(db *sql.DB) error {
	tables := []string{
		createUsersTable(),
		createItemsTable(),
		createConversationsTable(),
		createMessagesTable(),
		createReactionsTable(),
		createFollowsTable(),
		createReviewsTable(),
		createGradingInfoTable(),
		createInvestmentAssetsTable(),
		createWarehouseStorageTable(),
		createReportsTable(),
		createTransactionsTable(),
		createPriceHistoryTable(),
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return err
		}
	}
	return nil
}

func createUsersTable() string {
	return `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		avatar_url TEXT,
		bio TEXT,
		rating REAL,
		selling_count INTEGER DEFAULT 0,
		transaction_count INTEGER DEFAULT 0,
		follower_count INTEGER DEFAULT 0,
		review_count INTEGER DEFAULT 0,
		is_admin INTEGER DEFAULT 0
	);`
}

func createItemsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS items (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		price INTEGER NOT NULL,
		description TEXT,
		condition TEXT,
		category TEXT,
		image_url TEXT NOT NULL,
		is_sold_out INTEGER DEFAULT 0,
		seller_id TEXT,
		is_invest_item INTEGER DEFAULT 0,
		view_count INTEGER DEFAULT 0,
		like_count INTEGER DEFAULT 0,
		product_group TEXT,
		FOREIGN KEY(seller_id) REFERENCES users(id)
	);`
}

func createConversationsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL,
		buyer_id TEXT NOT NULL,
		seller_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id),
		FOREIGN KEY(buyer_id) REFERENCES users(id),
		FOREIGN KEY(seller_id) REFERENCES users(id)
	);`
}

func createMessagesTable() string {
	return `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		sender_id TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(conversation_id) REFERENCES conversations(id),
		FOREIGN KEY(sender_id) REFERENCES users(id)
	);`
}

func createReactionsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS item_reactions (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		reaction_type TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(item_id, user_id, reaction_type),
		FOREIGN KEY(item_id) REFERENCES items(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`
}

func createFollowsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS user_follows (
		id TEXT PRIMARY KEY,
		follower_id TEXT NOT NULL,
		followee_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(follower_id, followee_id),
		FOREIGN KEY(follower_id) REFERENCES users(id),
		FOREIGN KEY(followee_id) REFERENCES users(id)
	);`
}

func createReviewsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS user_reviews (
		id TEXT PRIMARY KEY,
		reviewer_id TEXT NOT NULL,
		reviewee_id TEXT NOT NULL,
		rating REAL NOT NULL CHECK(rating >= 1 AND rating <= 5),
		comment TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(reviewer_id) REFERENCES users(id),
		FOREIGN KEY(reviewee_id) REFERENCES users(id)
	);`
}

func createGradingInfoTable() string {
	return `
	CREATE TABLE IF NOT EXISTS grading_info (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL UNIQUE,
		grader TEXT NOT NULL,
		grade REAL,
		cert_number TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	);`
}

func createInvestmentAssetsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS investment_assets (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL UNIQUE,
		purchase_date TEXT,
		original_price INTEGER,
		estimated_value INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	);`
}

func createWarehouseStorageTable() string {
	return `
	CREATE TABLE IF NOT EXISTS warehouse_storage (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL UNIQUE,
		warehouse_id TEXT NOT NULL,
		estimated_value INTEGER,
		storage_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	);`
}

func createReportsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS user_reports (
		id TEXT PRIMARY KEY,
		reporter_id TEXT NOT NULL,
		reported_user_id TEXT NOT NULL,
		reason TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(reporter_id) REFERENCES users(id),
		FOREIGN KEY(reported_user_id) REFERENCES users(id)
	);`
}

func createTransactionsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL,
		buyer_id TEXT NOT NULL,
		seller_id TEXT NOT NULL,
		price INTEGER NOT NULL,
		quantity INTEGER DEFAULT 1,
		transaction_type TEXT NOT NULL,
		warehouse INTEGER DEFAULT 0,
		status TEXT DEFAULT 'completed',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id),
		FOREIGN KEY(buyer_id) REFERENCES users(id),
		FOREIGN KEY(seller_id) REFERENCES users(id)
	);`
}

func createPriceHistoryTable() string {
	return `
	CREATE TABLE IF NOT EXISTS price_history (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL,
		price INTEGER NOT NULL,
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	);`
}

// addColumnIfMissing adds a column to a table if it is not already present
func addColumnIfMissing(db *sql.DB, table, column, definition string) error {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name='%s'", table, column)
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		alter := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
		if _, err := db.Exec(alter); err != nil {
			return err
		}
	}
	return nil
}

// ensureUserColumns migrates legacy user tables
func ensureUserColumns(db *sql.DB) error {
	columns := map[string]string{
		"follower_count":    "INTEGER DEFAULT 0",
		"review_count":      "INTEGER DEFAULT 0",
		"transaction_count": "INTEGER DEFAULT 0",
		"is_admin":          "INTEGER DEFAULT 0",
	}

	for column, definition := range columns {
		if err := addColumnIfMissing(db, "users", column, definition); err != nil {
			return err
		}
	}

	itemColumns := map[string]string{
		"product_group": "TEXT",
		"description":   "TEXT",
		"condition":     "TEXT",
		"category":      "TEXT",
	}

	for column, definition := range itemColumns {
		if err := addColumnIfMissing(db, "items", column, definition); err != nil {
			return err
		}
	}

	return nil
}

// ensureTransactionColumns migrates legacy transaction tables
func ensureTransactionColumns(db *sql.DB) error {
	return addColumnIfMissing(db, "transactions", "warehouse", "INTEGER DEFAULT 0")
}
