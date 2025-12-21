package main

import "database/sql"

// syncListingsCounts recalculates listings_count for all users based on actual unsold items
func syncListingsCounts(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE users
		SET listings_count = (
			SELECT COUNT(*)
			FROM items i
			WHERE i.seller_id = users.id AND i.is_sold_out = 0
		)
	`)
	return err
}

// syncTransactionCounts recalculates transaction_count for all users based on recorded transactions
func syncTransactionCounts(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE users
		SET transaction_count = (
			SELECT COUNT(*)
			FROM transactions t
			WHERE t.seller_id = users.id OR t.buyer_id = users.id
		)
	`)
	return err
}

// syncUserRatings clears seeded ratings and recalculates from user_reviews only
func syncUserRatings(db *sql.DB) error {
	// First, clear all existing ratings to ensure clean state
	_, err := db.Exec(`UPDATE users SET rating = NULL, review_count = 0`)
	if err != nil {
		return err
	}

	// Now recompute rating and review_count from actual user_reviews
	_, err = db.Exec(`
		UPDATE users
		SET 
			rating = (
				SELECT AVG(r.rating)
				FROM user_reviews r
				WHERE r.reviewee_id = users.id
			),
			review_count = (
				SELECT COUNT(*)
				FROM user_reviews r
				WHERE r.reviewee_id = users.id
			)
		WHERE EXISTS (
			SELECT 1 FROM user_reviews r WHERE r.reviewee_id = users.id
		)
	`)
	return err
}

// syncFollowerCounts recalculates follower_count from user_follows table
func syncFollowerCounts(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE users
		SET follower_count = (
			SELECT COUNT(*)
			FROM user_follows f
			WHERE f.followee_id = users.id
		)
	`)
	return err
}
