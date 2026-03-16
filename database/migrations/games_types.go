package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func GamesTypesMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS games_types (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create games_types table: %v", err))
		}
		fmt.Println("games_types table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS games_types;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop games_types table: %v", err))
		}
		fmt.Println("games_types table dropped successfully.")

	default:
		fmt.Println("Invalid action for games_types migration. Use 'create' or 'drop'.")
	}
}
