package models

import (
	"database/sql"
	"fmt"
	"log"
	"sports-events-api/database"
	"strings"
	"time"
)

type LevelOfCompetition struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LevelConfigRequest struct {
	ID        int64     `json:"-"`
	EncID     string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Insert function
func InsertLevelOfCompetition(lc *LevelOfCompetition) (*LevelOfCompetition, error) {
	var existingID int64
	checkQuery := `SELECT id FROM level_of_competitions WHERE title = $1 AND status IN ('Active', 'Inactive')`
	err := database.DB.QueryRow(checkQuery, lc.Title).Scan(&existingID)

	// If a game type with the same name exists, return an error
	if err == nil {
		return nil, fmt.Errorf("level of competiton with the same name already exists")
	} else if err != sql.ErrNoRows {
		fmt.Printf("Error checking existing level of competition: %v\n", err)
		return nil, fmt.Errorf("error checking existing level of competition: %v", err)
	}

	query := `
		INSERT INTO level_of_competitions (title, status)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at;
	`

	err = database.DB.QueryRow(query, lc.Title, lc.Status).Scan(&lc.ID, &lc.CreatedAt, &lc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return lc, nil
}

// view level of competition
func GetLevelsOfCompetition(search, sort, dir, status string, limit, offset int64) (int64, []LevelOfCompetition, error) {
	var (
		levels []LevelOfCompetition
		total  int64
	)

	baseQuery := `FROM level_of_competitions WHERE 1=1 AND status IN ('Active', 'Inactive')`
	var filters []string
	var args []interface{}
	argIndex := 1

	// Search filter
	if search != "" {
		filters = append(filters, fmt.Sprintf("LOWER(title) LIKE $%d", argIndex))
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIndex++
	}

	// Status filter
	if status != "" {
		filters = append(filters, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	// Add filters to base query
	if len(filters) > 0 {
		baseQuery += " AND " + strings.Join(filters, " AND ")
	}

	// Count query
	countQuery := `SELECT COUNT(*) ` + baseQuery
	if err := database.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return 0, nil, err
	}

	// Sorting and pagination
	if sort == "" {
		sort = "created_at"
	}
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	// Add LIMIT and OFFSET placeholders
	dataQuery := `SELECT id, title, status, created_at, updated_at ` + baseQuery +
		fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sort, dir, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(dataQuery, args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var level LevelOfCompetition
		if err := rows.Scan(&level.ID, &level.Title, &level.Status, &level.CreatedAt, &level.UpdatedAt); err != nil {
			return 0, nil, err
		}
		levels = append(levels, level)
	}

	return total, levels, nil
}

// view level of competition by id
func GetLevelofCompetitionById(id int64) (*LevelOfCompetition, error) {
	// Query to fetch the level of competition details by ID
	query := `SELECT id, title, status, created_at, updated_at FROM level_of_competitions WHERE id=$1`

	var LevelOfCompetition LevelOfCompetition
	err := database.DB.QueryRow(query, id).Scan(
		&LevelOfCompetition.ID,
		&LevelOfCompetition.Title,
		// &LevelOfCompetition.Slug,
		&LevelOfCompetition.Status,
		&LevelOfCompetition.CreatedAt,
		&LevelOfCompetition.UpdatedAt,
	)

	// Check if the level of competition is marked as "Delete", return an error if so
	if LevelOfCompetition.Status == "Delete" {
		return nil, fmt.Errorf("cannot show the data of the deleted level of competition")
	}

	// Handle cases where no record was found for the provided ID
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("level of competition is not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching level of competition : %v", err)
	}

	// Return the fetched level of competition
	return &LevelOfCompetition, nil
}

// delete level of competition
func DeleteLevelofcompetitionByID(id int64) (*LevelOfCompetition, error) {
	// Check if the record exists in the database
	checkQuery := `SELECT id, status FROM level_of_competitions WHERE id = $1`
	var LevelOfCompetition LevelOfCompetition

	err := database.DB.QueryRow(checkQuery, id).Scan(&LevelOfCompetition.ID, &LevelOfCompetition.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("level of competition not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching level of competition: %v", err)
	}

	// If the level of competition is already marked as "Delete", return an error
	if LevelOfCompetition.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Mark the level of competition as deleted by updating its status
	deleteQuery := `UPDATE level_of_competitions SET status = 'Delete', updated_at = $1 WHERE id = $2`
	_, err = database.DB.Exec(deleteQuery, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("error deleting level of competition: %v", err)
	}

	// Update the status to "Delete" and return the updated level of competition
	LevelOfCompetition.Status = "Delete"
	LevelOfCompetition.UpdatedAt = time.Now()

	return &LevelOfCompetition, nil
}

// update level of competition
func UpdateLevelOfCompetition(data *LevelOfCompetition) (*LevelOfCompetition, error) {
	// Step 1: Check if the record exists
	checkQuery := `SELECT id, status FROM level_of_competitions WHERE id = $1`
	var existingID int
	var existingStatus string

	err := database.DB.QueryRow(checkQuery, data.ID).Scan(&existingID, &existingStatus)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("level of competition does not exist")
	} else if err != nil {
		return nil, fmt.Errorf("database error while checking level: %v", err)
	}

	if existingStatus == "Delete" {
		return nil, fmt.Errorf("cannot update deleted level of competition")
	}

	// Step 2: Check for duplicate title (excluding current record)
	var duplicateID int
	duplicateCheckQuery := `SELECT id FROM level_of_competitions WHERE LOWER(title) = LOWER($1) AND status IN ('Active', 'Inactive') AND id != $2`
	err = database.DB.QueryRow(duplicateCheckQuery, data.Title, data.ID).Scan(&duplicateID)

	if err == nil {
		return nil, fmt.Errorf("level of competition with the same title already exists")
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error while checking duplicate title: %v", err)
	}

	// Step 3: Proceed with the update
	data.UpdatedAt = time.Now()

	updateQuery := `
        UPDATE level_of_competitions
        SET title = $1, updated_at = $2
        WHERE id = $3
        RETURNING id, title, status, created_at, updated_at`

	var updated LevelOfCompetition
	err = database.DB.QueryRow(updateQuery, data.Title, data.UpdatedAt, data.ID).Scan(
		&updated.ID,
		&updated.Title,
		&updated.Status,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("database error during update: %v", err)
	}

	return &updated, nil
}

// update status level of competition
func UpdateLevelofCompetitionStatusByID(LevelOfCompetitionID int64, status string) error {
	// Prepare the SQL query to update the status of the game type
	query := `UPDATE level_of_competitions SET status = $1 WHERE id = $2`

	// Execute the query
	_, err := database.DB.Exec(query, status, LevelOfCompetitionID)
	if err != nil {
		log.Printf("Error updating status. err :  %v\n", err)
		return fmt.Errorf("failed to update status")
	}

	return nil
}

func GetLevelofCompetitionStatusByID(LevelOfCompetitionID int64) (string, error) {
	var status string
	query := `SELECT status FROM level_of_competitions WHERE id = $1`
	err := database.DB.QueryRow(query, LevelOfCompetitionID).Scan(&status)
	if err != nil {
		log.Printf("Error fetching status for title ID %d: %v\n", LevelOfCompetitionID, err)
		return "", fmt.Errorf("failed to fetch status")
	}
	return status, nil
}

// config level of competition
func GetAllLevelOfCompetition() ([]LevelOfCompetition, error) {
	query := `SELECT id, title, status, created_at, updated_at FROM level_of_competitions WHERE status = 'Active'`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var levels []LevelOfCompetition
	for rows.Next() {
		var l LevelOfCompetition
		err := rows.Scan(&l.ID, &l.Title, &l.Status, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			return nil, err
		}
		levels = append(levels, l)
	}

	return levels, nil
}
