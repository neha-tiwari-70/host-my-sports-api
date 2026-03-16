package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventTransactionsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS event_transactions (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL,
			event_id INT NOT NULL,
			payment_status VARCHAR(50) NOT NULL DEFAULT 'Pending' CHECK (payment_status IN ('Pending', 'Success', 'Failed', 'Expired')),
			razor_order_id VARCHAR(100),
			razor_payment_id VARCHAR(100),
			signature TEXT,
			fees INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create event_transactions table: %v", err))
		}
		fmt.Println("event_transactions table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS event_transactions;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop event_transactions table: %v", err))
		}
		fmt.Println("event_transactions table dropped successfully.")

	default:
		fmt.Println("Invalid action for event_transactions migration. Use 'create' or 'drop'.")
	}
}
