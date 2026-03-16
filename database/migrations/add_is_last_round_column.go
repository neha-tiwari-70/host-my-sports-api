package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AddIsLastRoundColumn(action string) {
	switch action {
	case "alter":
		query := `
		ALTER TABLE event_has_game_types
		ADD COLUMN IF NOT EXISTS  is_last_round Boolean DEFAULT FALSE`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter and add is_last_round column in  event_has_game_types table: %v", err))
		}
		fmt.Println("event_has_game_types table altered successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS matches;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_game_types table: %v", err))
		}
		fmt.Println("event_has_game_types table dropped successfully.")

	default:
		fmt.Println("Invalid action for event_has_game_types migration. Use 'create' or 'drop'.")
	}
}
