package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventHasImageMigration(action string) {
	switch action {
	case "create":
		query := `CREATE TABLE IF NOT EXISTS event_has_image (
			id SERIAL PRIMARY KEY,
			event_id INT NOT NULL,
			image TEXT NOT NULL,
			image_original_name TEXT, --28-Aug-25
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create event_has_image table: %v", err))
		}
		fmt.Println("event_has_image table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_has_image;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_has_image table: %v", err))
		}
		fmt.Println("event_has_image table dropped successfully.")

	default:
		fmt.Println("Invalid action for event_has_image migration. Use 'create' or 'drop'.")
	}
}

func EventHasIamgeOriginalName(action string) {
	switch action {
	case "alter":
		query := `ALTER TABLE event_has_image
			ADD COLUMN IF NOT EXISTS image_original_name TEXT
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter event_has_image table: %v", err))
		}
		fmt.Println("event_has_image (add iamge_original_name field)table altered successfully.")

	default:
		fmt.Println("Invalid action for event_has_image migration. Use 'create' or 'drop'.")
	}
}
