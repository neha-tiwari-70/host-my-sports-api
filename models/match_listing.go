package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"
	"strings"
	"time"
)

func GetMatchesByEventGameAndType(eventID, gameID int64, gameTypeIDs, categoryIDs []int64, TeamId ...int64) ([]Match, error) {
	var matches []Match

	// Prepare placeholders
	gtPlaceholders := make([]string, len(gameTypeIDs))
	catPlaceholders := make([]string, len(categoryIDs))

	argPos := 3
	for i := range gameTypeIDs {
		gtPlaceholders[i] = fmt.Sprintf("$%d", argPos)
		argPos++
	}
	for i := range categoryIDs {
		catPlaceholders[i] = fmt.Sprintf("$%d", argPos)
		argPos++
	}

	query := fmt.Sprintf(`
		WITH ranked_teams AS (
			SELECT
				m.id AS match_id,
				m.match_name,
				m.isDraw,
				m.scheduled_date,
				m.venue_link,
				m.venue,
				m.start_time,
				m.round_no,
				eht.id AS team_id,
				eht.team_name,
				eht.team_logo_path,
				ROW_NUMBER() OVER (PARTITION BY m.id ORDER BY eht.id) AS rn
			FROM matches m
			JOIN matches_has_teams mht ON m.id = mht.match_id
			JOIN event_has_teams eht ON mht.team_id = eht.id
			WHERE eht.event_id = $1 AND eht.game_id = $2
			AND eht.game_type_id IN (%s)
			AND eht.age_group_id IN (%s)
		)
		SELECT
			t1.match_id,
			t1.match_name,
			t1.venue_link,
			t1.venue,
			t1.isDraw,
			t1.scheduled_date,
			t1.start_time,
			t1.round_no,
			t1.team_id AS team1_id,
			t1.team_name AS team1_name,
			t1.team_logo_path AS team1_logo,
			t2.team_id AS team2_id,
			t2.team_name AS team2_name,
			t2.team_logo_path AS team2_logo
		FROM ranked_teams t1
		JOIN ranked_teams t2 ON t1.match_id = t2.match_id AND t1.rn = 1 AND t2.rn = 2`,
		strings.Join(gtPlaceholders, ","), strings.Join(catPlaceholders, ","),
	)

	args := append([]interface{}{eventID, gameID}, toInterfaceSlice(gameTypeIDs)...)
	args = append(args, toInterfaceSlice(categoryIDs)...)

	if len(TeamId) > 0 {
		query += fmt.Sprintf(`
			WHERE t1.team_id=$%d OR t2.team_id=$%d
		`, len(args)+1, len(args)+1)
		args = append(args, TeamId[0])
	}

	query += `
	ORDER BY t1.match_id;`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var match Match
		var team1Logo, team2Logo sql.NullString

		err := rows.Scan(
			&match.MatchId,
			&match.MatchName,
			&match.VenueLink,
			&match.VenueName,
			&match.IsDraw,
			&match.ScheduledDate,
			&match.StartTime,
			&match.RoundNo,
			&match.Team1ID,
			&match.Team1Name,
			&team1Logo,
			&match.Team2ID,
			&match.Team2Name,
			&team2Logo,
		)
		if err != nil {
			return nil, err
		}

		const defaultLogo = "public/static/staticTeamLogo.png"

		if team1Logo.Valid && fileExists(team1Logo.String) {
			match.Team1Logo = team1Logo.String
		} else {
			match.Team1Logo = defaultLogo
		}
		// fmt.Println("logo", match.Team1Logo)
		if team2Logo.Valid && fileExists(team2Logo.String) {
			match.Team2Logo = team2Logo.String
		} else {
			match.Team2Logo = defaultLogo
		}

		matches = append(matches, match)
	}

	return matches, nil
}

func toInterfaceSlice(ids []int64) []interface{} {
	result := make([]interface{}, len(ids))
	for i, id := range ids {
		result[i] = id
	}
	return result
}

func IsUserParticipating(userID, matchID int64) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM matches_has_teams mht
			JOIN participants p ON p.team_id = mht.team_id
			WHERE mht.match_id = $1 AND p.user_id = $2
		)
	`
	err := database.DB.QueryRow(query, matchID, userID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func GetMatches(search, sort, dir, status, scheduled_date string, limit, offset int64) (int, []Match, error) {
	var matches []Match
	args := []interface{}{limit, offset}
	query := `
	SELECT
    m.id AS match_id,
	m.event_has_game_types,
    m.match_name,
    m.scheduled_date,
    m.venue,
    m.venue_link,
    m.start_time,
    m.isDraw,
	COUNT(m.id) OVER() as totalRecords,

    -- Team 1 Info
    t1.id AS team1_id,
    e1.team_name AS team1_name,
    e1.team_logo_path AS team1_logo,

    -- Team 2 Info
    t2.id AS team2_id,
    e2.team_name AS team2_name,
    e2.team_logo_path AS team2_logo,

	 -- Event and Game Info
    ehg.event_id,
    ehg.game_id,
    ehgt.game_type_id

	FROM matches m

	-- Join for team1 and team2 from matches_has_teams
	JOIN(
		SELECT match_id, MIN(id) AS team1_match_team_id, MAX(id) AS team2_match_team_id
		FROM matches_has_teams
		GROUP BY match_id
		HAVING COUNT(*) = 2
	) mt ON m.id = mt.match_id

	-- Join team1 record
	JOIN matches_has_teams t1 ON t1.id = mt.team1_match_team_id
	JOIN event_has_teams e1 ON t1.team_id = e1.id

	-- Join team2 record
	JOIN matches_has_teams t2 ON t2.id = mt.team2_match_team_id
	JOIN event_has_teams e2 ON t2.team_id = e2.id

	-- New joins to get event_id, game_id, game_type_id
	LEFT JOIN event_has_game_types ehgt ON ehgt.id = m.event_has_game_types
	LEFT JOIN event_has_games ehg ON ehg.id = ehgt.event_has_game_id
	LEFT JOIN events ev ON ev.id = ehg.event_id

	WHERE 1=1
	`

	currentDate := time.Now()
	if status != "" {
		switch status {
		case "Upcoming":
			// Filter events that are upcoming (start date in the future)
			query += " AND m.scheduled_date > $3"
			args = append(args, currentDate)
		case "Past":
			// Filter events that are in the past (end date before the current date)
			query += " AND m.scheduled_date < $3"
			args = append(args, currentDate)
		default:
			// If status contains multiple values, split them and filter accordingly
			statusValues := strings.Split(status, ",")
			statusPlaceholders := []string{}
			for _, s := range statusValues {
				statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, strings.TrimSpace(s))
			}
			query += fmt.Sprintf(" AND status IN (%s)", strings.Join(statusPlaceholders, ", "))
		}
	}
	// fmt.Println("Query is : ", query)
	// fmt.Println("Query is:", query)
	// fmt.Println("Args:", args)
	query += fmt.Sprintf(" ORDER BY m.%s %s LIMIT $1 OFFSET $2", sort, dir)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		// Log the error if the query fails
		fmt.Printf("Error querying matches : %v\n", err)
		return 0, nil, err
	}
	defer rows.Close()
	totalRecords := 0
	var team1Logo sql.NullString
	var team2Logo sql.NullString
	for rows.Next() {
		var match Match
		if err := rows.Scan(
			&match.MatchId,
			&match.EventHasGameTypeId,
			&match.MatchName,
			&match.ScheduledDate,
			&match.StadiumName,
			&match.VenueLink,
			&match.StartTime,
			&match.IsDraw,
			&totalRecords,
			&match.Team1ID,
			&match.Team1Name,
			&team1Logo,
			&match.Team2ID,
			&match.Team2Name,
			&team2Logo,
			&match.EventId,
			&match.GameId,
			&match.GameTypeId,
		); err != nil {
			// Log the error if there is an issue scanning the row
			fmt.Printf("Error scanning row : %v\n", err)
			return 0, nil, err
		}
		const defaultLogo = "public/static/staticTeamLogo.png"

		if team1Logo.Valid && fileExists(team1Logo.String) {
			match.Team1Logo = team1Logo.String
		} else {
			match.Team1Logo = defaultLogo
		}
		// fmt.Println("logo", match.Team1Logo)
		if team2Logo.Valid && fileExists(team2Logo.String) {
			match.Team2Logo = team2Logo.String
		} else {
			match.Team2Logo = defaultLogo
		}

		matches = append(matches, match)
	}
	return totalRecords, matches, nil
}

func GetTotalMatchCount(status, scheduledDate string) (int, error) {
	args := []interface{}{}
	query := `
	SELECT COUNT(*) FROM matches m
	WHERE 1=1
	`

	currentDate := time.Now()
	if status != "" {
		switch status {
		case "Upcoming":
			query += " AND m.scheduled_date > $1"
			args = append(args, currentDate)
		case "Past":
			query += " AND m.scheduled_date < $1"
			args = append(args, currentDate)
		default:
			statusValues := strings.Split(status, ",")
			statusPlaceholders := []string{}
			for _, s := range statusValues {
				statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, strings.TrimSpace(s))
			}
			query += fmt.Sprintf(" AND status IN (%s)", strings.Join(statusPlaceholders, ", "))
		}
	}

	var count int
	err := database.DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		fmt.Printf("Error getting total match count: %v\n", err)
		return 0, err
	}

	return count, nil
}
