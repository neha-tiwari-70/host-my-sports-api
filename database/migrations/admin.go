package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AdminMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS admin (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) NOT NULL,
			password VARCHAR(255) NOT NULL,
			mobile_no VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create admin table: %v", err))
		}
		fmt.Println("Admin table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS admin;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop admin table: %v", err))
		}
		fmt.Println("Admin table dropped successfully.")

	default:
		fmt.Println("Invalid action for Admin migration. Use 'create' or 'drop'.")
	}
}
