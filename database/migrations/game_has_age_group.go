package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func GameHasAgeGroupMigration(action string) {
	switch action {
	case "create":
		query := `CREATE TABLE IF NOT EXISTS game_has_age_group (
			id SERIAL PRIMARY KEY,
			game_id INT,
			age_group_id INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create game_has_age_group table: %v", err))
		}
		fmt.Println("game_has_age_group table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS game_has_age_group;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop game_has_age_group table: %v", err))
		}
		fmt.Println("game_has_age_group table dropped successfully.")
	}
}
