package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func BankDetailsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS bank_details (
			id SERIAL PRIMARY KEY,
			user_id INT,
			upi_id VARCHAR(100),
			qr_code VARCHAR(300),
			account_name VARCHAR(200),
			account_no VARCHAR(25),
			account_type VARCHAR(20),
			branch_name VARCHAR(100),
			ifsc_code VARCHAR(25),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create bank_details table: %v", err))
		}
		fmt.Println("bank_details table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS bank_details;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop bank_details table: %v", err))
		}
		fmt.Println("bank_details table dropped successfully.")

	default:
		fmt.Println("Invalid action for bank_details migration. Use 'create' or 'drop'.")
	}
}
