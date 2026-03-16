package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sports-events-api/database"
	"time"
)

type ModData struct {
	Name    string `json:"name"`
	Id      int64  `json:"id"`
	EventId int64  `json:"event_id"`
	Status  string `json:"status"`
}

// GetAllModeratorsForOrg retrieves all moderators linked to a given organization.
// This function performs the following:
//  1. Executes a SQL query that fetches all moderator IDs and names associated with the organization.
//  2. Aggregates the result in JSON format.
//  3. Parses the JSON result into a slice of ModData.
//
// Params:
//   - organizationId (int64): The ID of the organization whose moderators are to be retrieved.
//
// Returns:
//   - []ModData: A slice containing the moderators’ information.
//   - error: If any error occurs during the query or JSON unmarshalling.
func GetAllModeratorsForOrg(organizationId int64) ([]ModData, error) {
	var Result []ModData
	var resultJson sql.NullString

	query := `
	SELECT json_agg(result)
	FROM (
	  SELECT
	    mod.id AS id,
	    mod.name AS name,
		osm.event_id AS event_id,
		osm.status
	  FROM organization_has_score_moderator osm
	  JOIN users org ON osm.organization_id = org.id
	  JOIN users mod ON osm.moderator_id = mod.id
	  WHERE org.id = $1
	  AND mod.status='Active'
	  AND org.status='Active'
	  AND osm.status!='Delete'
	) AS result;
	`

	err := database.DB.QueryRow(query, organizationId).Scan(&resultJson)
	if err != nil {
		return Result, err
	}

	if resultJson.Valid {
		// Only unmarshal if the string is not empty
		if len(resultJson.String) > 0 {
			err = json.Unmarshal([]byte(resultJson.String), &Result)
			if err != nil {
				return Result, err
			}
		} else {
			// Empty JSON string is considered valid but unmarshal would fail
			Result = []ModData{}
		}
	} else {
		// Null JSON from DB
		Result = []ModData{}
	}
	return Result, nil
}

// CheckOrganizationHasModerator checks if a moderator is already assigned to an organization.
// Additionally, it ensures both the moderator and organization exist and are active.
//
// Params:
//   - OrganizationId (int64): ID of the organization.
//   - ModeratorId (int64): ID of the moderator.
//   - tx (*sql.Tx): Active transaction context.
//
// Returns:
//   - bool: True if the relationship exists.
//   - error: If any query fails or if either user does not exist or is inactive.
func CheckOrganizationHasModerator(OrganizationId int64, ModeratorId int64, EventId int64, tx *sql.Tx) (bool, error) {
	OrganizationHasModerator := false
	var query string
	var err error

	query = `SELECT EXISTS (
		SELECT 1 FROM organization_has_score_moderator WHERE moderator_id=$1 AND organization_id=$2 AND event_id=$3 AND status!='Delete')`
	err = tx.QueryRow(query, ModeratorId, OrganizationId, EventId).Scan(&OrganizationHasModerator)
	if err != nil {
		tx.Rollback()
		return OrganizationHasModerator, fmt.Errorf("database error for boolean ---> %v", err)
	}

	// Check that the moderator exists and is active
	var moderatorExists bool
	query = `SELECT EXISTS (
		SELECT 1 FROM users WHERE id=$1 AND status = 'Active'
	)`
	err = tx.QueryRow(query, ModeratorId).Scan(&moderatorExists)
	if err != nil {
		tx.Rollback()
		return OrganizationHasModerator, fmt.Errorf("database error for moderator ---> %v", err)
	}
	if !moderatorExists {
		tx.Rollback()
		return OrganizationHasModerator, fmt.Errorf("moderator with id '%v' does not exists in the system", ModeratorId)
	}

	// Check that the organizer exists and is active
	var organizerExists bool
	query = `SELECT EXISTS (
		SELECT 1 FROM users WHERE id=$1 AND status = 'Active'
		)`
	// SELECT 1 FROM users WHERE id=$1 AND status = 'Active' AND role_slug='organization'
	err = tx.QueryRow(query, OrganizationId).Scan(&organizerExists)
	if err != nil {
		tx.Rollback()
		return OrganizationHasModerator, fmt.Errorf("database error for organization---> %v", err)
	}
	if !organizerExists {
		tx.Rollback()
		return OrganizationHasModerator, fmt.Errorf("organization with id '%v' does not exists in the system", OrganizationId)
	}

	// Check that the organizer exists and is active and belongs to the organizer
	var eventExists bool
	query = `SELECT EXISTS (
		SELECT 1 FROM events WHERE id=$1 AND created_by_id=$2 AND status = 'Active'
	)`
	err = tx.QueryRow(query, EventId, OrganizationId).Scan(&eventExists)
	if err != nil {
		tx.Rollback()
		return OrganizationHasModerator, fmt.Errorf("database error for event---> %v", err)
	}
	if !eventExists {
		tx.Rollback()
		return OrganizationHasModerator, fmt.Errorf("event with id '%v' does not exists in the system", EventId)
	}

	return OrganizationHasModerator, nil
}

// AddModerator adds a moderator to an organization after checking constraints.
// It validates that the moderator is not already linked, then inserts the relation.
//
// Params:
//   - OrganizationId (int64): ID of the organization.
//   - ModeratorId (int64): ID of the moderator.
//
// Returns:
//   - error: If validation fails or insertion into the DB fails.
func AddModerator(OrganizationId int64, ModeratorId int64, EventId int64) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("error generating a transaction request --->%v", err)
	}
	UpdatedAt := time.Now() // Use current time if UpdatedAt is not set
	CreatedAt := time.Now() // Use current time if CreatedAt is not set

	query := `
	INSERT INTO organization_has_score_moderator (organization_id, moderator_id, event_id,created_at, updated_at) VALUES($1,$2,$3,$4,$5)
	`

	_, err = tx.Exec(query, OrganizationId, ModeratorId, EventId, CreatedAt, UpdatedAt)
	if err != nil {
		return fmt.Errorf("insertion error ---> %v", err)
	}
	tx.Commit()

	return nil
}

// UpdateModerator updates or deletes the moderator relationship for an organization.
// Supported actions:
//   - "delete": sets the relationship status to Delete.
//   - "update": toggles the status between Active and Inactive.
//
// Params:
//   - OrganizationId (int64): ID of the organization.
//   - ModeratorId (int64): ID of the moderator.
//   - action (string): Action to perform ("delete" or "update").
//
// Returns:
//   - error: If any database operation fails.
func UpdateModerator(OrganizationId int64, ModeratorId int64, EventId int64, action string) error {
	if action == "delete" {
		query := `UPDATE organization_has_score_moderator
		          SET status = CASE
		                         WHEN status != 'Delete' THEN 'Delete'
								 ELSE status
							   END
		          WHERE moderator_id = $1 AND organization_id = $2 AND event_id= $3;`

		_, err := database.DB.Exec(query, ModeratorId, OrganizationId, EventId)
		if err != nil {
			return fmt.Errorf("database error while deleting --->%v", err)
		}
	} else if action == "update" {
		query := `UPDATE organization_has_score_moderator
		          SET status = CASE
		                         WHEN status = 'Active' THEN 'Inactive'
		                         WHEN status = 'Inactive' THEN 'Active'
		                         ELSE status
		                       END
		          WHERE moderator_id = $1 AND organization_id = $2 AND event_id= $3;`
		_, err := database.DB.Exec(query, ModeratorId, OrganizationId, EventId)
		if err != nil {
			return fmt.Errorf("database error while updating status --->%v", err)
		}
	}
	return nil
}

type ShortEventData struct {
	Name string `json:"name"`
	Id   int64  `json:"id"`
}
type EncData struct {
	Name  string `json:"name"`
	EncId string `json:"id"`
}

func GetEventByOrganizationId(OrganizationId int64) ([]ShortEventData, error) {
	var Result []ShortEventData
	var resultJson sql.NullString

	query := `
	SELECT json_agg(result)
	FROM (
	  SELECT
	    id AS id,
	    name AS name
	  FROM events WHERE created_by_id=$1 AND status='Active'
	) AS result;
	`

	err := database.DB.QueryRow(query, OrganizationId).Scan(&resultJson)
	if err != nil {
		return Result, err
	}

	if resultJson.Valid {
		// Only unmarshal if the string is not empty
		if len(resultJson.String) > 0 {
			err = json.Unmarshal([]byte(resultJson.String), &Result)
			if err != nil {
				return Result, err
			}
		} else {
			// Empty JSON string is considered valid but unmarshal would fail
			Result = []ShortEventData{}
		}
	} else {
		// Null JSON from DB
		Result = []ShortEventData{}
	}
	return Result, nil
}
