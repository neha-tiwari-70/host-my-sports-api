package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventsAlterMigration(action string) {
	switch action {
	case "alter":
		query := `
		ALTER TABLE events
		DROP COLUMN IF EXISTS youtube_link,
		ADD COLUMN IF NOT EXISTS linkedin_link VARCHAR(100);
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter events table: %v", err))
		}
		fmt.Println("events table altered successfully (youtube_link removed, linkedin_link added).")

	default:
		fmt.Println("Invalid action for events migration. Use 'create' or 'drop'.")
	}
}
