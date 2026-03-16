package models

import (
	"database/sql"
	"fmt"
	"log"
	"sports-events-api/database"
	"strings"
	"time"

	"github.com/gosimple/slug"
)

// Games_Types represents the structure of a game type entity.
// This struct contains fields for the game's details such as its ID, name, slug, status, and timestamps.
type Games_Types struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name,omitempty" validate:"omitempty,min=2"` // The name of the game type (min length: 2 characters)
	Slug      string    `json:"slug,omitempty" validate:"omitempty,min=2"` // Slug version of the game name (min length: 2 characters)
	Status    string    `json:"status,omitempty"`                          // The current status of the game type (e.g., Active, Inactive)
	CreatedAt time.Time `json:"created_at,omitempty"`                      // The timestamp of when the game type was created
	UpdatedAt time.Time `json:"updated_at,omitempty"`                      // The timestamp of when the game type was last updated
}

// AddGame represents the data required to add a new game type.
// It only includes the name of the game type.
type AddGame struct {
	Name string `json:"name"` // The name of the new game type to be added
}

// InsertGamesTypes inserts a new game type into the database.
// This function performs the following:
//  1. Checks if the game type already exists using its slug (name-based identifier).
//  2. If the game type does not exist, it generates a new slug for the game and sets the initial status to "Active".
//  3. Inserts the game type into the database and returns the created game type with its generated ID.
//  4. Returns an error if a game type with the same name already exists or if any database query fails.
func InsertGamesTypes(games_types *Games_Types) (*Games_Types, error) {
	// Set creation and update timestamps if they are not set
	if games_types.CreatedAt.IsZero() {
		games_types.CreatedAt = time.Now()
	}
	if games_types.UpdatedAt.IsZero() {
		games_types.UpdatedAt = time.Now()
	}

	// Generate a slug from the name of the game type
	games_types.Slug = slug.Make(games_types.Name)
	games_types.Status = "Active" // Set the default status to "Active"

	// Check if a game type with the same slug already exists
	var existingID int64
	checkQuery := `SELECT id FROM games_types WHERE slug = $1 AND status IN ('Active', 'Inactive')`
	err := database.DB.QueryRow(checkQuery, games_types.Slug).Scan(&existingID)

	// If a game type with the same name exists, return an error
	if err == nil {
		return nil, fmt.Errorf("game type with the same name already exists")
	} else if err != sql.ErrNoRows {
		fmt.Printf("Error checking existing game types: %v\n", err)
		return nil, fmt.Errorf("error checking existing game type: %v", err)
	}

	// Insert the new game type into the database
	var games_typesID int64
	query := `INSERT INTO games_types(name, slug, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err = database.DB.QueryRow(query, games_types.Name, games_types.Slug, games_types.Status, games_types.CreatedAt, games_types.UpdatedAt).Scan(&games_typesID)

	// Return an error if the insertion fails
	if err != nil {
		fmt.Printf("Error During Database Query : %v\n", err)
		return nil, fmt.Errorf("unable to create game type : %v", err)
	}

	// Set the generated ID for the new game type and return it
	games_types.ID = games_typesID
	return games_types, nil
}

// GetGamesTypesById retrieves a game type by its ID from the database.
// This function performs the following:
//  1. Executes a query to fetch the game type's details (ID, name, slug, status, timestamps).
//  2. Checks if the fetched game type is marked as "Delete" (inactive), and returns an error if so.
//  3. Returns the game type if found, or an error if the game type is not found or there is a database issue.
func GetGamesTypesById(id int64) (*Games_Types, error) {
	// Query to fetch the game type details by ID
	query := `SELECT id, name, slug, status, created_at, updated_at FROM games_types WHERE id=$1`

	var gamesTypes Games_Types
	err := database.DB.QueryRow(query, id).Scan(
		&gamesTypes.ID,
		&gamesTypes.Name,
		&gamesTypes.Slug,
		&gamesTypes.Status,
		&gamesTypes.CreatedAt,
		&gamesTypes.UpdatedAt,
	)

	// Check if the game type is marked as "Delete", return an error if so
	if gamesTypes.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Handle cases where no record was found for the provided ID
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("game type is not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching game types : %v", err)
	}

	// Return the fetched game type
	return &gamesTypes, nil
}

// DeleteGamesTypesByID deletes a game type from the database by marking its status as "Delete".
// This function performs the following:
//  1. Checks if the game type exists by querying its ID and status.
//  2. If the game type exists and is not already marked as "Delete", it updates the status to "Delete" and updates the timestamp.
//  3. Returns the updated game type or an error if the game type does not exist or there is an issue with the deletion process.
func DeleteGamesTypesByID(id int64) (*Games_Types, error) {
	// Check if the record exists in the database
	checkQuery := `SELECT id, status FROM games_types WHERE id = $1`
	var gamesType Games_Types

	err := database.DB.QueryRow(checkQuery, id).Scan(&gamesType.ID, &gamesType.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("game type not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching game type: %v", err)
	}

	// If the game type is already marked as "Delete", return an error
	if gamesType.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Mark the game type as deleted by updating its status
	deleteQuery := `UPDATE games_types SET status = 'Delete', updated_at = $1 WHERE id = $2`
	_, err = database.DB.Exec(deleteQuery, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("error deleting game type: %v", err)
	}

	// Update the status to "Delete" and return the updated game type
	gamesType.Status = "Delete"
	gamesType.UpdatedAt = time.Now()

	return &gamesType, nil
}

// GetGamesTypes retrieves a list of game types with filtering, sorting, and pagination.
// This function performs the following:
//  1. Filters game types based on the provided status (Active or Inactive),
//     and allows for additional filtering by name or search term.
//  2. Adds sorting based on the specified column and direction.
//  3. Implements pagination by applying limit and offset to the query.
//  4. Executes the query and parses the results into a slice of Games_Types.
//  5. Returns the total count of records along with the slice of game types.
func GetGamesTypes(search, sort, dir, status string, limit, offset int64) (int, []Games_Types, error) {
	var gamesTypes []Games_Types
	args := []interface{}{limit, offset}
	query := `
        SELECT
            id, name, slug, status, created_at, updated_at, COUNT(id) OVER() AS totalrecords
        FROM
            games_types
        WHERE status IN ('Active', 'Inactive')` // Only fetch Active and Inactive statuses

	// Add additional status filtering if provided
	if status != "" {
		statusValues := strings.Split(status, ",")
		statusPlaceholders := []string{}
		for _, s := range statusValues {
			statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, strings.TrimSpace(s))
		}
		query += fmt.Sprintf(" AND status IN (%s)", strings.Join(statusPlaceholders, ", "))
	}

	// Add search functionality
	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d)", len(args)+1)
		args = append(args, "%"+search+"%")
	}

	// Add sorting and pagination
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $1 OFFSET $2", sort, dir)

	// Execute query
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying games_types: %v\n", err)
		return 0, nil, err
	}
	defer rows.Close()

	// Parse query results
	totalRecords := 0
	for rows.Next() {
		var gamesType Games_Types
		if err := rows.Scan(
			&gamesType.ID,
			&gamesType.Name,
			&gamesType.Slug,
			&gamesType.Status,
			&gamesType.CreatedAt,
			&gamesType.UpdatedAt,
			&totalRecords,
		); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			return 0, nil, err
		}
		gamesTypes = append(gamesTypes, gamesType)
	}
	return totalRecords, gamesTypes, nil
}

// UpdateGamesTypes updates a specific game type's details (name and slug).
// This function performs the following:
//  1. Checks if the game type exists in the database.
//  2. Verifies that the game type is not marked as "Delete" before allowing an update.
//  3. Generates a new slug based on the updated name, ensuring no duplicate slugs exist for active or inactive game types.
//  4. Updates the game type's name, slug, and updated timestamp.
//  5. Returns the updated game type or an error if any issues occur.
func UpdateGamesTypes(games_types *Games_Types) (*Games_Types, error) {
	// Check if the record exists
	checkQuery := `SELECT id, status FROM games_types WHERE id = $1`
	var existingID int64
	var existingStatus string

	err := database.DB.QueryRow(checkQuery, games_types.ID).Scan(&existingID, &existingStatus)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("game type does not exist")
	} else if err != nil {
		return nil, fmt.Errorf("database error while checking game type: %v", err)
	}

	if existingStatus == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Generate the new slug based on the updated name
	newSlug := slug.Make(games_types.Name)

	// Check if a game type with the same slug already exists and is Active/Inactive (excluding the current one)
	var duplicateID int64
	duplicateCheckQuery := `SELECT id FROM games_types WHERE slug = $1 AND status IN ('Active', 'Inactive') AND id != $2`
	err = database.DB.QueryRow(duplicateCheckQuery, newSlug, games_types.ID).Scan(&duplicateID)

	if err == nil {
		// If no error, it means a record with the same slug already exists
		return nil, fmt.Errorf("game type with the same name already exists")
	} else if err != sql.ErrNoRows {
		// If an error occurred that's not "no rows", return the error
		return nil, fmt.Errorf("database error while checking duplicate game type: %v", err)
	}

	// Proceed with the update
	games_types.UpdatedAt = time.Now()
	games_types.Slug = newSlug

	updateQuery := `
        UPDATE games_types 
        SET name = $1, slug = $2, updated_at = $3 
        WHERE id = $4
        RETURNING id, name, slug, status, created_at, updated_at`

	var updatedGameType Games_Types
	err = database.DB.QueryRow(updateQuery, games_types.Name, games_types.Slug, games_types.UpdatedAt, games_types.ID).Scan(
		&updatedGameType.ID,
		&updatedGameType.Name,
		&updatedGameType.Slug,
		&updatedGameType.Status,
		&updatedGameType.CreatedAt,
		&updatedGameType.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("database error during update: %v", err)
	}

	return &updatedGameType, nil
}

// UpdateGameTypeStatusByID updates the status of a game type based on its ID.
// This function performs the following:
//  1. Executes an SQL query to update the status field of the game type with the given ID.
//  2. Returns an error if the update fails or if there's a problem with the query execution.
func UpdateGameTypeStatusByID(gameTypeID int64, status string) error {
	// Prepare the SQL query to update the status of the game type
	query := `UPDATE games_types SET status = $1 WHERE id = $2`

	// Execute the query
	_, err := database.DB.Exec(query, status, gameTypeID)
	if err != nil {
		log.Printf("Error updating status. err :  %v\n", err)
		return fmt.Errorf("failed to update status")
	}

	return nil
}

// GetGameTypeStatusByID retrieves the current status of a game type based on its ID.
// This function performs the following:
//  1. Executes an SQL query to fetch the status of the game type.
//  2. Returns the status of the game type or an error if there's a problem with the query.
func GetGameTypeStatusByID(gameTypeID int64) (string, error) {
	var status string
	query := `SELECT status FROM games_types WHERE id = $1`
	err := database.DB.QueryRow(query, gameTypeID).Scan(&status)
	if err != nil {
		log.Printf("Error fetching status for game type ID %d: %v\n", gameTypeID, err)
		return "", fmt.Errorf("failed to fetch status")
	}
	return status, nil
}
