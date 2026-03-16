package controllers

import (
	"database/sql"
	"fmt"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type CityInfo struct {
	ID   int    `json:"id"`
	City string `json:"name"`
}

func GetTotalTournaments(c *gin.Context) {
	//extaract and decrypt the encrypted ID
	userId := DecryptParamId(c, "userId", true)
	if userId == 0 {
		return
	}
	err := error(nil)

	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	var timeCondition string

	if fromDate != "" && toDate != "" {
		timeCondition = fmt.Sprintf("AND e.to_date >= '%s' AND e.from_date <= '%s'", fromDate, toDate)
		// timeCondition = fmt.Sprintf("AND (e.from_date = '%s' AND e.to_date = '%s')", fromDate, toDate)
	} else if fromDate != "" {
		timeCondition = fmt.Sprintf("AND e.from_date >= '%s'", fromDate)
	} else if toDate != "" {
		timeCondition = fmt.Sprintf("AND e.to_date <= '%s'", toDate)
	} else {
		timeCondition = ""
	}

	// levelOfCompetition := c.DefaultQuery("level_of_competition_id", "")
	// var levelFilter string
	// if levelOfCompetition != "" {
	// 	levelFilter = "AND e.level_of_competition_id = " + levelOfCompetition
	// }

	eventIDEnc := c.Query("event_id")
	var eventFilter string

	if eventIDEnc != "" {
		eventID, err := crypto.NDecrypt(eventIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid event id", err)
			return
		}
		eventFilter = "AND e.id = " + strconv.FormatInt(eventID, 10)
	}

	locationwiseIDStr := c.Query("city_id")
	var locationFilter string
	var locationID int64

	if locationwiseIDStr != "" {
		var err error
		locationID, err = strconv.ParseInt(locationwiseIDStr, 10, 64)
		if err != nil {
			utils.HandleError(c, "Invalid city_id", err)
			return
		}
		locationFilter = "AND e.city_id = $2"
	}

	query := fmt.Sprintf(`
            SELECT DISTINCT e.id, e.name
            FROM event_has_users ehu
            JOIN events e ON e.id = ehu.event_id
            WHERE ehu.user_id = $1
            %s
            %s
            %s
        `, timeCondition, eventFilter, locationFilter)

	var rows *sql.Rows
	if locationFilter != "" {
		rows, err = database.DB.Query(query, userId, locationID)
	} else {
		rows, err = database.DB.Query(query, userId)
	}

	// rows, err := database.DB.Query(query, userId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch tournaments", err)
		return
	}
	defer rows.Close()

	var tournaments []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			utils.HandleError(c, "Error scanning tournament", err)
			return
		}
		tournaments = append(tournaments, map[string]interface{}{
			// "id": id,
			"id":   crypto.NEncrypt(int64(id)),
			"name": name,
		})
	}

	utils.HandleSuccess(c, "Tournaments fetched successfully", map[string]interface{}{
		"total_tournaments": len(tournaments),
		"tournaments":       tournaments,
	})
}

func GetUserCompetitionLevels(c *gin.Context) {
	//extaract and decrypt the encrypted ID
	userId := DecryptParamId(c, "userId", true)
	if userId == 0 {
		return
	}
	err := error(nil)

	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	var timeCondition string

	if fromDate != "" && toDate != "" {
		timeCondition = fmt.Sprintf("AND e.to_date >= '%s' AND e.from_date <= '%s'", fromDate, toDate)
	} else if fromDate != "" {
		timeCondition = fmt.Sprintf("AND e.from_date >= '%s'", fromDate)
	} else if toDate != "" {
		timeCondition = fmt.Sprintf("AND e.to_date <= '%s'", toDate)
	}

	eventIDEnc := c.Query("event_id")
	var eventFilter string
	if eventIDEnc != "" {
		eventID, err := crypto.NDecrypt(eventIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid event id", err)
			return
		}
		eventFilter = "AND e.id = " + strconv.FormatInt(eventID, 10)
	}

	query := fmt.Sprintf(`
		SELECT e.level_of_competition_id, loc.title, COUNT(*) as count
			FROM event_has_users ehu
			JOIN events e ON e.id = ehu.event_id %s
			JOIN level_of_competitions loc ON loc.id = e.level_of_competition_id
		WHERE ehu.user_id = $1 AND e.level_of_competition_id IS NOT NULL
		%s
		GROUP BY e.level_of_competition_id, loc.title
		ORDER BY count DESC
	`, timeCondition, eventFilter)

	rows, err := database.DB.Query(query, userId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch level of competition", err)
		return
	}
	defer rows.Close()

	var levels []map[string]interface{}
	for rows.Next() {
		var id int64
		var name string
		var count int
		if err := rows.Scan(&id, &name, &count); err != nil {
			utils.HandleError(c, "Error scanning levels", err)
			return
		}

		encID := crypto.NEncrypt(id)
		if encID == "" {
			utils.HandleError(c, "Failed to encrypt level of competition id", nil)
			return
		}

		levels = append(levels, map[string]interface{}{
			"id":    encID,
			"name":  name,
			"count": count,
		})
	}

	utils.HandleSuccess(c, "Levels fetched successfully", levels)
}

func GetCityofUser(c *gin.Context) {
	//extaract and decrypt the encrypted ID
	userId := DecryptParamId(c, "userId", true)
	if userId == 0 {
		return
	}
	err := error(nil)

	query := `
        SELECT DISTINCT ct.id, ct.city
        FROM event_has_users ehu
        JOIN events e ON e.id = ehu.event_id
        JOIN cities ct ON ct.id::text = e.city_id
        WHERE ehu.user_id = $1
    `

	rows, err := database.DB.Query(query, userId)
	if err != nil {
		utils.HandleError(c, "Database query failed", err)
		return
	}
	defer rows.Close()

	var cities []CityInfo
	for rows.Next() {
		var city CityInfo
		if err := rows.Scan(&city.ID, &city.City); err != nil {
			utils.HandleError(c, "Error scanning cities", err)
			return
		}
		cities = append(cities, city)
	}

	utils.HandleSuccess(c, "Cities fetched successfully.", cities)
}

func GetMatchStats(c *gin.Context) {
	//extaract and decrypt the encrypted ID
	userId := DecryptParamId(c, "userId", true)
	if userId == 0 {
		return
	}
	err := error(nil)

	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	var timeCondition string
	if fromDate != "" && toDate != "" {
		timeCondition = fmt.Sprintf("AND m.scheduled_date BETWEEN '%s' AND '%s'", fromDate, toDate)
	} else if fromDate != "" {
		timeCondition = fmt.Sprintf("AND m.scheduled_date >= '%s'", fromDate)
	} else if toDate != "" {
		timeCondition = fmt.Sprintf("AND m.scheduled_date <= '%s'", toDate)
	}

	eventIDEnc := c.Query("event_id")
	var eventFilter string

	if eventIDEnc != "" {
		eventID, err := crypto.NDecrypt(eventIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid event id", err)
			return
		}
		// eventFilter = "AND eht.id = " + strconv.FormatInt(eventID, 10)
		eventFilter = "AND e.id = " + strconv.FormatInt(eventID, 10)
	}

	//total matches
	/*totalMatchesQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT mht.match_id)
		FROM matches_has_teams mht
		JOIN matches m ON m.id = mht.match_id
		JOIN event_has_teams eht ON mht.team_id = eht.id
		JOIN event_has_users ehu ON ehu.event_has_team_id = eht.id
		WHERE ehu.user_id = $1 %s %s
	`, timeCondition, eventFilter)*/
	totalMatchesQuery := fmt.Sprintf(`
	SELECT COUNT(*) FROM (
	SELECT m.id
	FROM events e
	JOIN event_has_teams eht on e.id = eht.event_id
	JOIN event_has_users ehu on ehu.event_has_team_id = eht.id
	JOIN matches_has_teams mht on mht.team_id = eht.id
	JOIN matches m on m.id = mht.match_id
	WHERE ehu.user_id = $1
	%s %s
	group by m.id
) AS unique_matches; `, timeCondition, eventFilter)

	var totalMatches int
	err = database.DB.QueryRow(totalMatchesQuery, userId).Scan(&totalMatches)
	if err != nil {
		utils.HandleError(c, "Failed to fetch total matches count", err)
		return
	}

	//match results
	matchResultsQuery := fmt.Sprintf(`
		SELECT
			CASE
				WHEN mht.points = 2 THEN 'win'
				WHEN mht.points = 1 THEN 'draw'
				WHEN mht.points = 0 THEN 'loss'
				ELSE 'null'
			END AS result,
			COUNT(*) AS count
		FROM matches_has_teams mht
		JOIN matches m ON m.id = mht.match_id
		JOIN event_has_teams eht ON eht.id = mht.team_id
		JOIN events e ON e.id = eht.event_id
		JOIN event_has_users ehu ON ehu.event_has_team_id = eht.id
		WHERE ehu.user_id = $1 %s %s
		GROUP BY result
	`, timeCondition, eventFilter)

	rows, err := database.DB.Query(matchResultsQuery, userId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch match results", err)
		return
	}
	defer rows.Close()

	matchResults := map[string]int{
		"win":  0,
		"loss": 0,
		"draw": 0,
	}

	for rows.Next() {
		var result string
		var count int
		if err := rows.Scan(&result, &count); err != nil {
			utils.HandleError(c, "Error scanning match result", err)
			return
		}
		matchResults[result] = count
	}

	//overall score
	scoreQuery := fmt.Sprintf(`
	SELECT COALESCE(SUM(mths.points_scored), 0)
	FROM match_team_has_scores mths
	JOIN matches m ON m.id = mths.match_id
	JOIN event_has_teams eht ON mths.team_id = eht.id
	JOIN events e ON e.id = eht.event_id
	JOIN event_has_users ehu ON ehu.event_has_team_id = eht.id
	WHERE ehu.user_id = $1 %s %s
`, timeCondition, eventFilter)

	var overallScore int
	err = database.DB.QueryRow(scoreQuery, userId).Scan(&overallScore)
	if err != nil {
		utils.HandleError(c, "Failed to fetch overall score", err)
		return
	}

	response := map[string]interface{}{
		"total_matches": totalMatches,
		"match_results": matchResults,
		"overall_score": overallScore,
	}

	utils.HandleSuccess(c, "Match statistics fetched successfully", response)
}

func GetTotalGameParticipated(c *gin.Context) {
	//extaract and decrypt the encrypted ID
	userId := DecryptParamId(c, "userId", true)
	if userId == 0 {
		return
	}
	err := error(nil)

	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	var timeCondition string

	if fromDate != "" && toDate != "" {
		timeCondition = fmt.Sprintf("AND e.from_date >= '%s' AND e.to_date <= '%s'", fromDate, toDate)
		// timeCondition = fmt.Sprintf("AND (e.from_date = '%s' AND e.to_date = '%s')", fromDate, toDate)
	} else if fromDate != "" {
		timeCondition = fmt.Sprintf("AND e.from_date = '%s'", fromDate)
	} else if toDate != "" {
		timeCondition = fmt.Sprintf("AND e.to_date = '%s'", toDate)
	} else {
		timeCondition = ""
	}

	eventIDEnc := c.Query("event_id")
	var eventFilter string

	if eventIDEnc != "" {
		eventID, err := crypto.NDecrypt(eventIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid event id", err)
			return
		}
		eventFilter = "AND e.id = " + strconv.FormatInt(eventID, 10)
	}

	query := fmt.Sprintf(`
		    SELECT COUNT(*) FROM (
		        SELECT DISTINCT ehu.user_id, ehu.event_id, ehu.game_id
		        FROM event_has_users ehu
		        JOIN events e ON e.id = ehu.event_id
		        WHERE ehu.user_id = $1
		        %s
		        %s
		    ) AS unique_participations;
		`, timeCondition, eventFilter)

	var total int
	err = database.DB.QueryRow(query, userId).Scan(&total)
	if err != nil {
		utils.HandleError(c, "Failed to fetch game's participated", err)
		return
	}

	utils.HandleSuccess(c, "Game's participated fetched successfully", map[string]int{
		"game_participated": total,
	})
}

func GetUserGames(c *gin.Context) {
	var payload struct {
		UserEncId string `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.HandleError(c, "Invalid payload", err)
		return
	}

	// Decrypt user_id
	userId, err := crypto.NDecrypt(payload.UserEncId)
	if err != nil {
		utils.HandleError(c, "Failed to decrypt user ID", err)
		return
	}

	eventIDEnc := c.Query("event_id")
	var eventFilter string

	if eventIDEnc != "" {
		eventID, err := crypto.NDecrypt(eventIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid event id", err)
			return
		}
		eventFilter = "AND e.id = " + strconv.FormatInt(eventID, 10)
	}

	query := fmt.Sprintf(`
	SELECT DISTINCT g.id, g.game_name
	FROM event_has_users ehu
	JOIN games g ON g.id = ehu.game_id
	JOIN events e ON e.id = ehu.event_id
	WHERE ehu.user_id = $1
	%s
	`, eventFilter)

	rows, err := database.DB.Query(query, userId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch user games", err)
		return
	}
	defer rows.Close()

	type Game struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var games []Game
	for rows.Next() {
		var (
			id   int64
			name string
		)
		if err := rows.Scan(&id, &name); err != nil {
			utils.HandleError(c, "Failed to scan row", err)
			return
		}
		// encId, _ := crypto.Encrypt(id)
		encId := crypto.NEncrypt(id)
		games = append(games, Game{ID: encId, Name: name})
	}

	utils.HandleSuccess(c, "User games fetched successfully", games)
}

func GetDashboardGraphStats(c *gin.Context) {
	type RequestPayload struct {
		UserIDEnc string `json:"user_id"`
		GameIDEnc string `json:"game_id,omitempty"`
		TypeIDEnc string `json:"type_id,omitempty"`
	}

	var payload RequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.HandleError(c, "Invalid request payload", err)
		return
	}

	userId, err := crypto.NDecrypt(payload.UserIDEnc)
	if err != nil {
		utils.HandleError(c, "Failed to decrypt user ID", err)
		return
	}

	eventIDEnc := c.Query("event_id")
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	filter := ""
	args := []interface{}{userId}

	dateFilter := BuildDateFilterQuery(fromDate, toDate, "eht.")
	filter += " " + dateFilter

	if payload.GameIDEnc != "" {
		gameID, err := crypto.NDecrypt(payload.GameIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid game_id", err)
			return
		}
		filter += " AND eht.game_id = $" + fmt.Sprint(len(args)+1)
		args = append(args, gameID)
	}

	if payload.TypeIDEnc != "" {
		typeID, err := crypto.NDecrypt(payload.TypeIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid game_type_id", err)
			return
		}
		filter += " AND eht.game_type_id = $" + fmt.Sprint(len(args)+1)
		args = append(args, typeID)
	}

	if eventIDEnc != "" {
		eventID, err := crypto.NDecrypt(eventIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid event id", err)
			return
		}
		filter += " AND ehu.event_id = $" + fmt.Sprint(len(args)+1)
		args = append(args, eventID)
	}

	graphQuery := fmt.Sprintf(`
		WITH months AS (
			SELECT
					TO_CHAR(d, 'Month') AS month_name,
					TO_CHAR(d, 'YYYY-MM') AS month_key
			FROM generate_series(
					date_trunc('year', CURRENT_DATE),
					date_trunc('month', CURRENT_DATE),
					interval '1 month'
			) AS d
	),
	match_summary AS (
			SELECT
					TO_CHAR(DATE_TRUNC('month', m.scheduled_date), 'YYYY-MM') AS match_month,
					COUNT(DISTINCT mht.match_id) AS total_matches
			FROM matches_has_teams mht
			JOIN matches m ON m.id = mht.match_id
			JOIN event_has_teams eht ON mht.team_id = eht.id
			JOIN event_has_users ehu ON ehu.event_has_team_id = eht.id
			WHERE ehu.user_id = $1 %s
			GROUP BY match_month
	),
	score_summary AS (
			SELECT
					TO_CHAR(DATE_TRUNC('month', m.scheduled_date), 'YYYY-MM') AS score_month,
					COALESCE(SUM(mths.points_scored), 0) AS overall_score,
					COALESCE(SUM(CASE WHEN mths.set_number = 0 THEN mths.points_scored ELSE 0 END), 0) AS goals,
					COALESCE(SUM(CASE WHEN mths.set_number > 0 THEN mths.points_scored ELSE 0 END), 0) AS points
			FROM match_team_has_scores mths
			JOIN matches m ON m.id = mths.match_id
			JOIN event_has_teams eht ON mths.team_id = eht.id
			JOIN event_has_users ehu ON ehu.event_has_team_id = eht.id
			WHERE ehu.user_id = $1 %s
			GROUP BY score_month
	)
	SELECT
			m.month_name,
			COALESCE(ms.total_matches, 0) AS total_matches,
			COALESCE(ss.overall_score, 0) AS overall_score,
			COALESCE(ss.goals, 0) AS goals,
			COALESCE(ss.points, 0) AS points
	FROM months m
	LEFT JOIN match_summary ms ON ms.match_month = m.month_key
	LEFT JOIN score_summary ss ON ss.score_month = m.month_key
	ORDER BY m.month_key;
		`, filter, filter)

	graphRows, err := database.DB.Query(graphQuery, args...)
	if err != nil {
		utils.HandleError(c, "Failed to fetch graph data", err)
		return
	}
	defer graphRows.Close()

	var graphData []map[string]interface{}
	for graphRows.Next() {
		var month string
		var totalMatches, overallScore, goals, points int
		if err := graphRows.Scan(&month, &totalMatches, &overallScore, &goals, &points); err != nil {
			utils.HandleError(c, "Error scanning graph result", err)
			return
		}
		graphData = append(graphData, map[string]interface{}{
			"month":         strings.TrimSpace(month),
			"total_matches": totalMatches,
			"overall_score": overallScore,
			"points":        points,
			"goals":         goals,
		})
	}

	var gameTypeData interface{}
	if payload.GameIDEnc != "" {
		gameID, err := crypto.NDecrypt(payload.GameIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid game_id", err)
			return
		}

		args2 := []interface{}{userId, gameID}
		query := `
	SELECT
		gt.id,
		gt.name AS game_type_name,
		g.game_name,
		COUNT(*) AS total
	FROM event_has_teams eht
	JOIN events e ON e.id = eht.event_id
	JOIN games g ON g.id = eht.game_id
	JOIN event_has_users ehu ON eht.id = ehu.event_has_team_id
	JOIN game_has_types ght ON ght.game_id = eht.game_id AND ght.game_type_id = eht.game_type_id
	JOIN games_types gt ON gt.id = ght.game_type_id
	WHERE ehu.user_id = $1 AND g.id = $2 AND e.status = 'Active'
`

		argPos := 3
		if payload.TypeIDEnc != "" {
			typeID, err := crypto.NDecrypt(payload.TypeIDEnc)
			if err != nil {
				utils.HandleError(c, "Invalid game_type_id", err)
				return
			}
			query += fmt.Sprintf(" AND ght.id = $%d", argPos)
			args2 = append(args2, typeID)
			argPos++
		}

		if eventIDEnc != "" {
			eventID, err := crypto.NDecrypt(eventIDEnc)
			if err != nil {
				utils.HandleError(c, "Invalid event id", err)
				return
			}
			query += fmt.Sprintf(" AND ehu.event_id = $%d", argPos)
			args2 = append(args2, eventID)
		}

		query += `
	GROUP BY gt.id, gt.name, g.game_name
	ORDER BY g.game_name;`

		type GameTypeStat struct {
			ID           int64  `json:"-"`
			EncID        string `json:"id"`
			GameTypeName string `json:"game_type_name"`
			GameName     string `json:"game_name"`
			Total        int    `json:"total"`
		}

		rows, err := database.DB.Query(query, args2...)
		if err != nil {
			utils.HandleError(c, "Failed to fetch game type stats", err)
			return
		}
		defer rows.Close()

		var stats []GameTypeStat
		var totalGameCounts int
		for rows.Next() {
			var stat GameTypeStat
			if err := rows.Scan(&stat.ID, &stat.GameTypeName, &stat.GameName, &stat.Total); err != nil {
				utils.HandleError(c, "Failed to scan game type stat", err)
				return
			}
			// stat.EncID = crypto.NEncrypt(stat.ID)
			stat.EncID = crypto.NEncrypt(stat.ID)
			totalGameCounts += stat.Total
			stats = append(stats, stat)
		}

		if len(stats) > 0 {
			gameTypeData = map[string]interface{}{
				"total_game_count": totalGameCounts,
				"game_name":        stats[0].GameName,
				"game_types":       stats,
			}
		}
	}

	utils.HandleSuccess(c, "Dashboard stats fetched successfully", map[string]interface{}{
		"graph_data":     graphData,
		"game_type_data": gameTypeData,
	})
}

func BuildDateFilterQuery(fromDate, toDate, columnPrefix string) string {
	if fromDate != "" && toDate != "" {
		return fmt.Sprintf("AND %screated_at::date BETWEEN '%s' AND '%s'", columnPrefix, fromDate, toDate)
	} else if fromDate != "" {
		return fmt.Sprintf("AND %screated_at::date >= '%s'", columnPrefix, fromDate)
	} else if toDate != "" {
		return fmt.Sprintf("AND %screated_at::date <= '%s'", columnPrefix, toDate)
	}
	return ""
}

func GetUserCityNamesFromEvents(c *gin.Context) {
	//extaract and decrypt the encrypted ID
	userId := DecryptParamId(c, "userId", true)
	if userId == 0 {
		return
	}
	err := error(nil)

	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	var timeCondition string

	if fromDate != "" && toDate != "" {
		timeCondition = fmt.Sprintf("AND e.to_date >= '%s' AND e.from_date <= '%s'", fromDate, toDate)
	} else if fromDate != "" {
		timeCondition = fmt.Sprintf("AND e.from_date >= '%s'", fromDate)
	} else if toDate != "" {
		timeCondition = fmt.Sprintf("AND e.to_date <= '%s'", toDate)
	} else {
		timeCondition = ""
	}

	eventIDEnc := c.Query("event_id")
	var eventFilter string
	if eventIDEnc != "" {
		eventID, err := crypto.NDecrypt(eventIDEnc)
		if err != nil {
			utils.HandleError(c, "Invalid event id", err)
			return
		}
		eventFilter = "AND e.id = " + strconv.FormatInt(eventID, 10)
	}

	query := fmt.Sprintf(`
        SELECT ct.id, ct.city, COUNT(*) as player_count
        FROM event_has_users ehu
        JOIN events e ON e.id = ehu.event_id %s
        JOIN cities ct ON ct.id::text = e.city_id
        WHERE ehu.user_id = $1
        %s
        GROUP BY ct.id, ct.city
    `, " "+timeCondition, eventFilter)

	rows, err := database.DB.Query(query, userId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch city names", err)
		return
	}
	defer rows.Close()

	var cities []map[string]interface{}
	for rows.Next() {
		var cityID int64
		var cityName string
		var count int
		if err := rows.Scan(&cityID, &cityName, &count); err != nil {
			utils.HandleError(c, "Error scanning city data", err)
			return
		}

		encID := crypto.NEncrypt(cityID)
		if encID == "" {
			utils.HandleError(c, "Failed to encrypt city ID", nil)
			return
		}

		cities = append(cities, map[string]interface{}{
			"id":    encID,
			"name":  cityName,
			"count": count,
		})
	}

	utils.HandleSuccess(c, "Cities fetched successfully", cities)
}
