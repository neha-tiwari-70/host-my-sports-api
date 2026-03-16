package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func UserHasInterestedGames(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS user_has_interested_games (
			id SERIAL PRIMARY KEY,
			user_id NUMERIC,
			game_id NUMERIC
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create user_has_interested_games table: %v", err))
		}
		fmt.Println("user_has_interested_games table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS user_has_interested_games;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop user_has_interested_games table: %v", err))
		}
		fmt.Println("user_has_interested_games table dropped successfully.")

	default:
		fmt.Println("Invalid action for user_has_interested_games migration. Use 'create' or 'drop'.")
	}
}
