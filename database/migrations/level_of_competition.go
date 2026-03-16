package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func CreateLevelOfCompetitionTable(action string) {
	switch action {
	case "create":
		query := `CREATE TABLE IF NOT EXISTS level_of_competitions (
			id SERIAL PRIMARY KEY,
			title VARCHAR(500),
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create level_of_competitions table: %v", err))
		}
		fmt.Println("level_of_competitions table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS level_of_competitions;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop level_of_competitions table: %v", err))
		}
		fmt.Println("level_of_competitions table dropped successfully.")

	default:
		fmt.Println("Invalid action for level_of_competitions migration. Use 'create' or 'drop'.")
	}
}
