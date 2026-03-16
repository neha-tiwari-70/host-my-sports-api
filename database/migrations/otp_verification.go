package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func OTPMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS otp_verification (
			id SERIAL PRIMARY KEY,
			user_id BIGINT, 
			otp VARCHAR(255),
			expire_at TIMESTAMP, 
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create otp_verification table: %v", err))
		}
		fmt.Println("OTP verification table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS otp_verification;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop otp_verification table: %v", err))
		}
		fmt.Println("otp_verification table dropped successfully.")

	default:
		fmt.Println("Invalid action for otp_verification table migration. Use 'create' or 'drop'.")
	}
}
