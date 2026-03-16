package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventHasGamesMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS event_has_games (
			id SERIAL PRIMARY KEY,
			event_id INT,
			game_id INT,
			team_size NUMERIC,
			duration VARCHAR(255),
			type_of_tournament VARCHAR(100),
			max_registration VARCHAR(100),
			maximum_set_points VARCHAR(100),
			sets VARCHAR(100),
			fees NUMERIC DEFAULT 0, -- New field added
			number_of_overs INT DEFAULT 0,
			ball_type VARCHAR(100),
			cycle_type VARCHAR(100),
			distance_category INT DEFAULT 0,
			is_tshirt_size_required BOOLEAN DEFAULT FALSE, --22-july-2025
			number_of_overs INT DEFAULT 0, --04-sep-2025
			ball_type VARCHAR(100) CHECK (ball_type IN ('season ball','tennis ball','plastic ball','soft ball','rubber ball','tape ball')), --04-sep-2025
			cycle_type VARCHAR(100) CHECK (cycle_type IN ('road bike','mountain bike','hybrid bike','bmx','any cycle')), --12-sep-2025
			distance_category INT DEFAULT 0, --12-sep-2025
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create event_has_games table: %v", err))
		}
		fmt.Println("event_has_games table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_games;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_games table: %v", err))
		}
		fmt.Println("event_has_games table dropped successfully.")

	case "update":
		alterQuery := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS fees NUMERIC DEFAULT 0;
		`
		_, err := database.DB.Exec(alterQuery)
		if err != nil {
			panic(fmt.Sprintf("Failed to update event_has_games table: %v", err))
		}
		fmt.Println("event_has_games table updated successfully.")

	default:
		fmt.Println("Invalid action for event_has_games migration. Use 'create', 'drop' or 'update'.")
	}
}

func UpdateGameFields(gameType string, eventID int) {
	fields := map[string][]string{
		"Football":     {"number_of_players", "type_of_tournament", "duration", "gametype", "max_registration"},
		"Table Tennis": {"gametype", "type_of_tournament", "sets", "category", "age_group", "max_registration"},
		"Chess":        {"type_of_tournament", "gametype", "age_group", "max_registration"},
	}

	if columns, exists := fields[gameType]; exists {
		updateQuery := "UPDATE event_has_games SET "
		for i, column := range columns {
			if i > 0 {
				updateQuery += ", "
			}
			updateQuery += fmt.Sprintf("%s = NULL", column)
		}
		updateQuery += " WHERE event_id = $1;"

		_, err := database.DB.Exec(updateQuery, eventID)
		if err != nil {
			panic(fmt.Sprintf("Failed to update fields for %s: %v", gameType, err))
		}
		fmt.Printf("Fields for %s updated successfully.\n", gameType)
	}
}

func DeleteCategoryAgeGroupColumn() {
	query := `
		ALTER TABLE event_has_games
		DROP COLUMN IF EXISTS category,
		DROP COLUMN IF EXISTS age_group;
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_game_types table: %v", err))
	}
	fmt.Println("event_has_game table with deletion of category and age group column altered successfully.")
}

func AddIsTshirtSizeRequiredColumn() {
	query := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS is_tshirt_size_required BOOLEAN DEFAULT FALSE
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_games table: %v", err))
	}
	fmt.Println("event_has_games table altered to add is_tshirt_size_required column successfully.")
}

func AddNumberOfOversColumn() {
	query := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS number_of_overs INT DEFAULT 0
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_games table: %v", err))
	}
	fmt.Println("event_has_games table altered to add number_of_overs column successfully.")
}

func AddBallTypeColumn() {
	query := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS ball_type VARCHAR(100) CHECK (ball_type IN ('season ball','tennis ball','plastic ball','soft ball','rubber ball','tape ball'))
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_games table: %v", err))
	}
	fmt.Println("event_has_games table altered to add ball_type column successfully.")
}

func AddCycleTypeColumn() {
	query := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS cycle_type VARCHAR(100) CHECK (cycle_type IN ('road bike','mountain bike','hybrid bike','bmx','any cycle'))
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_games table: %v", err))
	}
	fmt.Println("event_has_games table altered to add cycle_type column successfully.")
}

func AddDistanceCategoryColumn() {
	query := `
		ALTER TABLE event_has_games
		ADD COLUMN IF NOT EXISTS distance_category INT DEFAULT 0
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_games table: %v", err))
	}
	fmt.Println("event_has_games table altered to add distance_category column successfully.")
}
