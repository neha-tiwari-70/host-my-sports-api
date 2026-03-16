package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func VerifiedTransactionMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS verified_transaction (
			id SERIAL PRIMARY KEY,
			razorpay_payment_id VARCHAR(255) Unique,
			payment_method VARCHAR(255),
			card_id VARCHAR(255),
			bank VARCHAR(255),
			wallet VARCHAR(255),
			upi VARCHAR(255),
			email VARCHAR(255),
			contact VARCHAR(255),
			card_details VARCHAR(255),
			card_holder_name VARCHAR(255),
			card_type VARCHAR(255),
			card_network VARCHAR(255),
			issuer VARCHAR(255),
			emi boolean,
			bank_transaction_id VARCHAR(255),
			rrn VARCHAR(255),
			upi_transaction_id VARCHAR(255),
			auth_code VARCHAR(255),
			amount integer,
			error_description VARCHAR(255),
			error_source VARCHAR(255),
			error_step VARCHAR(255),
			error_reason VARCHAR(255),
			category VARCHAR(255),
			order_id VARCHAR(255),
			pnr VARCHAR(255),
			status VARCHAR(10) DEFAULT 'Pending' NOT NULL CHECK (status IN ('authorized', 'captured', 'failed')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create verified_transaction table: %v", err))
		}
		fmt.Println("verified_transaction table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS verified_transaction;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop verified_transaction table: %v", err))
		}
		fmt.Println("verified_transaction table dropped successfully.")

	default:
		fmt.Println("Invalid action for User migration. Use 'create' or 'drop'.")
	}
}
