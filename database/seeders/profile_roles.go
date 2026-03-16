package seeders

import (
	"fmt"
	"sports-events-api/database"
)

func ProfileRolesSeeder() {
	query := `
	INSERT INTO profile_roles (role, slug, status, created_at, updated_at)
	VALUES
		('Player', 'player', 'Active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		('Coach', 'coach', 'Active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		('Referee', 'referee', 'Active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to seed profile roles table: %v", err))
	}

	fmt.Println("Profile Roles table seeded successfully.")
}
