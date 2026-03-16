package seeders

import (
	"fmt"
	"sports-events-api/database"
)

func AgeGroupSeeder() {
	query := `
	INSERT INTO age_group (category, minAge, maxAge, slug)
VALUES
    -- Under categories
    ('Under 5', NULL, 5, 'under-5'),
    ('Under 6', NULL, 6, 'under-6'),
    ('Under 7', NULL, 7, 'under-7'),
    ('Under 8', NULL, 8, 'under-8'),
    ('Under 9', NULL, 9, 'under-9'),
    ('Under 10', NULL, 10, 'under-10'),
    ('Under 11', NULL, 11, 'under-11'),
    ('Under 12', NULL, 12, 'under-12'),
    ('Under 13', NULL, 13, 'under-13'),
    ('Under 14', NULL, 14, 'under-14'),
    ('Under 15', NULL, 15, 'under-15'),
    ('Under 16', NULL, 16, 'under-16'),
    ('Under 17', NULL, 17, 'under-17'),
    ('Under 18', NULL, 18, 'under-18'),
    ('Under 19', NULL, 19, 'under-19'),
    ('Under 20', NULL, 20, 'under-20'),
    ('Under 21', NULL, 21, 'under-21'),
    ('Under 23', NULL, 23, 'under-23'),
    ('Under 25', NULL, 25, 'under-25'),

    -- Open category
    ('Open', NULL, NULL, 'open'),

    -- Veteran / Seniors categories
    ('Senior (23+)', 23, NULL, 'Senior-23-plus'),
    ('Veteran (35-44)', 35, 44, 'veteran-35-44'),
    ('Veteran (40-49)', 40, 49, 'veteran-40-49'),
    ('Veteran (50-59)', 50, 59, 'veteran-50-59'),
    ('Veteran (50+)', 50, NULL, 'veteran-50-plus'),
    ('Super Veterans (45-54)', 45, 54, 'super-veterans-45-54'),
    ('Super veteran (60+)', 60, NULL, 'super-veteran-60-plus'),
    ('Seniors (65+)', 65, NULL, 'seniors-65-plus'),
    ('Masters (40+)', 40, NULL, 'masters-40-plus'),
    ('Masters (54+)', 54, NULL, 'masters-54-plus');
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to seed age_group table: %v", err))
	}

	fmt.Println("age_group table seeded successfully.")
}

func FillMissingAgeGroups() {
	query := `
	INSERT INTO age_group (category, minAge, maxAge, slug)
VALUES
    ('Under 6', NULL, 6, 'under-6'),
    ('Under 8', NULL, 8, 'under-8'),
    ('Under 12', NULL, 12, 'under-12'),
    ('Under 14', NULL, 14, 'under-14'),
    ('Under 16', NULL, 16, 'under-16'),
    ('Under 18', NULL, 18, 'under-18'),
    ('Under 20', NULL, 20, 'under-20'),
    --Seniors categories

    ('Masters (40+)', 40, NULL, 'masters-40-plus');
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to seed missing age groups: %v", err))
	}

	fmt.Println("Missing age groups seeded successfully.")
}

func AddAbove23AgeGroup() {
	query := `
        INSERT INTO age_group (category, minAge, maxAge, slug)
        SELECT 'Senior (23+)', 23, NULL, 'Senior-23-plus'
        WHERE NOT EXISTS (
            SELECT 1 FROM age_group WHERE slug = 'Senior-23-plus'
        );
    `

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to seed missing age groups: %v", err))
	}

	fmt.Println("Missing age groups seeded successfully.")
}

func AddAbove18AgeGroup() {
	query := `
        INSERT INTO age_group (category, minAge, maxAge, slug)
        SELECT 'Above 18', 18, NULL, 'above-18'
        WHERE NOT EXISTS (
            SELECT 1 FROM age_group WHERE slug = 'above-18'
        );
    `

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to seed missing age group: %v", err))
	}

	fmt.Println("Above 18 age group seed successfully.")
}
