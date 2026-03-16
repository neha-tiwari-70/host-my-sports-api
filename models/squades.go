package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"strings"

	"github.com/lib/pq"
)

type PlayerStats struct {
	PlayerID      int64  `json:"-"`
	PlayerEncID   string `json:"player_id"`
	ContactNo     string `json:"contact_no"`
	IsCaptain     bool   `json:"is_captain"`
	PlayerName    string `json:"player_name"`
	GoalsScored   int    `json:"goals_scored"`
	ProfileImage  string `json:"profile_image"`
	YellowCards   int    `json:"yellow_cards"`
	RedCards      int    `json:"red_cards"`
	Suspensions   int    `json:"suspensions"`
	IsSentOff     bool   `json:"is_sent_off"`
	SuspensionEnd string `json:"suspension_end"`
	Age           int    `json:"age"`
}

type SquadStats struct {
	EventId     string        `json:"event_id"`
	TeamId      string        `json:"team_id"`
	CreatedById string        `json:"created_by_id"`
	TeamName    string        `json:"team_name"`
	MatchCount  int           `json:"matches"`
	WonGames    int           `json:"won"`
	LostGames   int           `json:"lost"`
	DrawnGames  int           `json:"drawn"`
	TotalPoints int           `json:"points"`
	TotalGoals  *int          `json:"goals"`
	TeamLogo    string        `json:"team_logo"`
	Players     []PlayerStats `json:"players"`
}

// for player enc id
func EncodeID(id int64) string {
	return fmt.Sprintf("ENC-%d", id)
}

func GetSquadStats(eventID, gameID int64, gameTypeIDs, categoryIDs []int64, TeamId ...int64) ([]SquadStats, error) {
	var squadStats []SquadStats

	gtPlaceholders := make([]string, len(gameTypeIDs))
	catPlaceholders := make([]string, len(categoryIDs))
	args := []interface{}{eventID, gameID}
	argIdx := 3

	for i, id := range gameTypeIDs {
		gtPlaceholders[i] = fmt.Sprintf("$%d", argIdx)
		args = append(args, id)
		argIdx++
	}
	for i, id := range categoryIDs {
		catPlaceholders[i] = fmt.Sprintf("$%d", argIdx)
		args = append(args, id)
		argIdx++
	}

	query := `
		SELECT
			et.id,
			et.team_name,
			et.team_logo_path,
			COUNT(DISTINCT mht.match_id) AS match_count,
			SUM(CASE WHEN mht.points > 0 THEN 1 ELSE 0 END) AS won_games,
			SUM(CASE WHEN mht.points < 0 THEN 1 ELSE 0 END) AS lost_games,
			SUM(CASE WHEN mht.points = 0 THEN 1 ELSE 0 END) AS drawn_games,
			COALESCE(SUM(mht.points), 0) AS total_points,
			CASE WHEN et.game_id = 1 THEN COUNT(mths.id) ELSE NULL END AS total_goals,
			e.created_by_id AS created_by_id
		FROM event_has_teams et
		JOIN matches_has_teams mht ON et.id = mht.team_id
		JOIN matches m ON mht.match_id = m.id
		LEFT JOIN match_team_has_scores mths ON mths.match_id = m.id AND mths.team_id = et.id
		LEFT JOIN events e ON et.event_id = e.id
		WHERE et.event_id = $1 AND et.game_id = $2`
	if len(TeamId) > 0 {
		query += fmt.Sprintf(` AND et.id=$%d`, len(args)+1)
		args = append(args, TeamId[0])
	}
	query += fmt.Sprintf(`
			AND et.game_type_id IN (%s)
			AND et.age_group_id IN (%s)
			AND et.status='Active'
		GROUP BY et.id, et.team_name, et.team_logo_path, et.game_id, e.created_by_id
	`, strings.Join(gtPlaceholders, ","), strings.Join(catPlaceholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch squad stats: %v", err)
	}
	defer rows.Close()

	teamIDs := make([]int64, 0)
	teamStatsMap := make(map[int64]*SquadStats)

	for rows.Next() {
		var teamID int64
		var squadStat SquadStats
		var logo sql.NullString
		var created_by_id int64
		err := rows.Scan(
			&teamID,
			&squadStat.TeamName,
			&logo,
			&squadStat.MatchCount,
			&squadStat.WonGames,
			&squadStat.LostGames,
			&squadStat.DrawnGames,
			&squadStat.TotalPoints,
			&squadStat.TotalGoals,
			&created_by_id,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		squadStat.TeamId = crypto.NEncrypt(teamID)
		squadStat.EventId = crypto.NEncrypt(eventID)
		squadStat.CreatedById = crypto.NEncrypt(created_by_id)
		if logo.Valid && fileExists(logo.String) {
			squadStat.TeamLogo = logo.String
		} else {
			squadStat.TeamLogo = "public/static/staticTeamLogo.png"
		}
		squadStat.CreatedById = crypto.NEncrypt(created_by_id)
		teamIDs = append(teamIDs, teamID)
		teamStatsMap[teamID] = &squadStat
	}

	if len(teamIDs) == 0 {
		return nil, fmt.Errorf("no squad stats found")
	}

	goalsQuery := `
		SELECT team_id, player_id, COALESCE(SUM(points_scored), 0) AS goals
		FROM match_team_has_scores
		WHERE team_id = ANY($1) AND player_id IS NOT NULL
		GROUP BY team_id, player_id
	`
	goalsRows, err := database.DB.Query(goalsQuery, pq.Array(teamIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goals scored: %v", err)
	}
	defer goalsRows.Close()

	cardsQuery := `
		SELECT
			team_id,
			player_id,
			SUM(yellow_cards) AS yellow_cards,
			SUM(red_cards) AS red_cards,
			SUM(suspensions) AS suspensions,
			BOOL_OR(is_sent_off) AS is_sent_off,
			MAX(COALESCE(suspension_end, '')) AS suspension_end
		FROM match_player_cards
		WHERE team_id = ANY($1)
		GROUP BY team_id, player_id
	`
	cardsRows, err := database.DB.Query(cardsQuery, pq.Array(teamIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cards: %v", err)
	}
	defer cardsRows.Close()

	playerGoals := make(map[int64]map[int64]int)
	for goalsRows.Next() {
		var teamID, playerID int64
		var goals int
		if err := goalsRows.Scan(&teamID, &playerID, &goals); err != nil {
			return nil, fmt.Errorf("failed to scan goals row: %v", err)
		}
		if _, ok := playerGoals[teamID]; !ok {
			playerGoals[teamID] = make(map[int64]int)
		}
		playerGoals[teamID][playerID] = goals
	}

	playerCards := make(map[int64]map[int64]PlayerStats)
	for cardsRows.Next() {
		var teamID, playerID int64
		var yellow, red, suspensions int
		var sentOff bool
		var suspensionEnd string

		if err := cardsRows.Scan(&teamID, &playerID, &yellow, &red, &suspensions, &sentOff, &suspensionEnd); err != nil {
			return nil, fmt.Errorf("failed to scan cards row: %v", err)
		}

		if _, ok := playerCards[teamID]; !ok {
			playerCards[teamID] = make(map[int64]PlayerStats)
		}
		playerCards[teamID][playerID] = PlayerStats{
			YellowCards:   yellow,
			RedCards:      red,
			Suspensions:   suspensions,
			IsSentOff:     sentOff,
			SuspensionEnd: suspensionEnd,
		}
	}

	for _, teamID := range teamIDs {
		members, err := GetMembersByTeamId(teamID)
		if err != nil {
			return nil, fmt.Errorf("failed to get team members: %v", err)
		}

		var players []PlayerStats
		for _, member := range members {
			goals := 0
			if teamGoalMap, ok := playerGoals[teamID]; ok {
				goals = teamGoalMap[member.Id]
			}

			profileImage, err := GetProfileImageById(int(member.Id))
			if err != nil {
				return nil, fmt.Errorf("failed to get profile image: %v", err)
			}

			players = append(players, PlayerStats{
				PlayerID:    member.Id,
				ContactNo:   member.ContactNo,
				IsCaptain:   member.IsCaptain,
				PlayerName:  member.Name,
				GoalsScored: goals,
				ProfileImage: func() string {
					if profileImage == "" || profileImage == "<nil>" {
						return ""
					}
					return profileImage
				}(),
				YellowCards: func() int {
					if p, ok := playerCards[teamID][member.Id]; ok {
						return p.YellowCards
					}
					return 0
				}(),
				RedCards: func() int {
					if p, ok := playerCards[teamID][member.Id]; ok {
						return p.RedCards
					}
					return 0
				}(),
				Suspensions: func() int {
					if p, ok := playerCards[teamID][member.Id]; ok {
						return p.Suspensions
					}
					return 0
				}(),
				IsSentOff: func() bool {
					if p, ok := playerCards[teamID][member.Id]; ok {
						return p.IsSentOff
					}
					return false
				}(),
				SuspensionEnd: func() string {
					if p, ok := playerCards[teamID][member.Id]; ok {
						return p.SuspensionEnd
					}
					return ""
				}(),
				Age: int(member.Age),
			})
		}

		teamStatsMap[teamID].Players = players
		squadStats = append(squadStats, *teamStatsMap[teamID])
	}

	return squadStats, nil
}

func GetSquadStatsWithoutMatches(eventID, gameID int64, gameTypeIDs, categoryIDs []int64) ([]SquadStats, error) {
	var squadStats []SquadStats

	gtPlaceholders := make([]string, len(gameTypeIDs))
	catPlaceholders := make([]string, len(categoryIDs))
	args := []interface{}{eventID, gameID}
	argIdx := 3

	for i, id := range gameTypeIDs {
		gtPlaceholders[i] = fmt.Sprintf("$%d", argIdx)
		args = append(args, id)
		argIdx++
	}
	for i, id := range categoryIDs {
		catPlaceholders[i] = fmt.Sprintf("$%d", argIdx)
		args = append(args, id)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT
			et.id,
			et.team_name,
			et.team_logo_path,
			0 AS match_count,
			0 AS won_games,
			0 AS lost_games,
			0 AS drawn_games,
			0 AS total_points,
			NULL AS total_goals
		FROM event_has_teams et
		WHERE et.event_id = $1 AND et.game_id = $2
			AND et.game_type_id IN (%s)
			AND et.age_group_id IN (%s)
			AND et.status='Active'
		GROUP BY et.id, et.team_name, et.team_logo_path, et.game_id
	`, strings.Join(gtPlaceholders, ","), strings.Join(catPlaceholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch squad stats: %v", err)
	}
	defer rows.Close()

	teamIDs := make([]int64, 0)
	teamStatsMap := make(map[int64]*SquadStats)

	for rows.Next() {
		var teamID int64
		var squadStat SquadStats
		var logo sql.NullString

		err := rows.Scan(
			&teamID,
			&squadStat.TeamName,
			// &squadStat.TeamLogo,
			&logo,
			&squadStat.MatchCount,
			&squadStat.WonGames,
			&squadStat.LostGames,
			&squadStat.DrawnGames,
			&squadStat.TotalPoints,
			&squadStat.TotalGoals,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// if logo.Valid && logo.String != "" {
		// 	squadStat.TeamLogo = logo.String
		// } else {
		// 	squadStat.TeamLogo = "public/static/staticTeamLogo.png"
		// }
		squadStat.TeamId = crypto.NEncrypt(teamID)
		squadStat.EventId = crypto.NEncrypt(eventID)
		if logo.Valid && fileExists(logo.String) {
			squadStat.TeamLogo = logo.String
		} else {
			squadStat.TeamLogo = "public/static/staticTeamLogo.png"
		}

		teamIDs = append(teamIDs, teamID)
		teamStatsMap[teamID] = &squadStat
	}

	if len(teamIDs) == 0 {
		return nil, fmt.Errorf("no squad stats found")
	}

	for _, teamID := range teamIDs {
		members, err := GetMembersByTeamId(teamID)
		if err != nil {
			return nil, fmt.Errorf("failed to get team members: %v", err)
		}

		var players []PlayerStats
		for _, member := range members {
			profileImage, err := GetProfileImageById(int(member.Id))
			if err != nil {
				return nil, fmt.Errorf("failed to get profile image: %v", err)
			}

			players = append(players, PlayerStats{
				PlayerID:    member.Id,
				ContactNo:   member.ContactNo,
				IsCaptain:   member.IsCaptain,
				PlayerName:  member.Name,
				GoalsScored: 0,
				ProfileImage: func() string {
					if profileImage == "" || profileImage == "<nil>" {
						return ""
					}
					return profileImage
				}(),
				Age: int(member.Age),
			})
		}

		teamStatsMap[teamID].Players = players
		squadStats = append(squadStats, *teamStatsMap[teamID])
	}

	return squadStats, nil
}

// check match is exist or not for related team
func CheckIfEventHasMatches(eventID, gameID int64) bool {
	var count int
	query := `
        SELECT COUNT(*)
        FROM matches
        WHERE event_has_game_types IN (
            SELECT id FROM event_has_game_types WHERE event_id = $1 AND game_id = $2
        )
    `
	err := database.DB.QueryRow(query, eventID, gameID).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func ReplacePlayerFromTeam(teamID, playerID int64, tx *sql.Tx, newPlayerID int64) error {
	// Check if the player is the team captain
	var isCaptain bool
	err := tx.QueryRow(`
        SELECT team_captain = $1
        FROM event_has_teams
        WHERE id = $2`, playerID, teamID).Scan(&isCaptain)
	if err != nil {
		return fmt.Errorf("failed to check if player is captain: %v", err)
	}
	if isCaptain {
		var newCaptainID int64
		if newPlayerID != 0 {
			newCaptainID = newPlayerID
		} else {
			err = tx.QueryRow(`
                SELECT user_id
                FROM event_has_users
                WHERE event_has_team_id = $1
                AND user_id != $2
                LIMIT 1`, teamID, playerID).Scan(&newCaptainID)
			if err != nil {
				if err == sql.ErrNoRows {
					return fmt.Errorf("you can't delete captain but you can replace captain")
				}
				return fmt.Errorf("failed to fetch new captain ID: %v", err)
			}
		}

		_, err = tx.Exec(`
            UPDATE event_has_teams
            SET team_captain = $1
            WHERE id = $2`, newCaptainID, teamID)
		if err != nil {
			return fmt.Errorf("failed to transfer captain role: %v", err)
		}

		_, err = tx.Exec(`
            DELETE FROM event_has_users
            WHERE event_has_team_id = $1 AND user_id = $2`, teamID, playerID)
		if err != nil {
			return fmt.Errorf("failed to delete old captain: %v", err)
		}

		return nil
	}

	// If not captain, proceed with deletion
	query := `
        DELETE FROM event_has_users
        WHERE event_has_team_id = $1 AND user_id = $2`
	result, err := tx.Exec(query, teamID, playerID)
	if err != nil {
		return fmt.Errorf("failed to delete player: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("player not found in team")
	}

	return nil
}
