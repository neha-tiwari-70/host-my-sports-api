package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AgeGroupMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS age_group (
			id SERIAL PRIMARY KEY,
			category VARCHAR(100) NOT NULL,
			minAge INT,
			maxAge INT,
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create age_group table: %v", err))
		}
		fmt.Println("age_group table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS age_group;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop age_group table: %v", err))
		}
		fmt.Println("age_group table dropped successfully.")

	default:
		fmt.Println("Invalid action for age_group migration. Use 'create' or 'drop'.")
	}
}
