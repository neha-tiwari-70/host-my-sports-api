package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func MatchPlayerCards(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS match_player_cards (
    id SERIAL PRIMARY KEY,
    match_id INT NOT NULL,
    team_id INT NOT NULL,
    player_id INT NOT NULL,
    yellow_cards INT DEFAULT 0,
    red_cards INT DEFAULT 0,
    suspensions INT DEFAULT 0,
    is_sent_off BOOLEAN DEFAULT FALSE,
    suspension_end VARCHAR(255)
);`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create match_player_cards: %v", err))
		}
		fmt.Println("match_palyer_cards table created successfully.")

	default:
		fmt.Println("Invalid action for match_player_cards migration. Use 'create' or 'Drop'.")
	}
}
