package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"strings"
)

type PointTable struct {
	TeamID         string `json:"team_id"`
	TeamName       string `json:"team_name"`
	TeamLogo       string `json:"team_logo"`
	Played         int64  `json:"played"`
	Won            int64  `json:"won"`
	Bye            int64  `json:"bye"`
	Lost           int64  `json:"lost"`
	Draw           int64  `json:"draw"`
	Points         int64  `json:"points"`
	Last5          string `json:"last_5"`
	ScoredPoints   int64  `json:"match_point"`
	ScoredAt       string `json:"scored_at"`
	TournamentType string `json:"tournament_type"`
}

func GetPointTable(eventID, gameID int64, gameTypeIDs, categoryIDs []int64, TeamId ...int64) ([]PointTable, error) {
	var table []PointTable

	placeholdersGT := make([]string, len(gameTypeIDs))
	for i := range gameTypeIDs {
		placeholdersGT[i] = fmt.Sprintf("$%d", i+3)
	}

	placeholdersCT := make([]string, len(categoryIDs))
	for i := range categoryIDs {
		placeholdersCT[i] = fmt.Sprintf("$%d", i+3+len(gameTypeIDs))
	}

	query := fmt.Sprintf(`
	SELECT
		t.id AS team_id,
		t.team_name,
		t.team_logo_path AS team_logo,

		-- COUNT(CASE WHEN m.isDraw IS NOT NULL THEN 1 END) AS played,
		COUNT(DISTINCT m.id) AS played,
		COALESCE(SUM(CASE WHEN mht.points > opp.points THEN 1 ELSE 0 END), 0) AS won,
		COALESCE(SUM(CASE WHEN mht.points < opp.points THEN 1 ELSE 0 END), 0) AS lost,
		COALESCE(SUM(CASE WHEN m.isDraw = TRUE THEN 1 ELSE 0 END), 0) AS draw,

		COALESCE((
			SELECT COUNT(*)
			FROM matches m_bye
			JOIN matches_has_teams mht_bye ON mht_bye.match_id = m_bye.id
			WHERE mht_bye.team_id = t.id
			AND NOT EXISTS (
				SELECT 1
				FROM matches_has_teams mht_opp
				WHERE mht_opp.match_id = m_bye.id
					AND mht_opp.team_id != t.id
			)
		), 0) AS bye,

		COALESCE(SUM(CASE WHEN mht.points IS NULL THEN NULL ELSE mht.points END), NULL) AS points,

		(
			SELECT json_agg(result)
			FROM (
				SELECT
					CASE
						WHEN NOT EXISTS (
							SELECT 1
							FROM matches_has_teams opp2
							WHERE opp2.match_id = m2.id
							AND opp2.team_id != mht2.team_id
						) THEN 'B'
						WHEN mht2.points > opp2.points THEN 'W'
						WHEN mht2.points < opp2.points THEN 'L'
						WHEN mht2.points = opp2.points THEN 'D'
						ELSE 'N/A'
					END AS result
				FROM matches m2
				JOIN matches_has_teams mht2 ON m2.id = mht2.match_id
				LEFT JOIN matches_has_teams opp2 ON opp2.match_id = m2.id AND opp2.team_id != mht2.team_id
				WHERE mht2.team_id = t.id
				ORDER BY m2.id DESC
				LIMIT 5
			) result
		) AS last_matches,

		-- Points Scored & Scored At for Athletics
		CASE
			WHEN ehg.type_of_tournament = 'Atheletics' OR ehg.type_of_tournament IN ('Time Trial', 'Mass Start', 'Relay', 'Fun Ride', 'Endurance')
			 THEN (
				SELECT COALESCE(SUM(mths.points_scored), 0)
				FROM match_team_has_scores mths
				JOIN matches m2 ON m2.id = mths.match_id
				WHERE mths.team_id = t.id
				AND m2.event_has_game_types = m.event_has_game_types
			)
			ELSE NULL
		END AS points_scored,

		CASE
			WHEN ehg.type_of_tournament = 'Atheletics' OR ehg.type_of_tournament IN ('Time Trial', 'Mass Start', 'Relay', 'Fun Ride', 'Endurance')
			 THEN (
				SELECT MAX(mths.scored_at)
				FROM match_team_has_scores mths
				JOIN matches m2 ON m2.id = mths.match_id
				WHERE mths.team_id = t.id
				AND m2.event_has_game_types = m.event_has_game_types
			)
			ELSE NULL
		END AS scored_at

	FROM event_has_teams t
	JOIN matches_has_teams mht ON t.id = mht.team_id
	JOIN matches m ON mht.match_id = m.id
	LEFT JOIN matches_has_teams opp ON opp.match_id = m.id
									AND opp.team_id != t.id
	JOIN game_has_age_group gah ON gah.game_id = t.game_id
	JOIN age_group ag ON ag.id = gah.age_group_id
	JOIN event_has_games ehg ON ehg.event_id = t.event_id
							AND ehg.game_id = t.game_id
	JOIN event_has_game_types ehgt ON ehgt.event_has_game_id = ehg.id
								AND ehgt.game_type_id = t.game_type_id
	WHERE t.event_id = $1
	AND t.game_id = $2
	AND t.game_type_id IN (%s)
	AND ag.id IN (%s)
	AND m.isdraw IS NOT NULL
	`, strings.Join(placeholdersGT, ","), strings.Join(placeholdersCT, ","))

	args := []interface{}{eventID, gameID}
	args = append(args, toInterfaceSlice(gameTypeIDs)...)
	args = append(args, toInterfaceSlice(categoryIDs)...)

	if len(TeamId) > 0 {
		query += fmt.Sprintf(" AND t.id = $%d", len(args)+1)
		args = append(args, TeamId[0])
	}

	query += `
	GROUP BY t.id, t.team_name, t.team_logo_path, ehg.type_of_tournament, m.event_has_game_types
	ORDER BY points DESC, won DESC;`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var logo sql.NullString
		var row struct {
			TeamID       int64
			TeamName     string
			Played       int64
			Won          sql.NullInt64
			Lost         sql.NullInt64
			Draw         sql.NullInt64
			Bye          sql.NullInt64
			Points       sql.NullInt64
			LastMatches  sql.NullString
			ScoredPoints sql.NullInt64
			ScoredAt     sql.NullString
		}

		err := rows.Scan(
			&row.TeamID,
			&row.TeamName,
			&logo,
			&row.Played,
			&row.Won,
			&row.Lost,
			&row.Draw,
			&row.Bye,
			&row.Points,
			&row.LastMatches,
			&row.ScoredPoints,
			&row.ScoredAt,
		)
		if err != nil {
			return nil, err
		}

		teamLogo := "public/static/staticTeamLogo.png"
		if logo.Valid && fileExists(logo.String) {
			teamLogo = logo.String
		}

		last5 := "N/A"
		if row.LastMatches.Valid && strings.TrimSpace(row.LastMatches.String) != "" && row.LastMatches.String != "[]" {
			last5 = row.LastMatches.String
		}

		encryptedID := crypto.NEncrypt(row.TeamID)

		table = append(table, PointTable{
			TeamID:       encryptedID,
			TeamName:     row.TeamName,
			TeamLogo:     teamLogo,
			Played:       row.Played,
			Won:          getValidInt(row.Won),
			Lost:         getValidInt(row.Lost),
			Draw:         getValidInt(row.Draw),
			Bye:          getValidInt(row.Bye),
			Points:       getValidInt(row.Points),
			Last5:        last5,
			ScoredPoints: getValidInt(row.ScoredPoints),
			ScoredAt:     getValidString(row.ScoredAt),
		})
	}

	return table, nil
}

func getValidInt(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

func getValidString(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
