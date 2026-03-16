package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AddTshirtSize(action string) {
	switch action {
	case "alter":
		query := `
			ALTER TABLE event_has_users
			ADD COLUMN IF NOT EXISTS tshirt_size VARCHAR(10)
			CHECK (tshirt_size IN ('S', 'M', 'L', 'XL', '2XL', '3XL', '4XL', '5XL'));
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter event_has_users table: %v", err))
		}
		fmt.Println("event_has_users table altered successfully.")

	case "drop":
		query := `
			ALTER TABLE event_has_users
			DROP COLUMN IF EXISTS tshirt_size;
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop column from event_has_users table: %v", err))
		}
		fmt.Println("tshirt_size column dropped successfully.")

	default:
		fmt.Println("Invalid action for event_has_users migration. Use 'create' or 'drop'.")
	}
}
