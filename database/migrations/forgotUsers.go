package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func ForgotUsersMigration(action string) {

	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS forgot_users (
			id SERIAL PRIMARY KEY,
			is_admin BOOLEAN NOT NULL,
			email VARCHAR(100) NOT NULL,
			code VARCHAR(255) NOT NULL,
			status VARCHAR(10) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expire_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create forgot_users table: %v", err))
		}
		fmt.Println("Forgot_users table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS forgot_users;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop forgot_users table: %v", err))
		}
		fmt.Println("forgot_users table dropped successfully.")

	default:
		fmt.Println("Invalid action for forgot_users migration. Use 'create' or 'drop'.")
	}
}
