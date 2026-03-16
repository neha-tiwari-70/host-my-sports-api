package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sports-events-api/database"
	"sports-events-api/utils"
	"strings"
	"time"
)

// InsertTeamMembers adds each member of a team into the event_has_users table,
// after validating that they are not already registered for the same game in the event.
//
// This function performs the following:
// 1. Iterates over each member ID provided.
// 2. Checks if the user is already registered in the same event and game (to avoid duplicates).
// 3. Inserts the member as a participant in the event for the given team and game.
//
// Params:
//   - MemberIds ([]int64): Slice of user IDs to be added to the team.
//   - EventId (int64): ID of the event.
//   - GameId (int64): ID of the game.
//   - TeamId (int64): ID of the team.
//   - tx (*sql.Tx): Active database transaction to ensure atomicity.
//
// Returns:
//   - error: If a member is already registered or an insertion fails.
//
// func InsertTeamMembers(MemberIds []int64, EventId int64, GameId int64, TeamId int64, EventHasGameTypeId int64, tx *sql.Tx) error {
func InsertTeamMembers(MemberIds []int64, TshirtSizes map[int64]string, EventId int64, GameId int64, TeamId int64, EventHasGameTypeId int64, tx *sql.Tx) error {
	for _, MemberId := range MemberIds {
		var age int
		// check if player is already registered in the same game/event/team
		PlayerAlreadyRegistered, err := CheckParticipationInGameType(MemberId, EventId, GameId, TeamId, tx)

		if err != nil {
			tx.Rollback()
			return err
		} else if PlayerAlreadyRegistered {
			tx.Rollback()
			return fmt.Errorf("one of the player in the team is already registered for this game in this event: %v", MemberId)
		}

		user, err := GetUserByID(int(MemberId))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error fetching user--> %v", err)
		}
		user.Details, err = GetUserDetails(MemberId)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error fetching details--> %v", err)
		}

		if user.RoleSlug == "organization" {
			tx.Rollback()
			return fmt.Errorf("organization detected in team lineup; only individuals are allowed")
		}

		if user.Details.DOB != nil {
			// datestring := fmt.Sprintf("%d-%02d-%02d", dob.Time.Year(), int(dob.Time.Month()), dob.Time.Day())
			datestring, err := time.Parse(time.RFC3339, *user.Details.DOB)
			if err != nil {
				tx.Rollback()
				return err
			}
			age, err = utils.CalculateAge(fmt.Sprintf("%d-%02d-%02d", datestring.Year(), int(datestring.Month()), datestring.Day()))
			// fmt.Println("age:", age, "error:", err)
			if err != nil {
				tx.Rollback()
				return err
			}
			if age == 0 {
				tx.Rollback()
				return fmt.Errorf("age calculation error") // In case age calculation returns 0
			}
		}
		isValid, _, err := ValidateAgeAndGender(EventHasGameTypeId, age, *user.Details.Gender)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error validating age--> %v", err)
		}

		if !isValid {
			tx.Rollback()
			return fmt.Errorf("one of the users do not satisfy the age requirements")
		}
		var TshirtSizeArg sql.NullString
		if TshirtSizes[MemberId] != "" {
			TshirtSizeArg.String = strings.ToUpper(TshirtSizes[MemberId])
			TshirtSizeArg.Valid = true
		}

		// insert entry into DB
		// query := `INSERT INTO event_has_users (game_id, event_id, user_id, event_has_team_id)
		//           VALUES($1, $2, $3, $4)`
		// _, err = tx.Exec(query, GameId, EventId, MemberId, TeamId)
		query := `INSERT INTO event_has_users (game_id, event_id, user_id, event_has_team_id, tshirt_size)
          VALUES($1, $2, $3, $4, $5)`
		_, err = tx.Exec(query, GameId, EventId, MemberId, TeamId, TshirtSizeArg)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting user in event_has_users --> %v", err)
		}
	}
	return nil
}

// LogoUpdate updates the logo path of a team after removing the old logo file from storage (if any).
//
// This function performs the following:
// 1. Ensures the target folder exists or creates it.
// 2. Retrieves the previous logo path for the team.
// 3. Deletes the old logo file from the filesystem (if it exists).
// 4. Updates the database with the new logo path.
//
// Params:
//   - Path (string): New logo file path to set.
//   - TeamId (int64): ID of the team whose logo is being updated.
//   - tx (*sql.Tx): Active database transaction for consistency.
//
// Returns:
//   - string: The path that was set (same as input).
//   - error: If any filesystem or database operation fails.
func LogoUpdate(Path string, TeamId int64, tx *sql.Tx) (string, error) {
	// ensure logo folder exists
	_, err := utils.CreateFolder(FolderPath)
	if err != nil {
		tx.Rollback()
		return Path, fmt.Errorf("error creating folder -> %v", err)
	}

	// fetch old logo path and delete the file if it exists
	oldPath, err := GetTeamLogoById(TeamId)
	if err != nil {
		tx.Rollback()
		return Path, fmt.Errorf("failed fetch to Former Image %v", err)
	}
	if oldPath != "" {
		Path := filepath.Join(FolderPath, filepath.Base(oldPath))
		_ = os.Remove(Path) // ignore error, we proceed even if the file doesn't exist
		fmt.Println("\nDeleted:", oldPath)
	}

	// update DB with new path
	query := `
	    UPDATE event_has_teams SET
	        team_logo_path = $1
	    WHERE id = $2;
	`
	_, err = tx.Exec(query, Path, TeamId)
	if err != nil {
		tx.Rollback()
		return Path, err
	}
	return Path, nil
}

// func removeLogo(TeamId int64) error {
// 	oldPath, err := GetTeamLogoById(TeamId)
// 	if err != nil {
// 		return fmt.Errorf("failed fetch Former Image-> %v", err)
// 	}
// 	if oldPath != "" {
// 		Path := filepath.Join(FolderPath, filepath.Base(oldPath))
// 		_ = os.Remove(Path)
// 	}
// 	query := `
// 	    UPDATE event_has_teams SET
// 	        team_logo_path = ""
// 	    WHERE id = $1;
// 	`
// 	_, err = database.DB.Exec(query, TeamId)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// GetTeamLogoById retrieves the logo file path associated with a team.
//
// This function performs the following:
// 1. Executes a SELECT query to get the team_logo_path from event_has_teams table using team ID.
// 2. Returns an empty string if no record is found.
// 3. Returns the path as a string if available.
//
// Params:
//   - TeamId (int64): The ID of the team whose logo path is to be fetched.
//
// Returns:
//   - string: Path to the logo (can be empty if not found).
//   - error: If any error occurs during query execution.
func GetTeamLogoById(TeamId int64) (string, error) {
	var LogoPath any
	err := database.DB.QueryRow("SELECT team_logo_path FROM event_has_teams WHERE id = $1", TeamId).Scan(&LogoPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to fetch logo_image: %v", err)
	}
	return fmt.Sprintf("%v", LogoPath), nil
}

// GetTeams fetches a paginated list of teams filtered by search parameters like event, game, status, etc.
//
// This function performs the following:
// 1. Builds a dynamic SQL query based on optional filters like event ID, game ID, game type ID, and status.
// 2. Supports search on team name and sorts by specified field and direction.
// 3. Executes the query with pagination support and returns total record count and team list.
//
// Params:
//   - search (string): Keyword to filter team names (ILIKE).
//   - sort (string): Column name to sort results.
//   - dir (string): Sort direction ("ASC" or "DESC").
//   - status (string): Comma-separated status values to filter.
//   - event_id (int64): Event ID filter.
//   - game_id (int64): Game ID filter.
//   - game_type_id (int64): Game Type ID filter.
//   - limit (int64): Number of records per page.
//   - offset (int64): Offset for pagination.
//
// Returns:
//   - int: Total number of matching records (for pagination).
//   - []Teams: List of matching teams.
//   - error: If any error occurs during query or data processing.
func GetTeams(search, sort, dir, status string, event_id, game_id, game_type_id, age_group_id, limit, offset int64) (int, []Teams, error) {
	var teams []Teams
	args := []interface{}{limit, offset}
	query := `
        SELECT
            eht.id, eht.team_name, eht.team_captain, u.name AS team_captain_name, eht.team_logo_path, eht.slug, eht.status, eht.created_at, eht.group_no, eht.updated_at, COUNT(eht.id) OVER() AS totalrecords
        FROM
            event_has_teams eht
        JOIN
            users u ON eht.team_captain = u.id
        WHERE eht.status IN ('Active', 'Inactive')` // Only fetch Active and Inactive statuses

	// Add additional status filtering if provided
	if status != "" {
		statusValues := strings.Split(status, ",")
		statusPlaceholders := []string{}
		for _, s := range statusValues {
			statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, strings.TrimSpace(s))
		}
		query += fmt.Sprintf(" AND eht.status IN (%s)", strings.Join(statusPlaceholders, ", "))
	}

	// Add filters dynamically
	// Add event_id filter
	if event_id != 0 {
		query += fmt.Sprintf(" AND eht.event_id = $%d", len(args)+1)
		args = append(args, event_id)
	}

	// Add game_id filter
	if game_id != 0 {
		query += fmt.Sprintf(" AND eht.game_id = $%d", len(args)+1)
		args = append(args, game_id)
	}

	// Add game_type_id filter
	if game_type_id != 0 {
		query += fmt.Sprintf(" AND eht.game_type_id = $%d", len(args)+1)
		args = append(args, game_type_id)
	}

	if age_group_id != 0 {
		query += fmt.Sprintf(" AND eht.age_group_id = $%d", len(args)+1)
		args = append(args, age_group_id)
	}

	// Add search functionality
	if search != "" {
		query += fmt.Sprintf(" AND (eht.name ILIKE $%d)", len(args)+1)
		args = append(args, "%"+search+"%")
	}

	// Add sorting and pagination
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $1 OFFSET $2", sort, dir)

	// Execute query
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying teams: %v\n", err)
		return 0, nil, err
	}
	defer rows.Close()

	// Parse query results
	totalRecords := 0
	for rows.Next() {
		var logo sql.NullString
		var Team Teams
		if err := rows.Scan(
			&Team.TeamId,
			&Team.TeamName,
			&Team.TeamCaptainID,
			&Team.TeamCaptain,
			&logo,
			&Team.Slug,
			&Team.Status,
			&Team.CreatedAt,
			&Team.GroupNo,
			&Team.UpdatedAt,
			&totalRecords,
		); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			return 0, nil, err
		}

		if logo.Valid {
			Team.TeamLogoPath = logo.String
		} else {
			Team.TeamLogoPath = "public/uploads/static/staticLogo.png"
		}
		teams = append(teams, Team)
	}

	return totalRecords, teams, nil
}

// GetTeamPlayer checks if a specific user is a member of a team and returns their user code.
//
// This function performs the following:
// 1. Executes a JOIN query across users, event_has_users, and event_has_teams to find a match.
// 2. Returns the user_code if found, along with a boolean indicating membership.
//
// Params:
//   - TeamId (int64): The team ID to check membership against.
//   - UserId (int64): The user ID to verify.
//
// Returns:
//   - string: The user_code of the player (empty if not found).
//   - bool: Whether the user is part of the specified team.
//   - error: If any database error occurs.
func GetTeamPlayer(TeamId int64, UserId int64) (string, bool, error) {
	query := `
		SELECT (u.user_code)
		FROM users u
		JOIN event_has_users ehu ON ehu.user_id = u.id
		JOIN event_has_teams eht ON ehu.event_has_team_id = eht.id
		WHERE eht.id = $1 AND u.id = $2
	`
	var UserCode string

	err := database.DB.QueryRow(query, TeamId, UserId).Scan(&UserCode)
	if err == sql.ErrNoRows {
		return "", false, nil
	} else if err != nil {
		return "", false, fmt.Errorf("error fetching Team player %v", err)
	}

	return UserCode, true, nil
}

func GetEventHasGameTypeId(eventId int64, gameId int64, gameTypeId int64, ageGroupId int64) (int64, error) {
	query := `SELECT eht.id
			FROM event_has_game_types eht
			JOIN event_has_games ehg ON eht.event_has_game_id = ehg.id
			JOIN events e ON ehg.event_id = e.id
			WHERE e.id = $1
			AND ehg.game_id = $2
			AND eht.game_type_id = $3
			AND eht.age_group_id = $4`
	var id int64
	err := database.DB.QueryRow(query, eventId, gameId, gameTypeId, ageGroupId).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("no matching record found ")
	}
	// fmt.Println("event has game type id : = ", id)
	return id, nil
}

type NecessaryData struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	ContactNo string `json:"contact_no"`
	IsCaptain bool   `json:"is_captain"`
	Age       int    `json:"age"`
}

/*
func GetMembersByTeamId(TeamId int64) ([]NecessaryData, error) {

		var jsonData string

		var members []NecessaryData
		err := database.DB.QueryRow(`
			SELECT COALESCE(
		  json_agg(json_build_object('id', u.id, 'name', u.name, 'contact_no', u.mobile_no, 'is_captain', u.id=eht.team_captain),
			'age', EXTRACT(YEAR FROM AGE(CURRENT_DATE, u.date_of_birth)),
		  '[]'
		)
		FROM event_has_teams eht
		INNER JOIN event_has_users ehu ON ehu.event_has_team_id = eht.id
		INNER JOIN users u ON u.id = ehu.user_id
		WHERE eht.id = $1;
			`, TeamId).Scan(&jsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch team_members: %v", err)
		}

		err = json.Unmarshal([]byte(jsonData), &members)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
		}

		return members, err
	}
*/
func GetMembersByTeamId(TeamId int64) ([]NecessaryData, error) {
	var jsonData string

	var members []NecessaryData
	err := database.DB.QueryRow(`
        SELECT COALESCE(
            json_agg(json_build_object(
                'id', u.id,
                'name', u.name,
                'contact_no', u.mobile_no,
                'is_captain', u.id = eht.team_captain,
                'age', COALESCE(EXTRACT(YEAR FROM AGE(CURRENT_DATE, ud.dob))::integer, 0)
            )),
            '[]'
        )
        FROM event_has_teams eht
        INNER JOIN event_has_users ehu ON ehu.event_has_team_id = eht.id
        INNER JOIN users u ON u.id = ehu.user_id
				LEFT JOIN user_details ud ON u.id = ud.user_id  -- Add join to user_details
        WHERE eht.id = $1
    `, TeamId).Scan(&jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team_members: %v", err)
	}

	err = json.Unmarshal([]byte(jsonData), &members)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return members, nil
}

// GetTournamentTypeByEventAndGame fetches the tournament type based on event ID and game ID
func GetTournamentTypeByEventAndGame(eventId int64, gameId int64) (string, error) {
	var tournamentType string

	query := `SELECT type_of_tournament FROM event_has_games WHERE event_id = $1 AND game_id = $2 LIMIT 1`
	err := database.DB.QueryRow(query, eventId, gameId).Scan(&tournamentType)
	if err != nil {
		return "", err
	}

	return tournamentType, nil
}

func SetIsLastRound(eventHasGameTypeId int64) (bool, error) {
	query := `
	UPDATE event_has_game_types
	SET is_last_round = TRUE
	WHERE id = $1
`

	result, err := database.DB.Exec(query, eventHasGameTypeId)
	if err != nil {
		return false, fmt.Errorf("failed to update is_last_round: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return false, fmt.Errorf("no rows were updated")
	}

	return true, nil
}

func GetLastRound(eventHasGameTypeId int64) (bool, error) {
	query := `
		SELECT is_last_round
		FROM event_has_game_types
		WHERE id = $1
	`

	var isLastRound bool
	err := database.DB.QueryRow(query, eventHasGameTypeId).Scan(&isLastRound)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("no entry found for id %d", eventHasGameTypeId)
		}
		return false, fmt.Errorf("error querying is_last_round: %v", err)
	}

	return isLastRound, nil
}

func GetTeamIdByUserAndGame(eventId int64, gameId int64, userId int64) (int64, error) {
	query := `
		SELECT event_has_team_id FROM
		event_has_users
		WHERE user_id = $1 AND  game_id = $2 AND event_id= $3
	`
	var TeamId int64

	err := database.DB.QueryRow(query, userId, gameId, eventId).Scan(&TeamId)
	if err != nil {
		return 0, fmt.Errorf("error fetching Team Id %v", err)
	}

	return TeamId, nil
}
