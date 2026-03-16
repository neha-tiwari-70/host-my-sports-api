package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func PayoutMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS payouts (
			id SERIAL PRIMARY KEY,
			organizer_id INT,
			event_id INT,
			fund_account_id INT,
			razorpay_payout_id VARCHAR(200),
			amount INT,
			currency VARCHAR(10),
			mode VARCHAR(200),
			purpose VARCHAR(200),
			failure_reason VARCHAR(200),
			error_code VARCHAR(200),
			error_description TEXT,
			status VARCHAR(10) DEFAULT 'pending' NOT NULL CHECK (status IN ('pending', 'processing', 'processed', 'failed', 'reversed', 'queued')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create payouts table: %v", err))
		}
		fmt.Println("payouts table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS payouts;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop payouts table: %v", err))
		}
		fmt.Println("payouts table dropped successfully.")

	default:
		fmt.Println("Invalid action for payouts migration. Use 'create' or 'drop'.")
	}
}
