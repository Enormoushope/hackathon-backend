package main

import (
	"database/sql"
	"fmt"
)

// createTables creates all required database tables
func createTables(db *sql.DB, isMySQL bool) error {
	tables := []string{
		createUsersTable(isMySQL),
		createItemsTable(isMySQL),
		createConversationsTable(isMySQL),
		createMessagesTable(isMySQL),
		createReactionsTable(isMySQL),
		createFollowsTable(isMySQL),
		createReviewsTable(isMySQL),
		createGradingInfoTable(isMySQL),
		createInvestmentAssetsTable(isMySQL),
		createWarehouseStorageTable(isMySQL),
		createReportsTable(isMySQL),
		createTransactionsTable(isMySQL),
		createPriceHistoryTable(isMySQL),
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return err
		}
	}
	return nil
}

func createUsersTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	real := realType(isMySQL)
	integer := intType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS users (
		id %s PRIMARY KEY,
		name %s NOT NULL,
		avatar_url %s,
		bio %s,
		rating %s,
		listings_count %s DEFAULT 0,
		transaction_count %s DEFAULT 0,
		follower_count %s DEFAULT 0,
		review_count %s DEFAULT 0,
		is_verified %s DEFAULT 0
	) %s;`, id, text, text, text, real, integer, integer, integer, integer, integer, tableOptions(isMySQL))
}

func createItemsTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	integer := intType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS items (
		id %s PRIMARY KEY,
		title %s NOT NULL,
		price %s NOT NULL,
		description %s,
		condition %s,
		category %s,
		image_url %s NOT NULL,
		is_sold_out %s DEFAULT 0,
		seller_id %s,
		is_invest_item %s DEFAULT 0,
		view_count %s DEFAULT 0,
		like_count %s DEFAULT 0,
		product_group %s,
		FOREIGN KEY(seller_id) REFERENCES users(id)
	) %s;`, id, text, integer, text, text, text, text, integer, id, integer, integer, integer, text, tableOptions(isMySQL))
}

func createConversationsTable(isMySQL bool) string {
	id := idType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS conversations (
		id %s PRIMARY KEY,
		item_id %s NOT NULL,
		buyer_id %s NOT NULL,
		seller_id %s NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id),
		FOREIGN KEY(buyer_id) REFERENCES users(id),
		FOREIGN KEY(seller_id) REFERENCES users(id)
	) %s;`, id, id, id, id, tableOptions(isMySQL))
}

func createMessagesTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS messages (
		id %s PRIMARY KEY,
		conversation_id %s NOT NULL,
		sender_id %s NOT NULL,
		content %s NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(conversation_id) REFERENCES conversations(id),
		FOREIGN KEY(sender_id) REFERENCES users(id)
	) %s;`, id, id, id, text, tableOptions(isMySQL))
}

func createReactionsTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS item_reactions (
		id %s PRIMARY KEY,
		item_id %s NOT NULL,
		user_id %s NOT NULL,
		reaction_type %s NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(item_id, user_id, reaction_type),
		FOREIGN KEY(item_id) REFERENCES items(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	) %s;`, id, id, id, text, tableOptions(isMySQL))
}

func createFollowsTable(isMySQL bool) string {
	id := idType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS user_follows (
		id %s PRIMARY KEY,
		follower_id %s NOT NULL,
		followee_id %s NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(follower_id, followee_id),
		FOREIGN KEY(follower_id) REFERENCES users(id),
		FOREIGN KEY(followee_id) REFERENCES users(id)
	) %s;`, id, id, id, tableOptions(isMySQL))
}

func createReviewsTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	real := realType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS user_reviews (
		id %s PRIMARY KEY,
		reviewer_id %s NOT NULL,
		reviewee_id %s NOT NULL,
		rating %s NOT NULL,
		comment %s,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(reviewer_id) REFERENCES users(id),
		FOREIGN KEY(reviewee_id) REFERENCES users(id)
	) %s;`, id, id, id, real, text, tableOptions(isMySQL))
}

func createGradingInfoTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	real := realType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS grading_info (
		id %s PRIMARY KEY,
		item_id %s NOT NULL UNIQUE,
		grader %s NOT NULL,
		grade %s,
		cert_number %s,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	) %s;`, id, id, text, real, text, tableOptions(isMySQL))
}

func createInvestmentAssetsTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	integer := intType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS investment_assets (
		id %s PRIMARY KEY,
		item_id %s NOT NULL UNIQUE,
		purchase_date %s,
		original_price %s,
		estimated_value %s,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	) %s;`, id, id, text, integer, integer, tableOptions(isMySQL))
}

func createWarehouseStorageTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	integer := intType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS warehouse_storage (
		id %s PRIMARY KEY,
		item_id %s NOT NULL UNIQUE,
		warehouse_id %s NOT NULL,
		estimated_value %s,
		storage_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	) %s;`, id, id, text, integer, tableOptions(isMySQL))
}

func createReportsTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS user_reports (
		id %s PRIMARY KEY,
		reporter_id %s NOT NULL,
		reported_user_id %s NOT NULL,
		reason %s NOT NULL,
		description %s,
		status %s DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(reporter_id) REFERENCES users(id),
		FOREIGN KEY(reported_user_id) REFERENCES users(id)
	) %s;`, id, id, id, text, text, text, tableOptions(isMySQL))
}

func createTransactionsTable(isMySQL bool) string {
	id := idType(isMySQL)
	text := textType(isMySQL)
	integer := intType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS transactions (
		id %s PRIMARY KEY,
		item_id %s NOT NULL,
		buyer_id %s NOT NULL,
		seller_id %s NOT NULL,
		price %s NOT NULL,
		quantity %s DEFAULT 1,
		transaction_type %s NOT NULL,
		warehouse %s DEFAULT 0,
		status %s DEFAULT 'completed',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id),
		FOREIGN KEY(buyer_id) REFERENCES users(id),
		FOREIGN KEY(seller_id) REFERENCES users(id)
	) %s;`, id, id, id, id, integer, integer, text, integer, text, tableOptions(isMySQL))
}

func createPriceHistoryTable(isMySQL bool) string {
	id := idType(isMySQL)
	integer := intType(isMySQL)
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS price_history (
		id %s PRIMARY KEY,
		item_id %s NOT NULL,
		price %s NOT NULL,
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(item_id) REFERENCES items(id)
	) %s;`, id, id, integer, tableOptions(isMySQL))
}

// addColumnIfMissing adds a column to a table if it is not already present
func addColumnIfMissing(db *sql.DB, table, column, definition string, isMySQL bool) error {
	var count int
	if isMySQL {
		query := "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?"
		if err := db.QueryRow(query, table, column).Scan(&count); err != nil {
			return err
		}
	} else {
		query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name='%s'", table, column)
		if err := db.QueryRow(query).Scan(&count); err != nil {
			return err
		}
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
func ensureUserColumns(db *sql.DB, isMySQL bool) error {
	columns := map[string]string{
		"follower_count":    "INTEGER DEFAULT 0",
		"review_count":      "INTEGER DEFAULT 0",
		"transaction_count": "INTEGER DEFAULT 0",
		"is_admin":          "INTEGER DEFAULT 0",
	}

	for column, definition := range columns {
		if err := addColumnIfMissing(db, "users", column, definition, isMySQL); err != nil {
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
		if err := addColumnIfMissing(db, "items", column, definition, isMySQL); err != nil {
			return err
		}
	}

	return nil
}

// ensureTransactionColumns migrates legacy transaction tables
func ensureTransactionColumns(db *sql.DB, isMySQL bool) error {
	return addColumnIfMissing(db, "transactions", "warehouse", "INTEGER DEFAULT 0", isMySQL)
}

func idType(isMySQL bool) string {
	if isMySQL {
		return "VARCHAR(64)"
	}
	return "TEXT"
}

func textType(isMySQL bool) string {
	if isMySQL {
		return "TEXT"
	}
	return "TEXT"
}

func intType(isMySQL bool) string {
	if isMySQL {
		return "INT"
	}
	return "INTEGER"
}

func realType(isMySQL bool) string {
	if isMySQL {
		return "DOUBLE"
	}
	return "REAL"
}

func tableOptions(isMySQL bool) string {
	if isMySQL {
		return "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
	}
	return ""
}
