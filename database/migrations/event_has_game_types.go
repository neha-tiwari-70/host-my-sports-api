package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventHasGameTypesMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS event_has_game_types (
			id SERIAL PRIMARY KEY,
			event_has_game_id NUMERIC,
			game_type_id NUMERIC,
			age_group_id NUMERIC,
			min_player INT NOT NULL DEFAULT 0,
			max_player INT NOT NULL DEFAULT 0,
			is_last_round Boolean DEFAULT FALSE
		);`
		// ,is_last_round Boolean DEFAULT FALSE
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create event_has_game_types table: %v", err))
		}
		fmt.Println("event_has_game_types table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_game_types;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_game_types table: %v", err))
		}
		fmt.Println("event_has_game_types table dropped successfully.")

	default:
		fmt.Println("Invalid action for event_has_game_types migration. Use 'create' or 'drop'.")
	}
}

func AddAgeGroupIdColumn() {
	query := `
		ALTER TABLE event_has_game_types
		ADD COLUMN IF NOT EXISTS age_group_id NUMERIC
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_game_types table: %v", err))
	}
	fmt.Println("event_has_game_types table altered successfully.")
}

func AddMinMaxPlayerColumn() {
	query := `
		-- 1: Add new columns if not exists
		ALTER TABLE event_has_game_types
		ADD COLUMN IF NOT EXISTS min_player INT NOT NULL DEFAULT 0,
		ADD COLUMN IF NOT EXISTS max_player INT NOT NULL DEFAULT 0;

		-- 2: Copy data from event_has_games to event_has_game_types
		UPDATE event_has_game_types egt
		SET min_player = eg.number_of_players,
		    max_player = eg.number_of_players
		FROM event_has_games eg
		WHERE egt.event_has_game_id = eg.id;

		-- 3: Drop old column from evenbt_has_games
		ALTER TABLE event_has_games
		DROP COLUMN IF EXISTS number_of_players;
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to move players column: %v", err))
	}
	fmt.Println("number_of_players moved to event_has_game_types successfully.")
}
