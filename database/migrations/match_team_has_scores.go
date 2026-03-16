package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func MatchTeamHasScoresMigration(action string) {
	switch action {
	case "create":
		query := `
			CREATE TABLE IF NOT EXISTS match_team_has_scores (
				id SERIAL PRIMARY KEY,
				match_id INT NOT NULL,
				team_id INT NOT NULL,
				player_id INT,
				set_number INT,
				points_scored INT NOT NULL DEFAULT 1,
				scored_at VARCHAR(255),
				is_penalty BOOLEAN DEFAULT FALSE
			);
		`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create match_team_has_scores table: %v", err))
		}
		fmt.Println("match_team_has_scores table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS match_team_has_scores;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop match_team_has_scores table: %v", err))
		}
		fmt.Println("match_team_has_scores table dropped successfully.")

	case "update":
		alterQuery := `
		-- Example of updating table: add a new column if needed
		ALTER TABLE match_team_has_scores 
		ADD COLUMN IF NOT EXISTS scored_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
		`
		_, err := database.DB.Exec(alterQuery)
		if err != nil {
			panic(fmt.Sprintf("Failed to update match_team_has_scores table: %v", err))
		}
		fmt.Println("match_team_has_scores table updated successfully.")

	default:
		fmt.Println("Invalid action for match_team_has_scores migration. Use 'create', 'drop' or 'update'.")
	}
}

func IsPenaltyMigration() {
	query := `ALTER TABLE match_team_has_scores
		ADD COLUMN IF NOT EXISTS is_penalty BOOLEAN DEFAULT FALSE;
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter match_team_has_scores table: %v", err))
	}
	fmt.Println("is_penalty column added successfully")
}

func AddDistanceAndMetricColumns() {
	query := `ALTER TABLE match_team_has_scores
    ADD COLUMN IF NOT EXISTS distance DECIMAL(10,2),
    ADD COLUMN IF NOT EXISTS metric VARCHAR(15)
        CHECK (metric IN ('meters', 'centimeters', 'feet', 'inches'));
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter match_team_has_scores table: %v", err))
	}
	fmt.Println("distance and metric columns added successfully")
}
