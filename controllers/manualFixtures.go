package controllers

import (
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

func HandleUpdateMatchTeams(c *gin.Context) {
	var req struct {
		Matches []struct {
			MatchId string   `json:"match_id"`
			Teams   []string `json:"teams"` // encrypted team IDs
		} `json:"updated_matches"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request"})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		utils.HandleError(c, "Failed to perfrom drag and drop.", err)
		return
	}
	defer tx.Rollback()

	for _, m := range req.Matches {
		matchID, err := crypto.NDecrypt(m.MatchId)
		if err != nil {
			utils.HandleError(c, "Failed to perfrom drag and drop", err)
			return
		}

		if err := models.UpdateMatchTeams(tx, matchID, m.Teams); err != nil {
			utils.HandleError(c, "Failed to save changes", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		utils.HandleError(c, "Failed to perfrom drag and drop")
		return
	}
	utils.HandleSuccess(c, "Changes saved successfully", err)
}
