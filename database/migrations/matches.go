package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func MatchesMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS matches (
			id SERIAL PRIMARY KEY,
			event_has_game_types INT,
			match_name VARCHAR(500),
			schedule_by INT,
			round_no INT,
			scheduled_date DATE,
			venue VARCHAR,
			venue_link TEXT,
			start_time TIME,
			end_time TIME,
			isDraw boolean,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create matches table: %v", err))
		}
		fmt.Println("matches table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS matches;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop matches table: %v", err))
		}
		fmt.Println("matches table dropped successfully.")

	default:
		fmt.Println("Invalid action for matches migration. Use 'create' or 'drop'.")
	}
}
