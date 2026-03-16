package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func GamesMigration(action string) {
	switch action {
	case "create":
		query := `CREATE TABLE IF NOT EXISTS games (
			id SERIAL PRIMARY KEY,
			game_name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create games table: %v", err))
		}
		fmt.Println("games table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS games;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop games table: %v", err))
		}
		fmt.Println("games table dropped successfully.")

	case "alter":
		query := `ALTER TABLE games ADD COLUMN IF NOT EXISTS slug VARCHAR(100) NOT NULL;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to add slug column:%v", err))
		}
		fmt.Println("Slug column added successfully to game table.")

	default:
		fmt.Println("Invalid action for games migration. Use 'create' or 'drop'.")
	}

}
