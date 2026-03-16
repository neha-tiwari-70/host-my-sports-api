package migrations

import (
	"fmt"
	"log"
	"sports-events-api/database"
)

func UserDetailsMigration(action string) {
	switch action {
	case "create":
		query := `
		CREATE TABLE IF NOT EXISTS user_details (
			id SERIAL PRIMARY KEY,
			user_id INT,
			organization_name VARCHAR(250),
			nationality VARCHAR(100),
			skill_level VARCHAR(250),
			address TEXT,
			facebook_link Text,
			insta_link Text,
			height VARCHAR(100),
			weight VARCHAR(100),
			current_team VARCHAR(100),
			--gender VARCHAR(10) CHECK (gender IN ('Male', 'Female', 'Other')),
			gender VARCHAR(10) CHECK (gender IN ('Male', 'Female')),
			state INT,
			city INT,
			aadhar_no VARCHAR(100),
			dob DATE,
			profile_image_path VARCHAR(1000),
			status VARCHAR(10) DEFAULT 'Pending' NOT NULL CHECK (status IN ('Active', 'Inactive', 'Delete', 'Pending')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to create users table: %v", err))
		}
		fmt.Println("User_details table created successfully.")

	case "drop":
		query := `DROP TABLE IF EXISTS user_details;`
		_, err := database.DB.Exec(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to drop users table: %v", err))
		}
		fmt.Println("Users table dropped successfully.")

	default:
		fmt.Println("Invalid action for User migration. Use 'create' or 'drop'.")
	}
}

func AssignGenderToUsers() {
	query := `
		WITH matched_users AS (
			SELECT u.id, ud.id AS user_details_id,
				   ROW_NUMBER() OVER (ORDER BY RANDOM()) AS row_num,
				   COUNT(*) OVER () AS total
			FROM users u
			JOIN user_details ud ON ud.user_id = u.id
			WHERE (ud.gender IS NULL OR TRIM(ud.gender) = '')
		),
		gender_assignment AS (
			SELECT user_details_id,
				   CASE
					   WHEN row_num <= (total * 0.6) THEN 'Male'
					   ELSE 'Female'
				   END AS assigned_gender
			FROM matched_users
		)
		UPDATE user_details ud
		SET gender = ga.assigned_gender,
			updated_at = NOW()
		FROM gender_assignment ga
		WHERE ud.id = ga.user_details_id;
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to assign gender to users: %v", err))
	}

	fmt.Println("Gender assigned to users successfully (60% male, 40% female).")
}

func UpdateGenderConstraint() {
	// Step 1: Drop constraint if it exists
	dropConstraintQuery := `
		ALTER TABLE user_details
		DROP CONSTRAINT IF EXISTS gender_check;
	`

	_, err := database.DB.Exec(dropConstraintQuery)
	if err != nil {
		panic(fmt.Sprintf("Failed to drop existing gender constraint: %v", err))
	}

	// Step 2: Add the new gender constraint
	addConstraintQuery := `
		ALTER TABLE user_details
		ALTER COLUMN gender TYPE VARCHAR(10),
		ADD CONSTRAINT gender_check CHECK (gender IN ('Male', 'Female'));
	`

	_, err = database.DB.Exec(addConstraintQuery)
	if err != nil {
		panic(fmt.Sprintf("Failed to add gender constraint: %v", err))
	}

	fmt.Println("Gender column constraint enforced successfully.")
}

func AddContactPersonNameColumn() {
	query := `
		ALTER TABLE user_details
		ADD COLUMN IF NOT EXISTS organization_name VARCHAR(250)
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter user details table to add contact person name: %v", err))
	}
	fmt.Println("user details table with contact person name column altered successfully.")
}

func AddCoachNameColumn() {
	query := ` ALTER TABLE user_details
						ADD COLUMN IF NOT EXISTS coach_name VARCHAR(250)
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to alter user details table to add coach name: %v", err))
	}
	fmt.Println("user details table with coach name column altered succeessfully.")
}

func ChangeContactPersonNameFieldToOrganizationName() {
	query := `
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'user_details'
				AND column_name = 'contact_person_name'
			) THEN
				ALTER TABLE user_details
				RENAME COLUMN contact_person_name TO organization_name;
			END IF;
		END
		$$;
	`

	_, err := database.DB.Exec(query)
	if err != nil {
		log.Fatalf("Failed to alter user_details table: %v", err)
	}
	fmt.Println("user_details table altered successfully.")
}

func DropCurrentTeamColumn() {
	query := `
		ALTER TABLE user_details
		DROP COLUMN IF EXISTS current_team
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to drop current_team column: %v", err))
	}
	fmt.Println("successfully dropped current_team column")
}

func SyncStatusWithUserTable() {
	query := `
		UPDATE user_details
		SET status= u.status
		FROM users u
		where u.id= user_details.user_id
	`
	_, err := database.DB.Exec(query)
	if err != nil {
		panic(fmt.Sprintf("Failed to update user_details table: %v", err))
	}
	fmt.Println("successfully updated user_details table")
}
