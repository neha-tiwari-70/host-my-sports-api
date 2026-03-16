// package migrations

// import (
// 	"fmt"
// 	"sports-events-api/database"
// )

// func LevelOfCompetitionMigration(action string) {
// 	switch action {
// 	case "alter":
// 		query := `
// 		-- Drop old column
// 		ALTER TABLE events
// 		DROP COLUMN IF EXISTS level_of_competition;

// 		-- Add new column if not exists
// 		ALTER TABLE events
// 		ADD COLUMN IF NOT EXISTS level_of_competition_id INT;

// 		-- Add FK constraint safely (PostgreSQL doesn't support IF NOT EXISTS directly)
// 		DO $$
// 		BEGIN
// 			IF NOT EXISTS (
// 				SELECT 1 FROM information_schema.table_constraints
// 				WHERE constraint_name = 'fk_level_of_competition' AND table_name = 'events'
// 			) THEN
// 				ALTER TABLE events
// 				ADD CONSTRAINT fk_level_of_competition
// 				FOREIGN KEY (level_of_competition_id)
// 				REFERENCES level_of_competitions(id)
// 				ON DELETE SET NULL;
// 			END IF;
// 		END
// 		$$;
// 	`
// 		_, err := database.DB.Exec(query)
// 		if err != nil {
// 			panic(fmt.Sprintf("Failed to alter events table: %v", err))
// 		}
// 		fmt.Println("Alter completed: 'level_of_competition' removed, 'level_of_competition_id' added with FK.")

//		}
//	}
package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func LevelOfCompetitionMigration(action string) {
	switch action {
	case "alter":
		query := `
		-- Drop old column
		ALTER TABLE events
		DROP COLUMN IF EXISTS level_of_competition;

		-- Add new column if not exists
		ALTER TABLE events
		ADD COLUMN IF NOT EXISTS level_of_competition_id INT;
		`

		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to alter events table: %v", err))
		}

		fmt.Println("Alter completed: 'level_of_competition' removed, 'level_of_competition_id' added.")
	}
}
