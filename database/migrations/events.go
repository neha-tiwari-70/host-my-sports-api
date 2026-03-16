package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func EventsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			created_by_id INT NOT NULL,
			created_by_role Varchar(255),
			from_date DATE NOT NULL,
			to_date DATE NOT NULL,
			last_registration_date DATE,
			start_registration_date DATE,
			state_id VARCHAR(255) NOT NULL,
			city_id VARCHAR(255) NOT NULL,
			venue TYPE TEXT NOT NULL,
			fees VARCHAR(100) NOT NULL,
			logo VARCHAR(100) ,
			about TEXT ,
			facebook_link TYPE TEXT ,
			instagram_link TYPE TEXT ,
			linkedin_link TYPE TEXT,
			google_map_link TEXT,
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(100) DEFAULT 'Active' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create events table: %v", err))
		}
		fmt.Println("events table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS events;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop events table: %v", err))
		}
		fmt.Println("events table dropped successfully.")

	default:
		fmt.Println("Invalid action for events migration. Use 'create' or 'drop'.")
	}
}
