package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventHasUsersMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS event_has_users (
			id SERIAL PRIMARY KEY,
			event_id INT,
			game_id INT,
			user_id INT,
			event_has_team_id INT
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create event_has_users table: %v", err))
		}
		fmt.Println("event_has_users table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_users;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_users table: %v", err))
		}
		fmt.Println("event_has_users table dropped successfully.")

	default:
		fmt.Println("Invalid action for event_has_users migration. Use 'create' or 'drop'.")
	}
}
