package controllers

import (
	"fmt"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

// Request payload structure
type SaveGroupsRequest struct {
	Groups []struct {
		EncTeamID string `json:"team_id"`
		TeamID    int64  `json:"-"`
		GroupNo   int    `json:"group_no"`
		TeamName  string `json:"team_name"`
	} `json:"groups"`
}

func SaveGroupsHandler(c *gin.Context) {
	var req SaveGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}
	var err error
	for i, _ := range req.Groups {
		req.Groups[i].TeamID, err = crypto.NDecrypt(req.Groups[i].EncTeamID)
		if err != nil {
			utils.HandleError(c, "failed to decrypt ", err)
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		utils.HandleError(c, "Failed to initiate save groups.")
		return
	}
	defer tx.Rollback()

	query := `UPDATE event_has_teams SET group_no = $1, updated_at = NOW() WHERE id = $2`

	for _, g := range req.Groups {
		_, err := tx.Exec(query, g.GroupNo, g.TeamID)
		fmt.Println("Team name ", g.TeamName, "group changed to ", g.GroupNo)
		if err != nil {
			utils.HandleError(c, "Failed to update groups.", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		utils.HandleError(c, "Failed to save groups.", err)
		return
	}

	utils.HandleSuccess(c, "Groups saved successfully.")
}

func LockGroups(c *gin.Context) {
	var req struct {
		EventID    int64 `json:"event_id"`
		GameTypeID int64 `json:"game_type_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.DB.Exec(`UPDATE event_has_game_types SET is_locked = 1 WHERE event_id = $1 AND game_type_id = $2`,
		req.EventID, req.GameTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "locked"})
}
