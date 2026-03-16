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

func GetMatchResult(c *gin.Context) {
	eventID := DecryptParamId(c, "event_id", true)
	if eventID == 0 {
		return
	}
	gameID := DecryptParamId(c, "game_id", true)
	if gameID == 0 {
		return
	}
	categoryID := DecryptParamId(c, "category_id", true)
	if categoryID == 0 {
		return
	}
	participantID := DecryptParamId(c, "participant_id", false)
	gameTypeIds := c.Param("game_type_id")

	if gameTypeIds == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Missing required query parameters.",
		})
		return
	}

	tournament_type, err := models.GetTournamentTypeByEventAndGame(eventID, gameID)
	if err != nil {
		utils.HandleError(c, "Unable to fetch the tournament type")
		return
	}

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

	tournamentType, err := models.GetTournamentTypeByEventAndGame(eventID, gameID)
	if err != nil {
		utils.HandleError(c, "Failed to get tournament type", err)
		return
	}

	if tournamentType == "Atheletics" || tournamentType == "Time Trial" || tournamentType == "Mass Start" || tournamentType == "Relay" || tournamentType == "Fun Ride" || tournamentType == "Endurance" {
		var pointTable []models.PointTable
		var teamID int64

		if participantID == 0 {

			teamID, err = models.GetTeamIdByUserAndGame(eventID, gameID, participantID)
			if err != nil {
				utils.HandleError(c, "Error fetching team ID", err)
				return
			}

			pointTable, err = models.GetPointTable(eventID, gameID, gameTypeIDArray, []int64{categoryID}, teamID)
		} else {
			pointTable, err = models.GetPointTable(eventID, gameID, gameTypeIDArray, []int64{categoryID})
		}

		if err != nil {
			utils.HandleError(c, "Failed to fetch point table", err)
			return
		}

		type AtheleticsResult struct {
			TeamID         string `json:"team_id"`
			TeamName       string `json:"team_name"`
			MatchPoint     int64  `json:"match_point"`
			ScoredAt       string `json:"scored_at"`
			TournamentType string `json:"tournament_type"`
		}

		var trimmed []AtheleticsResult
		for _, row := range pointTable {
			trimmed = append(trimmed, AtheleticsResult{
				TeamID:         row.TeamID, // already encrypted
				TeamName:       row.TeamName,
				MatchPoint:     row.ScoredPoints,
				ScoredAt:       row.ScoredAt,
				TournamentType: tournamentType,
			})
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Match's result retrieved successfully", "data": trimmed, "tournament_type": tournament_type})

		return
	}

	// Non-athletics match results
	var matchResults []models.MatchResultResponse
	if participantID == 0 {
		matchResults, err = models.GetMatchResultsByEventGameAndType(eventID, gameID, categoryID, gameTypeIDArray)
	} else {

		TeamId, err1 := models.GetTeamIdByUserAndGame(eventID, gameID, participantID)
		if err1 != nil {
			utils.HandleError(c, "Error fetching team ID", err1)
			return
		}
		matchResults, err = models.GetMatchResultsByEventGameAndType(eventID, gameID, categoryID, gameTypeIDArray, TeamId)
	}

	if err != nil {
		if err.Error() == "no match results found" {
			utils.HandleSuccess(c, "No match results found", []any{})
		} else {
			utils.HandleError(c, "Failed to fetch match results", err)
		}
		return
	}

	for i := range matchResults {
		matchIDInt, err := strconv.ParseInt(matchResults[i].MatchEncId, 10, 64)
		if err != nil {
			utils.HandleError(c, "Failed to parse match ID", err)
			return
		}
		matchResults[i].MatchEncId = crypto.NEncrypt(matchIDInt)
		matchResults[i].TournamentType = tournamentType
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Match's result retrieved successfully", "data": matchResults, "tournament_type": tournament_type})
	// utils.HandleSuccess(c, "Match's result retrieved successfully", matchResults)
}
