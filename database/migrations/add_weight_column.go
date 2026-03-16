package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AddEventHasGamesWeight(action string) {
	switch action {
	case "alter":
		query := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS weight NUMERIC(5, 2)`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter event_has_games table: %v", err))
		}
		fmt.Println("event_has_games table altered successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_games;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_games: %v", err))
		}
		fmt.Println("event_has_games table droped successfully.")

	default:
		fmt.Println("Invalid action for event_has_games migration. Use 'create' or 'drop'.")
	}
}
