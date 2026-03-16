package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func BlogsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS blogs (
			id SERIAL PRIMARY KEY,
			title TEXT,
			image VARCHAR(500),
			description TEXT,
			content TEXT,
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create blogs table: %v", err))
		}
		fmt.Println("blogs table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS blogs;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop blogs table: %v", err))
		}
		fmt.Println("blogs table dropped successfully.")

	default:
		fmt.Println("Invalid action for blogs migration. Use 'create' or 'drop'.")
	}
}
