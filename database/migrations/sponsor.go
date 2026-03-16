package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func SponsorMigration(action string) {
	switch action {
	case "create":
		query := `CREATE TABLE IF NOT EXISTS event_has_sponsors (
			id SERIAL PRIMARY KEY,
			event_id INT NOT NULL,
			sponsor_title TEXT NOT NULL,
			sponsor_logo TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create sponsor table: %v", err))
		}
		fmt.Println("sponsor table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_sponsors;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop sponsor table: %v", err))
		}
		fmt.Println("sponsor table dropped successfully.")

	default:
		fmt.Println("Invalid action for sponsor migration. Use 'create' or 'drop'.")
	}
}
