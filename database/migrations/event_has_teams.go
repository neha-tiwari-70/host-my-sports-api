package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventHasTeamsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS event_has_teams (
			id SERIAL PRIMARY KEY,
			event_id INT,
			created_by INT,
			game_id INT,
			game_type_id INT,
			age_group_id NUMERIC,
			group_no NUMERIC,
			team_captain INT,
			team_name VARCHAR(255) ,
			team_logo_path VARCHAR(255),
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(100) DEFAULT 'Pending' NOT NULL CHECK (status IN ('Active','Pending', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create event_has_teams table: %v", err))
		}
		fmt.Println("event_has_teams table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_teams;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_teams table: %v", err))
		}
		fmt.Println("event_has_teams table dropped successfully.")

	default:
		fmt.Println("Invalid action for event_has_teams migration. Use 'create' or 'drop'.")
	}
}

func AddAgeGroupIdColumnTeams() {
	query := `
		ALTER TABLE event_has_teams
		ADD COLUMN IF NOT EXISTS age_group_id NUMERIC
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_teams table: %v", err))
	}
	fmt.Println("event_has_teams table altered successfully.")
}

func AddGroupNoColumn() {
	query := `
		ALTER TABLE event_has_teams
		ADD COLUMN IF NOT EXISTS group_no NUMERIC
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter event_has_teams table : %v", err))
	}
	fmt.Println("event_has_teams table altered to add group no column successfully.")
}
