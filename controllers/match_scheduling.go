package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ScheduleMatches(c *gin.Context) {
	var teamDetails models.TeamArray
	//1.  Get Teams Array
	if err := c.ShouldBindBodyWithJSON(&teamDetails); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid Input",
			"data":    err.Error(),
		})
		return
	}
	roundNumber := c.Query("roundNo")
	roundNo, err := strconv.ParseInt(roundNumber, 10, 64)
	if err != nil {
		fmt.Errorf("error converting round to integer")
	}
	//2. Validate
	validationMessage := ValidateStruct(&teamDetails)
	if validationMessage != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": validationMessage,
		})
		return
	}

	// 2. Decrypt Team Ids
	var decryptedTeams []int64
	for _, encryptedTeamId := range teamDetails.Teams {
		decryptedTeamId, err := crypto.NDecrypt(encryptedTeamId)
		if err != nil {
			utils.HandleError(c, "Couldn't decrypt team Ids")
			return
		}
		decryptedTeams = append(decryptedTeams, decryptedTeamId)
	}

	// 3. Get All Event Related Ids(event_has_game_type id, event id, game id, game_type id and type of tournament) check if decryptedTeams exist in same event
	eventInfo, err := models.VerifyTeamDetails(decryptedTeams)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	// var filteredTeams []int64
	if roundNo != 0 {
		_, err := models.HandleMatches(eventInfo.EventHasGameTypeID, eventInfo.CreatedBy, roundNo, eventInfo.TypeOfTournament, decryptedTeams, eventInfo.GameName)
		if err != nil {
			utils.HandleError(c, "Unable to generate matches.", err)
			return
		}
		latestRound, err := models.GetLatestRound(eventInfo.EventHasGameTypeID)
		if err != nil {
			utils.HandleError(c, err.Error())
			return
		}
		latestMatches, err := models.FetchLatestMatchesWithTeams(eventInfo.EventHasGameTypeID, latestRound)
		if err != nil {
			utils.HandleError(c, "Failed to fecth the latest matches", err)
			return
		}
		if eventInfo.TypeOfTournament == "Atheletics" {
			if err != nil {
				utils.HandleError(c, "Error fetching athletics matches", err)
				return
			}

			utils.HandleSuccess(c, "Matches generated successfully", gin.H{
				"matches":         latestMatches,
				"round_no":        latestRound,
				"tournament_type": "Athletics",
				"is_last_round":   false,
			})
			return
		} else {
			isMatchAlreadyExist, err := models.IsMatchAlreadyScheduled(eventInfo.EventHasGameTypeID)
			if err != nil {
				utils.HandleError(c, err.Error())
			}
			// 5. Final response
			utils.HandleSuccess(c, "Matches generated successfully", gin.H{
				"is_match_already_exist": isMatchAlreadyExist,
				"round_no":               latestRound,
				"matches":                latestMatches,
				"tournament_type":        eventInfo.TypeOfTournament,
				"is_last_round":          false,
			})
		}
	}
}

func AddMatchInfo(c *gin.Context) {
	// Decrypt the encrypted ID
	decryptMatchId := DecryptParamId(c, "id", true)
	if decryptMatchId == 0 {
		return
	}
	err := error(nil)
	var matchData models.MatchData

	if err := c.ShouldBindJSON(&matchData); err != nil {
		utils.HandleError(c, "Invalid Input", err)
		return
	}

	validationError := ValidateStruct(&matchData)
	if validationError != "" {
		utils.HandleError(c, validationError)
		return
	}

	start, err1 := time.Parse("15:04", matchData.StartTime)
	end, err2 := time.Parse("15:04", matchData.EndTime)
	if err1 == nil && err2 == nil && !start.Before(end) {
		utils.HandleInvalidEntries(c, "Start time must be before end time")
		return
	}

	if matchData.VenueLink != "" && !utils.IsValidGoogleMapsURL(matchData.VenueLink) {
		utils.HandleInvalidEntries(c, "Please provide a valid venue link")
		return
	}

	matchData.MatchId = decryptMatchId
	var eventType int
	err = database.DB.QueryRow(`SELECT event_has_game_types FROM matches WHERE id = $1`, matchData.MatchId).Scan(&eventType)
	if err != nil {
		fmt.Errorf("error retrieving event info: %v", err)
	}
	var venueConflictId int
	venueConflictQuery := `
		SELECT id FROM matches
		WHERE event_has_game_types = $1 AND id != $2
		AND scheduled_date = $3
		AND start_time < $5 AND end_time > $4
		AND venue ILIKE $6
		LIMIT 1;
		`
	err = database.DB.QueryRow(venueConflictQuery, eventType, matchData.MatchId, matchData.ScheduledDate, matchData.StartTime, matchData.EndTime, matchData.Venue).Scan(&venueConflictId)
	if err == nil {
		utils.HandleError(c, fmt.Sprintf("Another match is already scheduled at '%s' on %s at %s.", matchData.Venue, matchData.ScheduledDate, matchData.StartTime))
		return
	} else if err != sql.ErrNoRows {
		fmt.Errorf("error checking venue conflict: %v", err)
	}
	var matchConflictId int
	matchConflictQuery := `
		SELECT id FROM matches 
		WHERE event_has_game_types = $1 AND id != $2
		AND match_name ILIKE $3
		LIMIT 1;
		`
	err = database.DB.QueryRow(matchConflictQuery, eventType, matchData.MatchId, matchData.MatchName).Scan(&matchConflictId)
	if err == nil {
		utils.HandleError(c, fmt.Sprintf("Another match with name '%s' already exist.", matchData.MatchName))
		return
	} else if err != sql.ErrNoRows {
		fmt.Errorf("error checking match conflict: %v", err)
	}
	updatedMatchData, err := models.EditMatchData(&matchData)
	if err != nil {
		utils.HandleError(c, "Unable to update match data", err)
		return
	}
	updatedMatchData.MatchEncId = crypto.NEncrypt(decryptMatchId)
	utils.HandleSuccess(c, "Match data updated successfully", updatedMatchData)
}

func GetMatchDataById(c *gin.Context) {
	// fmt.Println("sentry test")
	// sentry.CaptureException(fmt.Errorf("test")) //for stage testing
	// Decrypt the encrypted ID
	matchId := DecryptParamId(c, "id", true)
	if matchId == 0 {
		return
	}

	match, err := models.GetMatchById(matchId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	match.MatchEncId = crypto.NEncrypt(match.MatchId)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Matches retrieved successfully",
		"data":    match,
	})

}

func GetTeamPlayers(c *gin.Context) {
	var teamEncIds []string
	var teamsArr []any
	if err := c.ShouldBindBodyWithJSON(&teamEncIds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "teamId parameter is missing in the request.",
			"data":    err.Error(),
		})
		return
	}
	for _, teamEncId := range teamEncIds {
		teamId, err := crypto.NDecrypt(teamEncId)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting team one id ---->%v", err))
		}

		members, err := models.GetMembersByTeamId(teamId)
		if err != nil {
			utils.HandleError(c, "Error fetching members", fmt.Errorf("error fetching members for team ---> %v", err))
		}

		var EncMembers []any

		for i := range members {
			encId := crypto.NEncrypt(members[i].Id)
			EncMembers = append(EncMembers, gin.H{
				"Id":   encId,
				"Name": members[i].Name,
			})
		}

		teamsArr = append(teamsArr, gin.H{
			"team_id": teamEncId,
			"members": EncMembers,
		})
		// teamIds = append(teamIds, teamId)
	}
	utils.HandleSuccess(c, "successfully retrieved team members of both teams", teamsArr)
}
