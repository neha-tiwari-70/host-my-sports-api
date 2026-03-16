package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func FundAccountsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS fund_accounts (
			id SERIAL PRIMARY KEY,
			organizer_id INT,
			contact_id INT,
			razorpay_fund_account_id VARCHAR(500),
			account_type VARCHAR(500),
			upi_id VARCHAR(200),
			account_number VARCHAR(18),
			ifsc VARCHAR(15),
			bank_name VARCHAR(200),
			name VARCHAR(500),
			active VARCHAR(200),
			status VARCHAR(200),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create fund_accounts table: %v", err))
		}
		fmt.Println("fund_accounts table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS fund_accounts;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop fund_accounts table: %v", err))
		}
		fmt.Println("fund_accounts table dropped successfully.")

	default:
		fmt.Println("Invalid action for fund_accounts migration. Use 'create' or 'drop'.")
	}
}
