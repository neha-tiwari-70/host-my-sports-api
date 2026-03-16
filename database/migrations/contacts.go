package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func ContactsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS contacts (
			id SERIAL PRIMARY KEY,
			organizer_id INT,
			razorpay_contact_id VARCHAR(500),
			name VARCHAR(500),
			email VARCHAR(1000),
			mobile_no VARCHAR(13),
			type VARCHAR(200),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create contacts table: %v", err))
		}
		fmt.Println("contacts table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS contacts;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop contacts table: %v", err))
		}
		fmt.Println("contacts table dropped successfully.")

	default:
		fmt.Println("Invalid action for contacts migration. Use 'create' or 'drop'.")
	}
}
