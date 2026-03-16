package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func MatchesHasTeamsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS matches_has_teams (
			id SERIAL PRIMARY KEY,
			match_id INT,
			team_id INT,
			points BIGINT
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create matches_has_teams table: %v", err))
		}
		fmt.Println("matches_has_teams table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS matches_has_teams;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop matches_has_teams table: %v", err))
		}
		fmt.Println("matches_has_teams table dropped successfully.")

	default:
		fmt.Println("Invalid action for matches_has_teams migration. Use 'create' or 'drop'.")
	}
}
