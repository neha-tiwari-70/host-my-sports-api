package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AddMaximumSetPoint(action string) {
	switch action {
	case "alter":
		query := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS maximum_set_points VARCHAR(100)`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter event_has_games table %v", err))
		}
		fmt.Println("event_has_games table altered successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_games;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_games: %v", err))
		}
		fmt.Println("event_has_games table dropped successfully.")

	default:
		fmt.Println("Invalid action for event_has_games migration. Use 'create' or 'drop'.")
	}
}
