package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"
	"strings"
)

type MatchResultResponse struct {
	// MatchID    string `json:"match_id"`
	MatchEncId     string `json:"match_id"`
	MatchName      string `json:"match_name"`
	IsDraw         bool   `json:"is_draw"`
	ScheduledDate  string `json:"scheduled_date"`
	Venue          string `json:"venue"`
	StartTime      string `json:"start_time"`
	Team1Name      string `json:"team1_name"`
	Team1Logo      string `json:"team1_logo"`
	Team1Points    int64  `json:"team1_points"`
	Team2Name      string `json:"team2_name"`
	Team2Logo      string `json:"team2_logo"`
	Team2Points    int64  `json:"team2_points"`
	IsCompleted    bool   `json:"is_completed"`
	RoundNo       int64  `json:"round_no"`
	TournamentType string `json:"tournament_type"`
}

func GetMatchResultsByEventGameAndType(eventID, gameID, categoryID int64, gameTypeIDs []int64, TeamId ...int64) ([]MatchResultResponse, error) {
	var matchResults []MatchResultResponse

	placeholders := make([]string, len(gameTypeIDs))
	for i := range gameTypeIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+4)
	}

	query := fmt.Sprintf(`
		WITH ranked_teams AS (
			SELECT
				m.id AS match_id,
				m.match_name,
				m.isDraw,
				m.scheduled_date,
				m.venue,
				m.start_time,
				m.round_no,
				mht.team_id,
				eht.team_name,
				eht.team_logo_path,
				mht.points,
				ROW_NUMBER() OVER (PARTITION BY m.id ORDER BY eht.id) AS rn
			FROM matches m
			JOIN matches_has_teams mht ON m.id = mht.match_id
			JOIN event_has_teams eht ON mht.team_id = eht.id
			WHERE eht.event_id = $1
			  AND eht.game_id = $2
			  AND eht.age_group_id = $3
			  AND eht.game_type_id IN (%s)
		)
		SELECT
			t1.match_id,
			t1.match_name,
			t1.isDraw,
			t1.scheduled_date,
			t1.venue,
			t1.start_time,
			t1.round_no,

			t1.team_name AS team1_name,
			t1.team_logo_path AS team1_logo,
			t1.points AS team1_points,

			t2.team_name AS team2_name,
			t2.team_logo_path AS team2_logo,
			t2.points AS team2_points
		FROM ranked_teams t1
		JOIN ranked_teams t2 ON t1.match_id = t2.match_id AND t1.rn = 1 AND t2.rn = 2
	`, strings.Join(placeholders, ", "))

	args := append([]interface{}{eventID, gameID, categoryID}, toInterfaceSlice(gameTypeIDs)...)

	if len(TeamId) > 0 {
		query += fmt.Sprintf(`
			WHERE t1.team_id=$%d OR t2.team_id=$%d
		`, len(args)+1, len(args)+1)
		args = append(args, TeamId[0])
	}

	query += `ORDER BY t1.match_id;`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			matchEncID    sql.NullString
			matchName     sql.NullString
			isDraw        sql.NullBool
			scheduledDate sql.NullString
			venue         sql.NullString
			startTime     sql.NullString
			roundNo       sql.NullInt64
			team1Name     sql.NullString
			team1Logo     sql.NullString
			team1Points   sql.NullInt64
			team2Name     sql.NullString
			team2Logo     sql.NullString
			team2Points   sql.NullInt64
		)

		err := rows.Scan(
			&matchEncID,
			&matchName,
			&isDraw,
			&scheduledDate,
			&venue,
			&startTime,
			&roundNo,
			&team1Name,
			&team1Logo,
			&team1Points,
			&team2Name,
			&team2Logo,
			&team2Points,
		)
		if err != nil {
			return nil, err
		}

		// Default logo if not valid or file missing
		team1LogoPath := nullStringToStr(team1Logo)
		if !fileExists(team1LogoPath) {
			team1LogoPath = "public/static/staticTeamLogo.png"
		}
		team2LogoPath := nullStringToStr(team2Logo)
		if !fileExists(team2LogoPath) {
			team2LogoPath = "public/static/staticTeamLogo.png"
		}

		response := MatchResultResponse{
			MatchEncId:    nullStringToStr(matchEncID),
			MatchName:     nullStringToStr(matchName),
			IsDraw:        nullBoolToBool(isDraw),
			ScheduledDate: nullStringToStr(scheduledDate),
			Venue:         nullStringToStr(venue),
			StartTime:     nullStringToStr(startTime),
			RoundNo:       nullIntToInt(roundNo),
			Team1Name:     nullStringToStr(team1Name),
			Team1Logo:     team1LogoPath,
			Team1Points:   nullIntToInt(team1Points),
			Team2Name:     nullStringToStr(team2Name),
			Team2Logo:     team2LogoPath,
			Team2Points:   nullIntToInt(team2Points),
			IsCompleted:   isDraw.Valid,
		}

		matchResults = append(matchResults, response)
	}

	if len(matchResults) == 0 {
		return nil, fmt.Errorf("no match results found")
	}

	return matchResults, nil
}

func nullStringToStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullBoolToBool(nb sql.NullBool) bool {
	if nb.Valid {
		return nb.Bool
	}
	return false
}

func nullIntToInt(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// func toInterfaceSlice(slice []int64) []interface{} {
// 	result := make([]interface{}, len(slice))
// 	for i, v := range slice {
// 		result[i] = v
// 	}
// 	return result
// }

// func fileExists(path string) bool {
// 	// Placeholder – implement if needed
// 	return true
// }
