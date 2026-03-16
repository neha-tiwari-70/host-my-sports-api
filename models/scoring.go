package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sports-events-api/database"
	"strconv"
	"strings"
	"time"
)

type MatchTeamScorePayload struct {
	MatchEncID        string       `json:"match_id"`
	MatchID           int64        `json:"-"`
	TeamEncID         string       `json:"team_id"`
	TeamID            int64        `json:"-"`
	Scores            []Score      `json:"scores"`
	TypeOfTournament  string       `json:"type_of_tournament"`
	GameSuperCategory string       `json:"game_super_category"`
	GameSubCategory   string       `json:"game_sub_category"`
	Cards             []CardAction `json:"cards"`
}

type Score struct {
	EncID        string  `json:"id"`
	ID           int64   `json:"-"`
	PlayerEncID  string  `json:"player_id,omitempty"`
	PlayerID     int64   `json:"-"`
	SetNo        int     `json:"set_no,omitempty"`
	PointsScored int     `json:"points_scored"`
	ScoredAt     string  `json:"scored_at"`
	IsPenalty    bool    `json:"is_penalty"`
	Distance     float64 `json:"distance"`
	Metric       string  `json:"metric"`
	Action       string  `json:"action"`
}
type CardAction struct {
	EncID         string `json:"card_id,omitempty"`
	ID            int64  `json:"-"`
	PlayerEncID   string `json:"player_id"`
	PlayerID      int64  `json:"-"`
	CardType      string `json:"card_type"`
	Action        string `json:"action"`
	GivenAt       string `json:"given_at,omitempty"`
	SuspensionEnd string `json:"suspension_end,omitempty"`
}

type MatchPlayerCardPayload struct {
	MatchEncID string       `json:"match_id"`
	MatchID    int64        `json:"-"`
	TeamEncID  string       `json:"team_id"`
	TeamID     int64        `json:"-"`
	Cards      []CardAction `json:"cards"`
	GameType   string       `json:"game_type"`
	Action     string       `json:"action"`
}
type Card struct {
	EncID         string `json:"card_id,omitempty"`
	ID            int64  `json:"-"`
	PlayerEncID   string `json:"player_id"`
	PlayerID      int64  `json:"-"`
	YellowCards   int    `json:"yellow_cards"`
	RedCards      int    `json:"red_cards"`
	Suspensions   int    `json:"suspensions"`
	IsSentOff     bool   `json:"is_sent_off"`
	SuspensionEnd string `json:"suspension_end,omitempty"`
}

func ProcessMatchTeamScores(payload MatchTeamScorePayload) (MatchTeamScorePayload, error) {
	tx, err := database.DB.Begin()
	if err != nil {
		return payload, fmt.Errorf("failed to start transaction: %v", err)
	}

	for i := range payload.Scores {
		switch payload.Scores[i].Action {
		case "insert":
			payload.Scores[i], err = InsertScoreEntry(payload.Scores[i], payload.MatchID, payload.TeamID, tx)
			if err != nil {
				return payload, err
			}

		case "update":
			// validScore, err := IsScoreLinkedToMatchTeam(payload.MatchID, payload.TeamID, payload.Scores[i].ID, tx)
			// if err != nil {
			// 	tx.Rollback()
			// 	return payload, err
			// }
			// if !validScore {
			// 	tx.Rollback()
			// 	return payload, fmt.Errorf("the scores you are trying to update does not belong to this match team")
			// }
			err = DeleteScoreEntry(payload.Scores[i].ID, tx)
			if err != nil {
				return payload, err
			}
			payload.Scores[i], err = InsertScoreEntry(payload.Scores[i], payload.MatchID, payload.TeamID, tx)
			if err != nil {
				return payload, err
			}

		case "delete":
			validScore, err := IsScoreLinkedToMatchTeam(payload.MatchID, payload.TeamID, payload.Scores[i].ID, tx)
			if err != nil {
				tx.Rollback()
				return payload, err
			}
			if !validScore {
				tx.Rollback()
				return payload, fmt.Errorf("the scores you are trying to delete does not belong to this match team")
			}
			err = DeleteScoreEntry(payload.Scores[i].ID, tx)
			if err != nil {
				return payload, err
			}
		case "":
			continue

		default:
			tx.Rollback()
			return payload, fmt.Errorf("unsupported action: %s", payload.Scores[i].Action)
		}
	}

	tx.Commit()
	return payload, err
}

// func DetermineWinner(MatchId int64, superCategory string, subCategory string, tx *sql.Tx) (int64, bool, error) {
func GetMaxSetPointsByTeamId(teamID int64) (string, error) {
	var eventID, gameID int
	var maxSetPoints string

	// Step 1: Get event_id and game_id from event_has_teams
	queryTeam := `SELECT event_id, game_id FROM event_has_teams WHERE id = $1`
	err := database.DB.QueryRow(queryTeam, teamID).Scan(&eventID, &gameID)
	if err != nil {
		return "21", fmt.Errorf("error fetching event and game IDs %v", err)
	}

	// Step 2: Get maximum_set_points from event_has_games
	queryGame := `SELECT maximum_set_points FROM event_has_games WHERE event_id = $1 AND game_id = $2 LIMIT 1`
	err = database.DB.QueryRow(queryGame, eventID, gameID).Scan(&maxSetPoints)
	if err != nil {
		return "21", fmt.Errorf("error fetching maximum_set_points %v", err)
	}

	return maxSetPoints, nil
}

// DetermineWinner determines the winner of a match based on the match's superCategory and subCategory rules.
// Parameters:
//   - MatchId: unique identifier of the match
//   - superCategory: high-level match type ("SetOriented", "GoalOriented", "PointBased", "Athletics")
//   - subCategory: additional category detail (only relevant for "Athletics", e.g., "TimeBased" or "RankBased")
//   - tx: SQL transaction object used for querying and updating match-related data
//
// Returns:
//   - int64: the ID of the winning team (0 if draw or no winner)
//   - bool: whether the match ended in a draw
//   - error: if something went wrong in data retrieval, calculation, or if match is incomplete
//
// This function:
//  1. Retrieves match data (team IDs, game name, sets).
//  2. Uses different rules depending on `superCategory` to determine the winner.
//  3. Validates that required data is present and consistent.
//  4. Returns results or errors accordingly.
func DetermineWinner(MatchId int64, superCategory string, subCategory string, tx *sql.Tx) (int64, bool, error) {
	// Initialize storage for both teams' IDs (default zero values)
	var TeamIds = []int64{0, 0}

	// Nullable winner ID (can remain invalid if draw or incomplete)
	var WinnerId sql.NullInt64

	// Storage for game name and number of sets (as strings)
	var game string
	var sets string

	// Flags for checking conditions during evaluation
	var BothTeamsHaveScores bool
	var IsDraw bool
	var IsIncomplete bool

	// Step 1: Retrieve team IDs, game name, and set count for the given match
	err := tx.QueryRow(`
	SELECT
		team_ids[1] AS team1_id,
		team_ids[2] AS team2_id,
		game_name,
		sets
		FROM (
			SELECT ARRAY_AGG(mt.team_id ORDER BY mt.team_id) AS team_ids, gm.game_name, ehg.sets
			FROM matches_has_teams mt
			JOIN event_has_teams eht ON eht.id = mt.team_id
			JOIN games gm ON gm.id = eht.game_id
			JOIN event_has_games ehg ON eht.event_id = ehg.event_id AND eht.game_id = ehg.game_id
			WHERE mt.match_id = $1
			GROUP BY gm.game_name, ehg.sets
		) AS sub;`, MatchId).Scan(&TeamIds[0], &TeamIds[1], &game, &sets)
	if err != nil {
		// Fail early if team or game data cannot be retrieved
		tx.Rollback()
		return 0, false, fmt.Errorf("error finding second team---> %v", err)
	}

	// Step 2: Apply different winner determination logic based on superCategory
	switch superCategory {

	case "SetOriented":
		{
			// Check if both teams have scores recorded for the expected sets
			query := `SELECT EXISTS (
				SELECT 1
				FROM (
					SELECT DISTINCT team_id
					FROM match_team_has_scores
					WHERE set_number = $1
					AND team_id IN ($2, $3)
					AND match_id=$4
				) AS team_check
				HAVING COUNT(*) = 2
			)`
			err = tx.QueryRow(query, sets, TeamIds[0], TeamIds[1], MatchId).Scan(&BothTeamsHaveScores)
			if err != nil {
				tx.Rollback()
				return 0, false, fmt.Errorf("error finding team scores---> %v", err)
			}

			// Retrieve the maximum points needed to win a set (game-specific)
			maxSetPoints, err := GetMaxSetPointsByTeamId(TeamIds[0])

			maxTableTennisScore, _ := strconv.ParseInt(maxSetPoints, 10, 64)

			// Proceed only if both teams have scores recorded
			if BothTeamsHaveScores {

				query = `-- Step 1: Filter relevant scores for the match between two teams (372 and 375) across 5 sets
					WITH filtered_scores AS (
						SELECT set_number, team_id, points_scored
						FROM match_team_has_scores
						WHERE set_number BETWEEN 1 AND $1           -- Limit to sets 1 through $1
						AND team_id IN ($2, $3) AND match_id=$4               -- Only include the two teams involved in the match
					),

					-- Step 2: Pair the scores of the two teams for each set
					paired_scores AS (
						SELECT
							fs1.set_number as set_number,
							fs1.team_id AS team1_id,               -- First team in the pair
							fs1.points_scored AS team1_points,     -- First team's points
							fs2.team_id AS team2_id,               -- Second team in the pair
							fs2.points_scored AS team2_points      -- Second team's points
						FROM filtered_scores fs1
						JOIN filtered_scores fs2
							ON fs1.set_number = fs2.set_number     -- Same set
							AND fs1.team_id != fs2.team_id         -- Ensure teams are not the same (avoid self-join)
					),
					-- Step 2.5: Check if all sets are completed (i.e., at least one team has 21 points in each set)
					incomplete_sets AS (
						SELECT set_number
						FROM paired_scores
						GROUP BY set_number
						HAVING MAX(team1_points) < $5 AND MAX(team2_points) < $5
					),

					-- Step 3: Determine the winner for each set
					set_winners AS (
						SELECT DISTINCT
							set_number,
							CASE
								WHEN team1_points > team2_points THEN team1_id     -- team1 wins
								WHEN team1_points < team2_points THEN team2_id     -- team2 wins
								ELSE NULL                                           -- Draw if points are equal
							END AS winner_team_id  -- Ignore drawn sets
							FROM paired_scores
							GROUP BY (winner_team_id, paired_scores.set_number)
					),

					-- Step 4: Count how many sets each team has won
					team_wins AS (
						SELECT winner_team_id, COUNT(*) AS win_count
						FROM set_winners
						WHERE winner_team_id IS NOT NULL       -- Ignore drawn sets
						GROUP BY winner_team_id
					),

					-- Step 5: Identify the highest number of sets won by any team
					win_summary AS (
						SELECT
							MAX(win_count) AS max_wins         -- Get the maximum number of wins among teams
						FROM team_wins
					),

					-- Step 6: Identify all teams that have this highest win count
					-- (if two teams have same count, it's a draw)
					tied_teams AS (
						SELECT winner_team_id
						FROM team_wins, win_summary
						WHERE win_count = max_wins
					)

					-- Step 7: Final decision on winner and draw
					SELECT
						CASE
							WHEN COUNT(*) = 1 THEN MAX(winner_team_id)   -- Only one team with most wins => they are the winner
							ELSE NULL                                    -- More than one team => no winner
						END AS winner_team_id,
						CASE
							WHEN COUNT(*) = 1 THEN FALSE                 -- Single winner => not a draw
							ELSE TRUE                                    -- Multiple top scorers => draw
						END AS is_draw,
						EXISTS (SELECT 1 FROM incomplete_sets) AS has_incomplete_set
					FROM tied_teams;`

				err = tx.QueryRow(query, sets, TeamIds[0], TeamIds[1], MatchId, maxTableTennisScore).Scan(&WinnerId, &IsDraw, &IsIncomplete)
				if err != nil {
					tx.Rollback()
					return 0, false, fmt.Errorf("error finding winner---> %v", err)
				}

				// Fail if any set is incomplete
				if IsIncomplete {
					return 0, false, fmt.Errorf("one or more set is incomplete")
				}
			} else {
				tx.Rollback()
				return 0, false, fmt.Errorf("missing scores")
			}
		}

	case "GoalOriented":
		{
			// Count the total points scored per team
			var team1Count, team2Count int
			query := `
				SELECT team_id, COALESCE(SUM(points_scored), 0) AS score_count
				FROM match_team_has_scores
				WHERE match_id = $1 AND team_id IN ($2, $3)
				GROUP BY team_id
			`
			rows, err := tx.Query(query, MatchId, TeamIds[0], TeamIds[1])
			if err != nil {
				tx.Rollback()
				return 0, false, fmt.Errorf("error counting team scores ---> %v", err)
			}
			defer rows.Close()

			// Store team scores
			for rows.Next() {
				var teamID int64
				var count int
				if err := rows.Scan(&teamID, &count); err != nil {
					tx.Rollback()
					return 0, false, fmt.Errorf("error scanning team score counts ---> %v", err)
				}
				switch teamID {
				case TeamIds[0]:
					team1Count = count
				case TeamIds[1]:
					team2Count = count
				}
			}

			// Determine winner or draw
			if team1Count > team2Count {
				WinnerId.Int64 = TeamIds[0]
				IsDraw = false
			} else if team1Count < team2Count {
				WinnerId.Int64 = TeamIds[1]
				IsDraw = false
			} else {
				IsDraw = true
			}
		}

	case "PointBased":
		{
			// Similar to GoalOriented but uses GROUP BY team_id, points_scored
			var team1Count, team2Count int
			query := `
				SELECT team_id, points_scored
				FROM match_team_has_scores
				WHERE match_id = $1 AND team_id IN ($2, $3)
				GROUP BY team_id, points_scored
			`
			rows, err := tx.Query(query, MatchId, TeamIds[0], TeamIds[1])
			if err != nil {
				tx.Rollback()
				return 0, false, fmt.Errorf("error counting team scores ---> %v", err)
			}
			defer rows.Close()

			for rows.Next() {
				var teamID int64
				var count int
				if err := rows.Scan(&teamID, &count); err != nil {
					tx.Rollback()
					return 0, false, fmt.Errorf("error scanning team score counts ---> %v", err)
				}
				switch teamID {
				case TeamIds[0]:
					team1Count = count
				case TeamIds[1]:
					team2Count = count
				}
			}

			if team1Count > team2Count {
				WinnerId.Int64 = TeamIds[0]
				IsDraw = false
			} else if team1Count < team2Count {
				WinnerId.Int64 = TeamIds[1]
				IsDraw = false
			} else {
				IsDraw = true
			}
		}

	case "Athletics":
		type scores struct {
			TeamId       int64   `json:"team_id"`
			Distance     float64 `json:"distance"`
			ScoredAt     string  `json:"scored_at"`
			PointsScored int64   `json:"points_scored"`
		}

		// helper: assign points (3,2,1,0...)
		assignPoints := func(matchID int64, tx *sql.Tx, teamIDs []int64) error {
			for i, tid := range teamIDs {
				points := 0
				switch i {
				case 0:
					points = 3
				case 1:
					points = 2
				case 2:
					points = 1
				default:
					points = 0
				}
				if _, err := tx.Exec(
					`UPDATE match_team_has_scores SET points_scored = $1 WHERE match_id = $2 AND team_id = $3`,
					points, matchID, tid,
				); err != nil {
					tx.Rollback()
					return fmt.Errorf("error updating points for team %d ---> %v", tid, err)
				}
			}
			return nil
		}

		// helper: fetch rows as a JSON aggregate and unmarshal into []scores.
		// orderBy must be a valid column name (team_id, distance, scored_at, points_scored).
		// orderDir should be "ASC" or "DESC".
		fetchTeamsJSON := func(matchID int64, orderBy string, orderDir string) ([]scores, error) {
			// Build query with ordering inside json_agg so the returned JSON array preserves ordering.
			// Note: orderBy and orderDir are interpolated into the SQL string — ensure they originate from trusted sources.
			query := fmt.Sprintf(`
				SELECT COALESCE(json_agg(t ORDER BY t.%s %s), '[]')
				FROM (
					SELECT team_id, distance, scored_at, points_scored
					FROM match_team_has_scores
					WHERE match_id = $1
				) t
			`, orderBy, orderDir)

			var jsonData []byte
			if err := tx.QueryRow(query, matchID).Scan(&jsonData); err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("query error fetching %s ---> %v", orderBy, err)
			}

			var out []scores
			if err := json.Unmarshal(jsonData, &out); err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("json unmarshal error for %s ---> %v", orderBy, err)
			}
			return out, nil
		}
		// Branch into subCategory-specific logic
		switch subCategory {
		case "TimeBased":
			// Order ascending (fastest first). Use scored_at string to detect exact ties.
			teams, err := fetchTeamsJSON(MatchId, "scored_at", "ASC")
			if err != nil {
				tx.Rollback()
				return 0, false, err
			}
			if len(teams) == 0 {
				tx.Rollback()
				return 0, false, fmt.Errorf("no scores found for match")
			}

			// Extract team IDs and assign points
			var ids []int64
			for _, t := range teams {
				ids = append(ids, t.TeamId)
			}
			if err := assignPoints(MatchId, tx, ids); err != nil {
				return 0, false, err
			}

			// Winner is first team
			WinnerId = sql.NullInt64{Int64: teams[0].TeamId, Valid: true}
			// Draw if first two scored_at values equal
			IsDraw = len(teams) > 1 && teams[0].ScoredAt == teams[1].ScoredAt

		case "DistanceBased":
			// Order ascending (shortest? if you want farthest first use DESC). Original used ORDER BY distance ASC.
			teams, err := fetchTeamsJSON(MatchId, "distance", "DESC")
			if err != nil {
				return 0, false, err
			}
			if len(teams) == 0 {
				tx.Rollback()
				return 0, false, fmt.Errorf("no scores found for match")
			}

			var ids []int64
			for _, t := range teams {
				ids = append(ids, t.TeamId)
			}
			if err := assignPoints(MatchId, tx, ids); err != nil {
				return 0, false, err
			}

			WinnerId = sql.NullInt64{Int64: teams[0].TeamId, Valid: true}
			// floating equality check with tiny epsilon
			if len(teams) > 1 {
				const eps = 1e-9
				IsDraw = math.Abs(teams[0].Distance-teams[1].Distance) < eps
			}

		case "RankBased":
			// Highest points_scored wins (DESC)
			teams, err := fetchTeamsJSON(MatchId, "points_scored", "DESC")
			if err != nil {
				return 0, false, err
			}
			if len(teams) == 0 {
				tx.Rollback()
				return 0, false, fmt.Errorf("no points found for match")
			}

			// Winner is first
			WinnerId = sql.NullInt64{Int64: teams[0].TeamId, Valid: true}
			// Detect draw if top two have equal points
			IsDraw = len(teams) > 1 && teams[0].PointsScored == teams[1].PointsScored

		default:
			tx.Rollback()
			return 0, false, fmt.Errorf("unknown Athletics subCategory: %s", subCategory)
		}

	default:
		// Catch invalid superCategory values
		tx.Rollback()
		return 0, false, fmt.Errorf("bad function call")
	}

	// Return winner ID (0 if invalid), draw flag, and nil error
	return WinnerId.Int64, IsDraw, nil
}

func InsertScoreEntry(MatchScore Score, MatchId int64, TeamId int64, tx *sql.Tx) (Score, error) {
	var nullMetric sql.NullString
	if MatchScore.Metric == "" {
		nullMetric.Valid = false
	} else {
		nullMetric.Valid = true
		nullMetric.String = MatchScore.Metric
	}
	var nullDistance sql.NullFloat64
	if MatchScore.Distance == 0 {
		nullDistance.Valid = false
	} else {
		nullDistance.Valid = true
		nullDistance.Float64 = MatchScore.Distance
	}
	err := tx.QueryRow(`
			INSERT INTO match_team_has_scores (match_id, team_id, player_id, set_number, points_scored, scored_at, is_penalty, distance, metric)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		MatchId, TeamId, MatchScore.PlayerID, MatchScore.SetNo, MatchScore.PointsScored, MatchScore.ScoredAt, MatchScore.IsPenalty, nullDistance, nullMetric,
	).Scan(&MatchScore.ID)
	if err != nil {
		tx.Rollback()
		return MatchScore, fmt.Errorf("insert error: %v", err)
	}
	// fmt.Println("id:", MatchScore.ID)
	return MatchScore, nil
}

func IsScoreLinkedToMatchTeam(MatchId int64, TeamId int64, Id int64, tx *sql.Tx) (bool, error) {
	var IsLinked bool
	err := tx.QueryRow(`SELECT EXISTS (
			SELECT 1 FROM match_team_has_scores WHERE match_id=$1 AND team_id=$2 AND id=$3
		)`, MatchId, TeamId, Id).Scan(&IsLinked)
	if err != nil {
		return false, fmt.Errorf("error while checking if score is linked to match team--> %v", err)
	}

	return IsLinked, nil
}

func DeleteScoreEntry(MatchTeamHasScoreId int64, tx *sql.Tx) error {
	_, err := tx.Exec(`DELETE FROM match_team_has_scores WHERE id = $1`, MatchTeamHasScoreId)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("delete error---> %v", err)
	}
	return nil
}

func DeleteAllMatchScores(MatchId int64, tx *sql.Tx) error {
	_, err := tx.Exec(`DELETE FROM match_team_has_scores WHERE match_id = $1 AND (scored_at != '' OR distance IS NOT NULL) `, MatchId)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("delete error---> %v", err)
	}
	return nil
}

func GetScoresByMatchTeamID(MatchID int64, TeamID int64) ([]Score, error) {
	rows, err := database.DB.Query(`
		SELECT id, player_id, set_number, points_scored, scored_at, is_penalty, distance, metric
		FROM match_team_has_scores
		WHERE match_id = $1 AND team_id = $2
		order by set_number`, MatchID, TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// fmt.Println("point score:", rows)

	var scores []Score
	for rows.Next() {
		var s Score
		var nullMetric sql.NullString
		var nullDistance sql.NullFloat64

		// err := rows.Scan(&s.EncID, &s.PlayerEncID, &s.SetNo, &s.PointsScored, &s.ScoredAt) for tezting
		err := rows.Scan(&s.ID, &s.PlayerID, &s.SetNo, &s.PointsScored, &s.ScoredAt, &s.IsPenalty, &nullDistance, &nullMetric)
		if err != nil {
			return nil, err
		}
		if nullMetric.Valid {
			s.Metric = nullMetric.String
		}
		if nullDistance.Valid {
			s.Distance = nullDistance.Float64
		}
		scores = append(scores, s)
	}
	return scores, nil
}

func GetMatchHasTeamId(MatchId int64, TeamId int64, tx *sql.Tx) (int64, error) {
	query := `
	SELECT id FROM matches_has_teams WHERE match_id= $1 AND team_id= $2
	`
	var MatchHasTeamId int64
	err := tx.QueryRow(query, MatchId, TeamId).Scan(&MatchHasTeamId)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("database error while fetching MatchHasTeamId--> %v", err)
	}
	return MatchHasTeamId, nil
}

// AllocateWinner updates the match result with draw status and allocates points to teams.
// Points logic: 2 = win, 1 = draw, 0 = loss
func AllocateWinner(MatchId int64, WinTeamId int64, IsDraw bool, tx *sql.Tx) error {
	// Step 1: Update the match with draw status
	_, err := tx.Exec(`
		UPDATE matches
		SET isDraw = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2;
	`, IsDraw, MatchId)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error updating IsDraw --> %v", err)
	}

	// Step 2: Update team points in matches_has_teams based on win/loss/draw
	_, err = tx.Exec(`
		UPDATE matches_has_teams
		SET points = CASE
			WHEN $1 THEN 1                      -- if it's a draw, all teams get 1 point
			WHEN team_id = $2 THEN 2           -- winner gets 2 points
			ELSE 0                             -- loser gets 0 points
		END
		WHERE match_id = $3;
	`, IsDraw, WinTeamId, MatchId)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error updating team points --> %v", err)
	}

	return nil
}

func RemoveWinner(MatchId int64, tx *sql.Tx) error {
	// Step 1: Clear draw status
	_, err := tx.Exec(`
		UPDATE matches
		SET isDraw = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1;
	`, MatchId)
	if err != nil {
		return fmt.Errorf("error updating isDraw: %w", err)
	}

	// Step 2: Nullify points for all teams in that match
	_, err = tx.Exec(`
		UPDATE matches_has_teams
		SET points = NULL
		WHERE match_id = $1;
	`, MatchId)
	if err != nil {
		return fmt.Errorf("error updating match team points: %w", err)
	}

	return nil
}

func ProcessMatchPlayerCardsFromScores(payload MatchTeamScorePayload, gameSubCategory string) (MatchTeamScorePayload, error) {
	tx, err := database.DB.Begin()
	if err != nil {
		return payload, fmt.Errorf("failed to start transaction: %v", err)
	}

	_, err = tx.Exec(`DELETE FROM match_player_cards WHERE match_id = $1 AND team_id = $2`, payload.MatchID, payload.TeamID)
	if err != nil {
		tx.Rollback()
		return payload, fmt.Errorf("failed to delete old cards: %v", err)
	}

	for i := range payload.Cards {
		card := &payload.Cards[i]

		var givenAt sql.NullTime
		if card.GivenAt != "" {
			var t time.Time
			if strings.Contains(card.GivenAt, "T") {
				t, err = time.Parse(time.RFC3339, card.GivenAt)
			} else {
				t, err = time.Parse("15:04", card.GivenAt)
			}
			if err != nil {
				tx.Rollback()
				return payload, fmt.Errorf("invalid givenAt format for player %d: %v", card.PlayerID, err)
			}
			givenAt.Valid = true
			givenAt.Time = t
		}

		*card, err = InsertPlayerCard(*card, payload.MatchID, payload.TeamID, givenAt, tx)
		if err != nil {
			tx.Rollback()
			return payload, err
		}
	}

	if err := tx.Commit(); err != nil {
		return payload, err
	}

	return payload, nil
}

// Insert a new card row
func InsertPlayerCard(card CardAction, matchID, teamID int64, givenAt sql.NullTime, tx *sql.Tx) (CardAction, error) {
	yellow, red, suspensions := 0, 0, 0
	isSentOff := false
	var suspensionEnd string

	switch card.CardType {
	case "yellow":
		yellow = 1
	case "red":
		red = 1
		isSentOff = true
	case "suspension":
		suspensions = 1
		if card.SuspensionEnd != "" {
			suspensionEnd = card.SuspensionEnd
		} else if givenAt.Valid {
			suspensionEnd = givenAt.Time.Format("15:04")
		} else {
			suspensionEnd = ""
		}
	}

	err := tx.QueryRow(`
		INSERT INTO match_player_cards
		(match_id, team_id, player_id, yellow_cards, red_cards, suspensions, is_sent_off, suspension_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, matchID, teamID, card.PlayerID, yellow, red, suspensions, isSentOff, suspensionEnd).Scan(&card.ID)

	if err != nil {
		return card, fmt.Errorf("insert player card error: %v", err)
	}

	return card, nil
}

// Fetch cards by match & team
func GetCardsByMatchTeamID(matchID, teamID int64) ([]Card, error) {
	query := `
        SELECT id, player_id, yellow_cards, red_cards, suspensions, is_sent_off, suspension_end
        FROM match_player_cards
        WHERE match_id = $1 AND team_id = $2
    `
	rows, err := database.DB.Query(query, matchID, teamID)
	if err != nil {
		return nil, fmt.Errorf("error fetching cards: %v", err)
	}
	defer rows.Close()

	var cards []Card
	for rows.Next() {
		var c Card
		err = rows.Scan(&c.ID, &c.PlayerID, &c.YellowCards, &c.RedCards, &c.Suspensions, &c.IsSentOff, &c.SuspensionEnd)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}
