package controllers

import (
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetMatchesByEvent(c *gin.Context) {
	eventID := DecryptParamId(c, "event_id", true)
	if eventID == 0 {
		return
	}
	gameID := DecryptParamId(c, "game_id", true)
	if gameID == 0 {
		return
	}
	participantID := DecryptParamId(c, "participant_id", false)
	gameTypeIds := c.Param("game_type_id")
	categoryIds := c.Param("category_ids")

	// fmt.Println("event id:", eventId)
	// fmt.Println("game id:", gameId)
	// fmt.Println("game type id:", gameTypeIds)
	// fmt.Println("game type category id:", categoryIds)

	// Check for missing query parameters
	if gameTypeIds == "" || categoryIds == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Missing required query parameters.",
		})
		return
	}

	// Parse game_type_ids as a list
	gameTypeIDList := strings.Split(gameTypeIds, ",")
	var gameTypeIDArray []int64
	for _, id := range gameTypeIDList {
		gameTypeID, err := crypto.NDecrypt(id)
		if err != nil {
			utils.HandleError(c, "Invalid game type ID", err)
			return
		}
		gameTypeIDArray = append(gameTypeIDArray, gameTypeID)
	}

	// category_ids
	categoryIDList := strings.Split(categoryIds, ",")
	var categoryIDArray []int64
	for _, id := range categoryIDList {
		decryptedID, err := crypto.NDecrypt(id)
		if err != nil {
			utils.HandleError(c, "Invalid category ID", err)
			return
		}
		categoryIDArray = append(categoryIDArray, decryptedID)
	}

	tournamentType, err := models.GetTournamentTypeByEventAndGame(eventID, gameID)
	if err != nil {
		utils.HandleError(c, "Failed to get tournament type", err)
		return
	}

	var matches []models.Match
	// Fetch matches
	if participantID == 0 {
		matches, err = models.GetMatchesByEventGameAndType(eventID, gameID, gameTypeIDArray, categoryIDArray)
	} else {

		TeamId, err1 := models.GetTeamIdByUserAndGame(eventID, gameID, participantID)
		if err1 != nil {
			utils.HandleError(c, "Error fetching teamId", err)
			return
		}

		matches, err = models.GetMatchesByEventGameAndType(eventID, gameID, gameTypeIDArray, categoryIDArray, TeamId)
	}
	if err != nil {
		utils.HandleError(c, "Failed to retrieve matches", err)
		return
	}

	// Call the model to fetch matches
	// matches, err := models.GetMatchesByEventGameAndType(eventID, gameID, gameTypeIDArray)
	// if err != nil {
	// 	utils.HandleError(c, "Failed to retrieve matches", err)
	// 	return
	// }

	// If no matches found, return empty response
	if len(matches) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "No matches found",
			"data":    []interface{}{},
		})
		return
	}

	// Loop through matches to enrich data
	for i := range matches {
		points, err := models.GetMatchPoints(matches[i].MatchId)
		if err != nil {
			utils.HandleError(c, "Failed to fetch match points", err)
			return
		}

		// Set stadium name
		matches[i].StadiumName = matches[i].MatchName

		// Calculate winner or draw
		team1Points := points[matches[i].Team1ID]
		team2Points := points[matches[i].Team2ID]

		var winTeamId *string
		var isDraw bool

		if team1Points > team2Points {
			winStr := strconv.FormatInt(matches[i].Team1ID, 10)
			winTeamId = &winStr
		} else if team2Points > team1Points {
			winStr := strconv.FormatInt(matches[i].Team2ID, 10)
			winTeamId = &winStr
		} else {
			isDraw = true
		}

		matches[i].WinTeamId = winTeamId

		if winTeamId != nil {
			winIDInt, err := strconv.ParseInt(*winTeamId, 10, 64)
			if err != nil {
				utils.HandleError(c, "Failed to convert win team ID", err)
				return
			}
			matches[i].WinTeamEncId = crypto.NEncrypt(winIDInt)
		}

		matches[i].IsDraw = &isDraw

		// Encrypt IDs
		matches[i].MatchEncId = crypto.NEncrypt(matches[i].MatchId)
		matches[i].Team1EncId = crypto.NEncrypt(matches[i].Team1ID)
		matches[i].Team2EncId = crypto.NEncrypt(matches[i].Team2ID)
	}

	// Inject tournament type
	for i := range matches {
		matches[i].TournamentType = tournamentType
	}

	utils.HandleSuccess(c, "Matches retrieved successfully", matches)
}

func GetAllMatches(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")
	dir := c.DefaultQuery("dir", "DESC")
	status := c.Query("status")
	offset := (page - 1) * limit
	scheduledDate := c.Query("scheduled_date")

	// Fetch data from the model with status filtering
	totalRecords, matches, err := models.GetMatches(search, sort, dir, status, scheduledDate, int64(limit), int64(offset))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"data":    err,
			"message": "Failed to fetch matches.",
		})
		return
	}

	// Encrypt the IDs of all events records
	encryptedMatches := make([]map[string]interface{}, 0)
	for _, match := range matches {
		match.MatchEncId = crypto.NEncrypt(match.MatchId)
		match.Team1EncId = crypto.NEncrypt(match.Team1ID)
		match.Team2EncId = crypto.NEncrypt(match.Team2ID)
		match.EventEncId = crypto.NEncrypt(match.EventId)
		match.GameEncId = crypto.NEncrypt(match.GameId)
		match.GameTypeEncId = crypto.NEncrypt(match.GameTypeId)
		// Check if event logo is valid
		team1LogoPath := match.Team1Logo
		defaultTeam1LogoPath := "public/static/staticTeamLogo.jpg"

		if match.Team1Logo == "" || !fileExists(team1LogoPath) {
			team1LogoPath = defaultTeam1LogoPath
		}

		team2LogoPath := match.Team2Logo
		// fmt.Println("TEam 2 Logo : ", team2LogoPath)
		defaultTeam2LogoPath := "public/static/staticTeamLogo.jpg"

		if match.Team2Logo == "" || !fileExists(team2LogoPath) {
			team2LogoPath = defaultTeam2LogoPath
		}
		// fmt.Println("TEam 2 Logo fbfd: ", team2LogoPath)
		encryptedMatches = append(encryptedMatches, map[string]interface{}{
			"match_id": match.MatchEncId,
			// "event_has_game_type_id": match.EventHasGameTypeId,
			"event_id":       match.EventEncId,
			"game_id":        match.GameEncId,
			"game_type_id":   match.GameTypeEncId,
			"match_name":     match.MatchName,
			"scheduled_date": match.ScheduledDate,
			"venue":          match.StadiumName,
			"venue_link":     match.VenueLink,
			"start_time":     match.StartTime,
			"is_draw":        match.IsDraw,
			"team1_id":       match.Team1EncId,
			"team1_name":     match.Team1Name,
			"team1_logo":     team1LogoPath,
			"team2_id":       match.Team2EncId,
			"team2_name":     match.Team2Name,
			"team2_logo":     team2LogoPath,
		})
	}

	// Respond with paginated and filtered data
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"totalRecords": totalRecords,
			"matches":      encryptedMatches,
		},
		"message": "Fetched all matches successfully.",
	})
}

func GetTotalMatchCount(c *gin.Context) {
	status := c.Query("status")
	scheduledDate := c.Query("scheduled_date")

	totalCount, err := models.GetTotalMatchCount(status, scheduledDate)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Failed to fetch total match count.",
			"data":    "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Total match count fetched successfully.",
		"data": gin.H{
			"totalMatches": totalCount,
		},
	})
}
