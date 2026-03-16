package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func UserMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			user_code VARCHAR(50) UNIQUE NOT NULL,  -- Unique user code
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100),
			password VARCHAR(255),
			mobile_no VARCHAR(255) NOT NULL,
			role_slug VARCHAR(255) DEFAULT 'user' NOT NULL,
			otp_status VARCHAR(15) DEFAULT 'Pending' NOT NULL CHECK (otp_status IN ('Pending', 'Verified', 'Expired')),
			email_status VARCHAR(15) DEFAULT 'Pending' NOT NULL CHECK (email_status IN ('Pending', 'Verified', 'Expired')),
			status VARCHAR(10) DEFAULT 'Pending' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete', 'Pending')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create users table: %v", err))
		}
		fmt.Println("Users table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS users;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop users table: %v", err))
		}
		fmt.Println("Users table dropped successfully.")

	default:
		fmt.Println("Invalid action for User migration. Use 'create' or 'drop'.")
	}
}

func SanitizeUserCodeColumn() {
	// Step 1: Clean user_code values to only contain numeric characters
	cleanQuery := `
		UPDATE users
		SET user_code = regexp_replace(user_code, '[^0-9]', '', 'g');
	`

	_, err := database.DB.Exec(cleanQuery)
	if err != nil {
		panic(fmt.Sprintf("Failed to clean user_code values: %v", err))
	}

	// Step 2: Convert user_code column from TEXT/VARCHAR to INT
	alterTypeQuery := `
		ALTER TABLE users
		ALTER COLUMN user_code TYPE INT
		USING user_code::INT;
	`

	_, err = database.DB.Exec(alterTypeQuery)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter user_code column type: %v", err))
	}

	fmt.Println("user_code column cleaned and converted to INT successfully.")
}
