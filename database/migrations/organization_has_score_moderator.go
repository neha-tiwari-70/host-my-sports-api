package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func OrganizationHasScoreMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS organization_has_score_moderator (
			id SERIAL PRIMARY KEY,
			organization_id NUMERIC,
			moderator_id NUMERIC,
			event_id NUMERIC,
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create organization_has_score_moderator table: %v", err))
		}
		fmt.Println("organization_has_score_moderator table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS organization_has_score_moderator;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop organization_has_score_moderator table: %v", err))
		}
		fmt.Println("organization_has_score_moderator table dropped successfully.")

	default:
		fmt.Println("Invalid action for organization_has_score_moderator migration. Use 'create' or 'drop'.")
	}
}

func AddEventIdColumn() {
	query := `
		ALTER TABLE organization_has_score_moderator
		ADD COLUMN IF NOT EXISTS event_id NUMERIC
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter organization_has_score_moderator table: %v", err))
	}
	fmt.Println("organization_has_score_moderator table altered successfully.")
}
