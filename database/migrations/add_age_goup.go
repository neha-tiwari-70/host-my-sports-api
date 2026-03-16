package migrations

import (
	"fmt"
	"sports-events-api/database"
)

func AddUnder14AgeGroup() {
	query := `
		INSERT INTO age_group (category, minAge, maxAge, slug)
		VALUES ('Under 14', NULL, 14, 'under-14')
		ON CONFLICT DO NOTHING;
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		fmt.Println("Error inserting Under 14 age group -->", err)
	} else {
		fmt.Println("Successfully inserted Under 14 age group")
	}
}
