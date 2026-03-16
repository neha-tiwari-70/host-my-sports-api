package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func ProfileRolesMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS profile_roles (
			id SERIAL PRIMARY KEY,
			role VARCHAR(100) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create profile roles table: %v", err))
		}
		fmt.Println("profile roles table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS profile_roles;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop profile roles table: %v", err))
		}
		fmt.Println("profile roles table dropped successfully.")

	default:
		fmt.Println("Invalid action for profile roles migration. Use 'create' or 'drop'.")
	}
}
