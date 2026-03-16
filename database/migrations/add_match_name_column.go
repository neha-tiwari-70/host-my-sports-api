package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AddMatchNameColumn(action string) {
	switch action {
	case "alter":
		query := `
		ALTER TABLE matches
		ADD COLUMN IF NOT EXISTS match_name VARCHAR(500)`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter matches table: %v", err))
		}
		fmt.Println("matches table altered successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS matches;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop matches table: %v", err))
		}
		fmt.Println("matches table dropped successfully.")

	default:
		fmt.Println("Invalid action for matches migration. Use 'create' or 'drop'.")
	}
}
