package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func UserHasProfileRoles(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS user_has_profile_roles (
			id SERIAL PRIMARY KEY,
			user_id NUMERIC,
			profile_role_id NUMERIC
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create user_has_profile_roles table: %v", err))
		}
		fmt.Println("user_has_profile_roles table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS user_has_profile_roles;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop user_has_profile_roles table: %v", err))
		}
		fmt.Println("user_has_profile_roles table dropped successfully.")

	default:
		fmt.Println("Invalid action for user_has_profile_roles migration. Use 'create' or 'drop'.")
	}
}
