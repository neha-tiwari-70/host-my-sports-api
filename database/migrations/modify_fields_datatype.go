package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventsModifyDatatype(action string) {
	switch action {
	case "alter":
		query := `
		ALTER TABLE events
		ALTER COLUMN venue TYPE TEXT,
		ALTER COLUMN linkedin_link TYPE TEXT,
		ALTER COLUMN instagram_link TYPE TEXT,
		ALTER COLUMN facebook_link TYPE TEXT;
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter events table: %v", err))
		}
		fmt.Println("events table altered successfully (venue, linkedin_link, instagram_link, facebook_link changed to TEXT).")

	default:
		fmt.Println("Invalid action for events migration V2. Use 'alter'.")
	}
}
