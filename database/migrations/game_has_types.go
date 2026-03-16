package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func GameHasTypesMigration(action string) {
	switch action {
	case "create":
		query := `CREATE TABLE IF NOT EXISTS game_has_types (
			id SERIAL PRIMARY KEY,
			game_id INT NOT NULL,
			game_type_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (game_id, game_type_id)
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create game_has_types table: %v", err))
		}
		fmt.Println("game_has_types table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS game_has_types;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop game_has_types table: %v", err))
		}
		fmt.Println("game_has_types table dropped successfully.")

	default:
		fmt.Println("Invalid action for game_has_types migration. Use 'create' or 'drop'.")
	}
}
