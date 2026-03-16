package seeders

import (
	"fmt"
	"sports-events-api/database"
)

func RolesSeeder() {
	query := `
	INSERT INTO roles (name, slug, status, created_at, updated_at)
	VALUES
		('User', 'user', 'Active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		('Organizer', 'organizer', 'Active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to seed roles table: %v", err))
	}

	fmt.Println("Roles table seeded successfully.")
}
