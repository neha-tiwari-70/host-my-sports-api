package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func PastTeamsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS past_teams(
			id SERIAL PRIMARY KEY,
			user_details_id INT,
			team_name VARCHAR(100)
		);
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create users table: %v", err))
		}
		fmt.Println("Users table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS users;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop users table: %v", err))
		}
		fmt.Println("Users table dropped successfully.")

	default:
		fmt.Println("Invalid action for User migration. Use 'create' or 'drop'.")
	}
}
