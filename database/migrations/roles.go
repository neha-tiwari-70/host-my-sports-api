package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func RolesMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS roles (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(10) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create roles table: %v", err))
		}
		fmt.Println("roles table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS roles;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop roles table: %v", err))
		}
		fmt.Println("roles table dropped successfully.")

	default:
		fmt.Println("Invalid action for roles migration. Use 'create' or 'drop'.")
	}
}
