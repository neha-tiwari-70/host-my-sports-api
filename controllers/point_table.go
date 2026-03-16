package controllers

import (
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetPointTable(c *gin.Context) {
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

	// Check for missing required parameters
	if gameTypeIds == "" || categoryIds == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Missing required query parameters.",
		})
		return
	}

	// Decrypt game type IDs
	gameTypeIDList := strings.Split(gameTypeIds, ",")
	var gameTypeIDArray []int64
	for _, id := range gameTypeIDList {
		gtid, err := crypto.NDecrypt(id)
		if err != nil {
			utils.HandleError(c, "Invalid game type ID", err)
			return
		}
		gameTypeIDArray = append(gameTypeIDArray, gtid)
	}

	// Decrypt category IDs
	categoryIDList := strings.Split(categoryIds, ",")
	var categoryIDArray []int64
	for _, id := range categoryIDList {
		cid, err := crypto.NDecrypt(id)
		if err != nil {
			utils.HandleError(c, "Invalid category ID", err)
			return
		}
		categoryIDArray = append(categoryIDArray, cid)
	}

	tournamentType, err := models.GetTournamentTypeByEventAndGame(eventID, gameID)
	if err != nil {
		utils.HandleError(c, "Failed to get tournament type", err)
		return
	}

	var pointTable []models.PointTable

	// Call the model function
	if participantID == 0 {
		pointTable, err = models.GetPointTable(eventID, gameID, gameTypeIDArray, categoryIDArray)
	} else {
		TeamId, err1 := models.GetTeamIdByUserAndGame(eventID, gameID, participantID)
		if err1 != nil {
			utils.HandleError(c, "Error fetching teamId", err)
			return
		}
		pointTable, err = models.GetPointTable(eventID, gameID, gameTypeIDArray, categoryIDArray, TeamId)
	}
	if err != nil {
		utils.HandleError(c, "Failed to fetch point table", err)
		return
	}

	// Inject tournament type
	for i := range pointTable {
		pointTable[i].TournamentType = tournamentType
	}

	if len(pointTable) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "No points to show",
			"data":    []interface{}{},
		})
		return
	}

	utils.HandleSuccess(c, "Point table retrieved successfully", pointTable)
}
