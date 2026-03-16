package models

import (
	// "crypto"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/utils"
	"strconv"
	"strings"
	"time"
)

// EventEntry represents the entire event entry form submitted by an organizer.
type EventEntry struct {
	CreatedByEncId string      `form:"created_by_id"` // Encrypted user ID (from frontend)
	CreatedById    int64       `form:"-"`             // Decrypted actual user ID (used internally)
	EventEncId     string      `form:"event_id"`      // Encrypted event ID (from frontend)
	EventId        int64       `form:"-"`             // Decrypted event ID
	Games          []GameEntry `form:"-"`             // List of games and associated teams
	CreatedAt      time.Time   `form:"created_at,omitempty"`
	UpdatedAt      time.Time   `form:"updated_at,omitempty"`
}

// GameEntry represents a game in an event along with its types and teams.
type GameEntry struct {
	EventHasGameEncId    string      `json:"event_has_game_id"` // encrypted event–game link ID
	EventHasGameId       int64       `json:"-"`                 // decrypted event–game ID
	GameName             string      `json:"game_name"`
	IsTshirtSizeRequired bool        `json:"is_tshirt_size_required"`
	Types                []TypeEntry `json:"types"` // slice of types with their teams
}

// TypeEntry holds a game type and its associated teams.
type TypeEntry struct {
	TypeEncId string  `json:"type_id"` // encrypted game type ID
	TypeId    int64   `json:"-"`       // decrypted type ID
	TypeName  string  `json:"type_name"`
	Teams     []Teams `json:"teams"`
}

// Teams holds team-level data including members, captain, type link, age group, and metadata.
type Teams struct {
	TeamEncId             string            `json:"team_id"`
	TeamId                int64             `json:"-"`
	AgeGroupCategory      string            `json:"age_group_category"`
	AgeGroupEncId         string            `json:"age_group_id"`
	AgeGroupId            int64             `json:"-"`
	EventHasGameTypeEncId string            `json:"event_has_game_type_id"`
	EventHasGameTypeId    int64             `json:"-"`
	TeamName              string            `json:"team_name"`
	TeamLogoPath          string            `json:"team_logo_path,omitempty"`
	TeamMemberEncId       []string          `json:"team_member_ids"`
	TeamMemberIDs         []int64           `json:"-"`
	TeamCaptainEncId      string            `json:"team_captain_id"`
	TeamCaptainID         int64             `json:"-"`
	TeamCaptain           string            `json:"captain_name"`
	Slug                  string            `json:"slug,omitempty"`
	Status                string            `json:"status,omitempty"`
	GroupNo               *int              `json:"group_no"`
	CreatedAt             time.Time         `json:"created_at,omitempty"`
	UpdatedAt             time.Time         `json:"updated_at,omitempty"`
	TshirtSize            map[string]string `json:"tshirt_size"`
}

type DeleteStruct struct {
	Id           int64  `json:"id"`
	TeamLogoPath string `json:"team_logo_path"`
	isDeleted    bool
}

// FolderPath defines where uploaded team logos are stored
const FolderPath = "public/event/team_logos"

// SaveGames processes the submitted EventEntry form by:
// - Validating the games and their teams
// - Deleting old teams and user entries
// - Saving the new teams and related information to the database
//
// Params:
//   - EventForm (EventEntry): The event entry containing game and team data to be saved.
//
// Returns:
//   - *EventEntry: The updated event form with new team and game data.
//   - *sql.Tx: The ongoing transaction that can be committed or rolled back.
//   - error: Any error encountered during the processing, or nil if successful.
func SaveGames(EventForm EventEntry) (*EventEntry, *sql.Tx, error) {
	// Start a transaction
	tx, err := database.DB.Begin()
	if err != nil {
		return &EventForm, nil, fmt.Errorf("error starting transaction-->%v", err)
	}

	// Assign default timestamps if not set
	if EventForm.CreatedAt.IsZero() {
		EventForm.CreatedAt = time.Now()
	}
	if EventForm.UpdatedAt.IsZero() {
		EventForm.UpdatedAt = time.Now()
	}

	for i, Game := range EventForm.Games {
		// Get game ID and validate it belongs to the specified event
		GameId, eventMatches, err := GetGameIdByEventGameId(Game.EventHasGameId, EventForm.EventId)
		if err != nil {
			return &EventForm, tx, fmt.Errorf("could not fetch game_id -> %v", err)
		}
		if !eventMatches {
			return &EventForm, tx, fmt.Errorf("this game does not belong in this event(event-id mismatch)")
		}

		AllTeams := GetFlatGameTeams(Game)

		//only relevant when update is happening else 0 entries will be deleted
		oldData, err := DeleteTeamAndUserEntries(EventForm.EventId, GameId, EventForm.CreatedById, tx)
		if err != nil {
			return &EventForm, tx, err
		}
		for i := range oldData {
			isTeamInNewData := false
			for _, team := range AllTeams {
				if oldData[i].Id == team.TeamId {
					isTeamInNewData = true
					break
				}
			}
			oldData[i].isDeleted = !isTeamInNewData
		}

		for j, Type := range Game.Types {
			for k, Team := range Type.Teams {
				// Validate that the game_type belongs to the current game
				query := `SELECT game_type_id, age_group_id, CASE WHEN event_has_game_id = $1 THEN TRUE ELSE FALSE END AS game_matches
				FROM event_has_game_types WHERE id=$2`
				var GameTypeId int64
				var gameMatches bool
				err := tx.QueryRow(query, Game.EventHasGameId, Team.EventHasGameTypeId).Scan(&GameTypeId, &Team.AgeGroupId, &gameMatches)
				if err != nil {
					tx.Rollback()
					return &EventForm, tx, fmt.Errorf("could not fetch game_type_id -> %v", err)
				}

				if !gameMatches {
					tx.Rollback()
					return &EventForm, tx, fmt.Errorf("this game_type or age_group does not belong in this event_game(event_has_game_id mismatch)")
				}

				maxReached, err := IsMaxRegistrationReached(EventForm.EventId, GameId, GameTypeId, Team.AgeGroupId)
				if err != nil {
					tx.Rollback()
					return &EventForm, tx, fmt.Errorf("is max registration reached failed: %v", err)
				}

				if maxReached {
					tx.Rollback()
					return &EventForm, tx, fmt.Errorf("one of the types you are trying to enroll has reached max-capacity")
				}

				// Check that captain is one of the team members
				teamHasCaptain := slices.Contains(Team.TeamMemberIDs, Team.TeamCaptainID)
				if !teamHasCaptain {
					tx.Rollback()
					return &EventForm, tx, fmt.Errorf("the captain is not in the team, please choose a captain already in the team")
				}

				// Generate slug for team name
				Team.Slug = utils.GenerateSlug(Team.TeamName)
				// fmt.Println("models-->TeamLogo before update block: ", Team.TeamLogoPath)

				// condition: team id is not 0 and team logo doesnot start with "http://localhost:8080/public\\event\\team_logos"
				if Team.TeamId != 0 && !(strings.HasPrefix(Team.TeamLogoPath, "http://localhost:8080/public\\event\\team_logos") || strings.HasPrefix(Team.TeamLogoPath, "http://localhost:8080/public/event/team_logos")) {
					for i := range oldData {
						if oldData[i].Id == Team.TeamId {
							oldData[i].isDeleted = true
							Team.TeamLogoPath = ""
							break
						}
					}
				}

				// insert team into event_has_teams table
				query = `INSERT INTO event_has_teams (event_id, game_id, game_type_id, age_group_id,team_name, team_captain, created_by, slug, created_at, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`
				err = tx.QueryRow(query, EventForm.EventId, GameId, GameTypeId, Team.AgeGroupId, Team.TeamName, Team.TeamCaptainID, EventForm.CreatedById, Team.Slug, EventForm.CreatedAt, EventForm.UpdatedAt).Scan(&Team.TeamId)
				if err != nil {
					tx.Rollback()
					return &EventForm, tx, fmt.Errorf("failed to insert into the database -> %v", err)
				}

				//if already exists
				if strings.HasPrefix(Team.TeamLogoPath, "http://localhost:8080/public\\event\\team_logos") || strings.HasPrefix(Team.TeamLogoPath, "http://localhost:8080/public/event/team_logos") {
					Team.TeamLogoPath = filepath.Join("public/event/team_logos", filepath.Base(Team.TeamLogoPath))
					LogoUpdate(Team.TeamLogoPath, Team.TeamId, tx)
				}
				// fmt.Println("models-->after change TeamLogo: ", Team.TeamLogoPath)
				convertedTshirtSizes := make(map[int64]string)

				for k, v := range Team.TshirtSize {
					userID, err := crypto.NDecrypt(k)
					if err != nil {
						tx.Rollback()
						return nil, tx, fmt.Errorf("failed to decrypt tshirt_size user ID: %v", err)
					}

					convertedTshirtSizes[userID] = v
					if Game.IsTshirtSizeRequired && v == "" {
						tx.Rollback()
						return nil, tx, fmt.Errorf("t-shirt size required")
					} else if !Game.IsTshirtSizeRequired && v != "" {
						tx.Rollback()
						return nil, tx, fmt.Errorf("unexpected t-shirt size")

					}
				}

				// Insert the team members into pivot table
				err = InsertTeamMembers(Team.TeamMemberIDs, convertedTshirtSizes, EventForm.EventId, GameId, Team.TeamId, Team.EventHasGameTypeId, tx)
				// err = InsertTeamMembers(Team.TeamMemberIDs, Team.TshirtSizes, EventForm.EventId, GameId, Team.TeamId, Team.EventHasGameTypeId, tx)
				if err != nil {
					//if there is an error in the function it'll be rolled back in the function itself
					return nil, tx, err
				}
				Type.Teams[k] = Team
			}

			//UPDATE BLOCK

			for i := range oldData {
				var oldPath string
				if oldData[i].isDeleted {
					oldPath = oldData[i].TeamLogoPath
				} else {
					continue
				}

				DeleteImageByImagePath(oldPath)
			}
			//UPDATE BLOCK:end
			Game.Types[j] = Type
		}
		EventForm.Games[i] = Game
	}
	return &EventForm, tx, nil
}

// DeleteTeamAndUserEntries deletes existing teams and user entries from an event/game before inserting updated ones.
// It fetches the IDs of teams marked as 'Pending' for deletion, removes the corresponding teams from the event,
// and also deletes associated user entries from the pivot table.
//
// Params:
//   - EventId (int64): The ID of the event from which teams and users are to be deleted.
//   - GameId (int64): The ID of the game for which the teams and users are to be deleted.
//   - CreatedBy (int64): The ID of the creator to filter the teams and users to be deleted.
//   - tx (*sql.Tx): The ongoing transaction used to execute the deletion operations.
//
// Returns:
//   - []DeleteStruct: A slice containing details of the deleted teams (e.g., team ID and logo path).
//   - error: Any error encountered during the deletion process, or nil if successful.
func DeleteTeamAndUserEntries(EventId int64, GameId int64, CreatedBy int64, tx *sql.Tx) ([]DeleteStruct, error) {
	//fetch the ids of teams to be deleted
	OldData := []DeleteStruct{}
	var oldJson sql.NullString
	query := `SELECT json_agg(
	    json_build_object(
	        'id', id,
	        'team_logo_path', team_logo_path
	    )
	) FROM event_has_teams WHERE game_id=$1 AND event_id=$2 AND created_by=$3 AND status='Pending';`
	err := tx.QueryRow(query, GameId, EventId, CreatedBy).Scan(&oldJson)
	if err != nil {
		tx.Rollback()
		return OldData, fmt.Errorf("failed to fetch id of teams to be deleted from event_has_teams-> %v", err)
	}

	if oldJson.Valid {
		err = json.Unmarshal([]byte(oldJson.String), &OldData)
		if err != nil {
			return OldData, fmt.Errorf("unmarshal error-->%v", err)
		}
	} else {
		return OldData, nil
	}
	//delete pivot table entries
	query = `DELETE FROM event_has_teams WHERE game_id=$1 AND event_id=$2 AND created_by=$3 AND status='Pending';`
	_, err = tx.Exec(query, GameId, EventId, CreatedBy)
	if err != nil {
		tx.Rollback()
		return OldData, fmt.Errorf("failed to delete previous entries from event_has_teams-> %v", err)
	}

	// Remove related user entries

	if len(OldData) > 0 {
		// Construct the SQL query with the right number of placeholders
		placeholders := []string{}
		args := []interface{}{}
		for i := range OldData {
			placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
			args = append(args, OldData[i].Id)
		}
		query = fmt.Sprintf(`DELETE FROM event_has_users WHERE event_has_team_id IN (%s)`, strings.Join(placeholders, ","))
		_, err = tx.Exec(query, args...)
		if err != nil {
			tx.Rollback()
			return OldData, fmt.Errorf("failed to delete from event_has_users -> %v", err)
		}
	}
	return OldData, nil
}

// CheckParticipationInGameType checks if a user is already registered in a specific game type of an event.
// It verifies if the user is either already registered in the event's game type, and if the user is active in the system.
// If a team ID is provided, it ensures that the user is not registered in the same team for the same game.
//
// Params:
//   - UserId (int64): The ID of the user to check for participation.
//   - EventId (int64): The ID of the event where the user may be registered.
//   - GameId (int64): The ID of the game type the user is being checked for participation in.
//   - TeamId (int64): The ID of the team (optional) to check that the user is not registered in the same team.
//   - tx (*sql.Tx): The ongoing transaction used to query the database.
//
// Returns:
//   - bool: True if the user is already registered in the specified game type, otherwise false.
//   - error: Any error encountered during the check, or nil if no error occurs.
func CheckParticipationInGameType(UserId int64, EventId int64, GameId int64, TeamId int64, tx *sql.Tx) (bool, error) {
	PlayerAlreadyRegistered := false
	var query string
	var err error
	if TeamId != 0 {
		query = `SELECT EXISTS (
			SELECT 1 FROM event_has_users WHERE game_id=$1 AND event_id=$2 AND user_id=$3 AND event_has_team_id!=$4
		)`
		err = tx.QueryRow(query, GameId, EventId, UserId, TeamId).Scan(&PlayerAlreadyRegistered)
	} else {
		query = `SELECT EXISTS (
			SELECT 1 FROM event_has_users WHERE game_id=$1 AND event_id=$2 AND user_id=$3
		)`
		err = tx.QueryRow(query, GameId, EventId, UserId).Scan(&PlayerAlreadyRegistered)
	}
	if err != nil {
		tx.Rollback()
		return PlayerAlreadyRegistered, fmt.Errorf("database error: %v", err)
	}

	// Check that the player exists and is active
	var playerExists bool
	query = `SELECT EXISTS (
		SELECT 1 FROM users WHERE id=$1 AND status = 'Active'
	)`
	err = tx.QueryRow(query, UserId).Scan(&playerExists)
	if err != nil {
		tx.Rollback()
		return PlayerAlreadyRegistered, fmt.Errorf("database error: %v", err)
	}
	if !playerExists {
		tx.Rollback()
		return PlayerAlreadyRegistered, fmt.Errorf("player with id '%v' does not exists in the system", UserId)
	}

	return PlayerAlreadyRegistered, nil
}

// GetSavedGames fetches previously saved (Pending) teams created by the user for a specific event.
// It returns all the games for the event along with their pending teams, grouped by game type.
// Each GameEntry contains the event–game link ID, game name, and a slice of TypeEntry,
// where each TypeEntry holds a type ID and its associated teams.
//
// Params:
//   - CreatedById (int64): ID of the user who created the teams.
//   - EventId     (int64): ID of the event for which to fetch pending teams.
//
// Returns:
//   - []GameEntry: slice of GameEntry, one per game in the event.
//   - error: non‑nil if the query or unmarshalling fails.
func GetSavedGames(CreatedById int64, EventId int64) ([]GameEntry, error) {
	var saved []GameEntry

	query := `
	WITH team_objs AS (
		SELECT
		ehg.id                        AS event_has_game_id,
		g.game_name                   AS game_name,
		eht.game_type_id              AS type_id,
		eht.age_group_id              AS age_group_id,
		JSON_BUILD_OBJECT(
			'team_id',            eht.id::TEXT,
			'team_name',          eht.team_name,
			'age_group_id',       eht.age_group_id::TEXT,
			'event_has_game_type_id', ehgt.id::TEXT,
			'team_captain_id',    eht.team_captain::TEXT,
			'team_member_ids', (
				SELECT COALESCE(ARRAY_AGG(ehu.user_id::TEXT), ARRAY[]::TEXT[])
				FROM event_has_users ehu
				WHERE ehu.event_has_team_id = eht.id
			),
			'team_logo_path',     eht.team_logo_path,
			'slug',               eht.slug,
			'status',             eht.status,
			'created_at',         TO_CHAR(eht.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
			'updated_at',         TO_CHAR(eht.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
		) AS team_json
		FROM event_has_teams eht
		JOIN event_has_games ehg
		ON eht.event_id = ehg.event_id
		AND eht.game_id  = ehg.game_id
		JOIN event_has_game_types ehgt
		ON ehg.id = ehgt.event_has_game_id
		AND ehgt.game_type_id = eht.game_type_id
		AND ehgt.age_group_id  = eht.age_group_id
		JOIN games g
		ON g.id = eht.game_id
		WHERE
		eht.created_by = $1
		AND ehg.event_id = $2
		AND eht.status = 'Pending'
	),
	types_grouped AS (
	  SELECT
	    event_has_game_id,
	    type_id,
	    JSON_AGG(team_json ORDER BY team_json->>'team_name') AS teams
	  FROM team_objs
	  GROUP BY event_has_game_id, type_id
	),
	games_nested AS (
	  SELECT
	    tg.event_has_game_id,
	    tn.game_name,
	    JSON_AGG(
	      JSON_BUILD_OBJECT(
	        'type_id', tg.type_id::TEXT,
	        'teams',   tg.teams
	      ) ORDER BY tg.type_id::INT
	    ) AS types
	  FROM types_grouped tg
	  JOIN (
	    SELECT DISTINCT ehg.id AS event_has_game_id, g.game_name
	    FROM event_has_games ehg
	    JOIN games g ON g.id = ehg.game_id
	    WHERE ehg.event_id = $2
	  ) tn
	    ON tg.event_has_game_id = tn.event_has_game_id
	  GROUP BY tg.event_has_game_id, tn.game_name
	)
	SELECT
	  event_has_game_id,
	  game_name,
	  types::TEXT
	FROM games_nested;
	`

	rows, err := database.DB.Query(query, CreatedById, EventId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry GameEntry
		var typesJSON string
		if err := rows.Scan(&entry.EventHasGameEncId, &entry.GameName, &typesJSON); err != nil {
			return nil, err
		}

		// Unmarshal the JSON array of types into entry.Types
		if err := json.Unmarshal([]byte(typesJSON), &entry.Types); err != nil {
			return nil, fmt.Errorf("unmarshal types: %v", err)
		}

		//convert all encoded IDs to int64
		for ti, t := range entry.Types {
			// type ID
			t.TypeId, err = strconv.ParseInt(t.TypeEncId, 10, 64)
			if err != nil {
				return nil, err
			}
			t.TypeEncId = ""

			// each team within this type
			for si, team := range t.Teams {
				// team ID
				team.TeamId, err = strconv.ParseInt(team.TeamEncId, 10, 64)
				if err != nil {
					return nil, err
				}
				team.TeamEncId = ""

				// age group ID
				team.AgeGroupId, err = strconv.ParseInt(team.AgeGroupEncId, 10, 64)
				if err != nil {
					return nil, err
				}
				team.AgeGroupEncId = ""

				// event_has_game_type ID
				team.EventHasGameTypeId, err = strconv.ParseInt(team.EventHasGameTypeEncId, 10, 64)
				if err != nil {
					return nil, err
				}
				team.EventHasGameTypeEncId = ""

				// captain ID
				team.TeamCaptainID, err = strconv.ParseInt(team.TeamCaptainEncId, 10, 64)
				if err != nil {
					return nil, err
				}
				team.TeamCaptainEncId = ""

				// member IDs
				for _, enc := range team.TeamMemberEncId {
					n, err := strconv.ParseInt(enc, 10, 64)
					if err != nil {
						return nil, err
					}
					team.TeamMemberIDs = append(team.TeamMemberIDs, n)
				}
				team.TeamMemberEncId = nil

				// write back
				t.Teams[si] = team
			}
			entry.Types[ti] = t
		}

		saved = append(saved, entry)
	}

	return saved, nil
}

// FinalizeParticipation updates the status of all team entries for a specific creator and event to "Active",
// and deletes the entries that are not included in the final submission (Pending status teams that do not exist in the final GameId array).
//
// Params:
//   - CreatedById (int64): The ID of the creator of the event teams.
//   - EventId (int64): The ID of the event for which participation is being finalized.
//   - GameIdArr ([]int64): An array of game IDs that represent the final teams that are active.
//
// Returns:
//   - error: Any error encountered during the update and delete operations, or nil if the operation is successful.
func FinalizeParticipation(CreatedById int64, EventId int64, GameIdArr []int64) error {
	tx, _ := database.DB.Begin()
	if len(GameIdArr) > 0 {
		// Construct the SQL query with the right number of placeholders
		placeholders := []string{}
		args := []interface{}{}
		for i, id := range GameIdArr {
			placeholders = append(placeholders, fmt.Sprintf("$%d", (i+1)+2))
			args = append(args, id)
		}

		query := fmt.Sprintf(`
			UPDATE event_has_teams
			SET status='Active'
			WHERE created_by=$1
			AND event_id=$2
			AND game_id IN (
				SELECT game_id
				FROM event_has_games
				WHERE id IN (%s)
			)
		`, strings.Join(placeholders, ","))
		allArgs := append([]interface{}{CreatedById, EventId}, args...)
		_, err := tx.Exec(query, allArgs...)
		if err != nil {
			tx.Rollback()
			return err
		}
		query = `SELECT json_agg(
			json_build_object(
				'id', id,
				'team_logo_path', team_logo_path
				)
			)
			FROM event_has_teams
			WHERE created_by=$1 AND event_id=$2 AND status='Pending'`
		var jsonResult []byte
		err = tx.QueryRow(query, CreatedById, EventId).Scan(&jsonResult)
		if err != nil {
			tx.Rollback()
			return err
		}
		var deletedData []DeleteStruct
		if jsonResult == nil {
			deletedData = []DeleteStruct{}
		} else {
			err = json.Unmarshal(jsonResult, &deletedData)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to unmarshal team deletedData: %w", err)
			}
		}

		err = DeleteMembersAndLogoByTeamIds(deletedData, tx)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error deleting event users-->%v", err)
		}
		query = `DELETE from event_has_teams WHERE created_by=$1 AND event_id=$2 AND status='Pending'`
		_, err = tx.Exec(query, CreatedById, EventId)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		tx.Rollback()
		return fmt.Errorf("no gameIds in array")
	}
	tx.Commit()
	return nil
}

// IsMaxRegistrationReached checks whether the number of currently active teams
// for the given event, game, game‐type, and age‐group has met or exceeded
// the maximum allowed registrations for that game.
//
// Params:
//   - eventID     (int64):   the event identifier
//   - gameID      (int64):   the game identifier
//   - gameTypeID  (int64):   the game‐type identifier
//   - ageGroupID  (int64):   the age‐group identifier
//
// Returns:
//   - bool: true if the active team count ≥ max_registration, false otherwise
//   - error: non‑nil if the query fails
func IsMaxRegistrationReached(eventID, gameID, gameTypeID, ageGroupID int64) (bool, error) {

	query := `
    SELECT
      (
        SELECT COUNT(*)
        FROM event_has_teams t
        WHERE t.status = 'Active'
          AND t.event_id       = $1
          AND t.game_id        = $2
          AND t.game_type_id   = $3
          AND t.age_group_id   = $4
      ) >= COALESCE(NULLIF(g.max_registration, '')::INT, 0)
      AS max_reached
    FROM event_has_games g
    WHERE g.event_id = $1
      AND g.game_id  = $2
    LIMIT 1;
    `

	var reached bool
	err := database.DB.
		QueryRow(query, eventID, gameID, gameTypeID, ageGroupID).
		Scan(&reached)

	if err != nil {
		return false, fmt.Errorf("error checking max registration: %w", err)
	}
	return reached, nil
}

func DeleteMembersAndLogoByTeamIds(teamData []DeleteStruct, tx *sql.Tx) error {
	if len(teamData) == 0 {
		return nil // Nothing to delete
	}

	// Build placeholders like $1, $2, ..., $n
	placeholders := make([]string, len(teamData))
	args := make([]interface{}, len(teamData))
	for i, data := range teamData {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = data.Id

		DeleteImageByImagePath(data.TeamLogoPath)
	}

	query := fmt.Sprintf(
		`DELETE FROM event_has_users WHERE event_has_team_id IN (%s)`,
		strings.Join(placeholders, ","),
	)

	_, err := tx.Exec(query, args...)
	return err
}

func DeleteImageByImagePath(oldPath string) {
	if oldPath != "" && filepath.Base(oldPath) != "staticTeamLogo.png" {
		oldPath = filepath.Join(FolderPath, filepath.Base(oldPath))
		if err := os.Remove(oldPath); err == nil {
			fmt.Println("Deleted old logo:", oldPath)
		} else if strings.Contains(err.Error(), "The system cannot find the file specified") || strings.Contains(err.Error(), "no such file or directory") {
			fmt.Println("no file deleted:", oldPath, " -->", err)
		} else {
			fmt.Printf("failed to delete old logo at %v -->%v\n", oldPath, err)
		}
	}
}

func GetFlatGameTeams(game GameEntry) []Teams {
	var AllTeams []Teams
	for _, gameType := range game.Types {
		AllTeams = append(AllTeams, gameType.Teams...)
	}
	return AllTeams
}

func VerifyTeamName(Name string, eventId int64, gameId int64, teamId int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM event_has_teams
			WHERE 
				event_id = $1
				AND game_id = $2
				AND id != $3
				AND team_name ILIKE $4
		);
	`
	var isTeamNameTaken bool
	err := database.DB.QueryRow(query, eventId, gameId, teamId, Name).Scan(&isTeamNameTaken)
	if err != nil {
		return false, fmt.Errorf("database error in VerifyTeamName-->%v", err)
	}
	return !isTeamNameTaken, nil
}
