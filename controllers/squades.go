package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetSquadStats(c *gin.Context) {
	err := error(nil)
	eventIDInt := DecryptParamId(c, "event_id", true)
	if eventIDInt == 0 {
		return
	}
	gameIDInt := DecryptParamId(c, "game_id", true)
	if gameIDInt == 0 {
		return
	}
	participantID := DecryptParamId(c, "participant_id", false)
	gameTypeIds := c.Param("game_type_id")
	categoryIds := c.Param("category_ids")

	// Check for missing query parameters
	if gameTypeIds == "" || categoryIds == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Missing required query parameters.",
		})
		return
	}

	// Decrypt game_type_id values
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

	// Decrypt category_ids
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

	// Fetch stats

	var squadStats []models.SquadStats
	// Fetch matches
	if participantID == 0 {
		squadStats, err = models.GetSquadStats(eventIDInt, gameIDInt, gameTypeIDArray, categoryIDArray)
	} else {
		TeamId, err1 := models.GetTeamIdByUserAndGame(eventIDInt, gameIDInt, participantID)
		if err1 != nil {
			utils.HandleError(c, "Error fetching teamId", err)
			return
		}
		squadStats, err = models.GetSquadStats(eventIDInt, gameIDInt, gameTypeIDArray, categoryIDArray, TeamId)
	}

	if err != nil {
		// If no match data found, fetch basic team info
		if strings.Contains(err.Error(), "no squad stats found") {
			squadStats, err = models.GetSquadStatsWithoutMatches(eventIDInt, gameIDInt, gameTypeIDArray, categoryIDArray)
			if err != nil {
				if strings.Contains(err.Error(), "no squad stats found") {
					utils.HandleInvalidEntries(c, "Failed to fetch squad stats", err)
					return
				}
				utils.HandleError(c, "Failed to fetch squad stats", err)
				return
			}
		} else {
			utils.HandleError(c, "Failed to fetch squad stats", err)
			return
		}
	}

	for i := range squadStats {
		for j := range squadStats[i].Players {
			squadStats[i].Players[j].PlayerEncID = crypto.NEncrypt(squadStats[i].Players[j].PlayerID)
		}
	}

	type EventData struct {
		EncEventId    string `json:"event_id"`
		EncGameId     string `json:"game_id"`
		EncGameTypeId string `json:"game_type_id"`
		EncAgeGroupId string `json:"age_group_id"`
	}

	var event EventData
	event.EncEventId = crypto.NEncrypt(eventIDInt)
	event.EncGameId = crypto.NEncrypt(gameIDInt)
	event.EncGameTypeId = gameTypeIds
	event.EncAgeGroupId = categoryIds

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"eventData": event,
		"data":      squadStats,
	})
}

func ReplacePlayerFromTeam(c *gin.Context) {
	type PlayerReplacement struct {
		DeletePlayerID    string `json:"delete_player_id"`
		NewPlayerUserCode string `json:"new_player_user_code"`
		TshirtSize        string `json:"tshirt_size"`
		IsCaptain         bool   `json:"is_captain"`
	}

	type UpdateSquadRequest struct {
		TeamEncID        string              `json:"team_id" binding:"required"`
		IsCaptainChanged bool                `json:"is_captain_changed"`
		Players          []PlayerReplacement `json:"players"`
	}

	var req UpdateSquadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	teamID, err := crypto.NDecrypt(req.TeamEncID)
	if err != nil {
		utils.HandleError(c, "Invalid team ID", err)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		utils.HandleError(c, "Failed to start transaction", err)
		return
	}

	var eventID, gameID, gameTypeID, ageGroupID int64
	err = tx.QueryRow(`
	        SELECT event_id, game_id, game_type_id, age_group_id
	        FROM event_has_teams
	        WHERE id = $1`, teamID).Scan(&eventID, &gameID, &gameTypeID, &ageGroupID)
	if err != nil {
		tx.Rollback()
		utils.HandleError(c, "Failed to fetch team details", err)
		return
	}

	var eventHasGameID int64
	err = tx.QueryRow(`
        SELECT id
        FROM event_has_games
        WHERE event_id = $1 AND game_id = $2
    `, eventID, gameID).Scan(&eventHasGameID)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			utils.HandleError(c, "No matching event_has_games record found for event_id and game_id", err)
			return
		}
		utils.HandleError(c, "Failed to fetch event_has_games ID", err)
		return
	}

	var minPlayer, maxPlayer int
	var ehGameTypeId int64
	err = tx.QueryRow(`
        SELECT id, min_player, max_player
        FROM event_has_game_types
        WHERE event_has_game_id = $1 AND game_type_id = $2 AND age_group_id = $3
    `, eventHasGameID, gameTypeID, ageGroupID).Scan(&ehGameTypeId, &minPlayer, &maxPlayer)
	if minPlayer == 0 {
		minPlayer = 1
	}
	if maxPlayer == 0 {
		maxPlayer = 1
	}
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			utils.HandleError(c, "No matching event_has_game_types record found", err)
			return
		}
		utils.HandleError(c, "Failed to fetch event_has_game_types details", err)
		return
	}

	if minPlayer <= 0 || maxPlayer < minPlayer {
		tx.Rollback()
		utils.HandleInvalidEntries(c, "Invalid min_player or max_player configuration", nil)
		return
	}

	var currentPlayerCount int
	err = tx.QueryRow(`
        SELECT COUNT(*)
        FROM event_has_users
        WHERE event_has_team_id = $1
    `, teamID).Scan(&currentPlayerCount)
	if err != nil {
		tx.Rollback()
		utils.HandleError(c, "Failed to count team members", err)
		return
	}

	for _, p := range req.Players {
		if p.DeletePlayerID != "" {
			playerID, err := crypto.NDecrypt(p.DeletePlayerID)
			if err != nil {
				tx.Rollback()
				utils.HandleError(c, "Invalid player ID to delete", err)
				return
			}

			var newPlayerID int64
			if p.NewPlayerUserCode != "" {
				user, _, err := models.GetUserByUserCode(p.NewPlayerUserCode, true, true)
				if err == nil {
					newPlayerID = int64(user.ID)
				}
			}

			err = models.ReplacePlayerFromTeam(teamID, playerID, tx, newPlayerID)
			if err != nil {
				tx.Rollback()
				utils.HandleError(c, "Failed to replace player: "+err.Error(), err)
				return
			}

			currentPlayerCount--
		}

		if p.NewPlayerUserCode != "" {
			user, age, err := models.GetUserByUserCode(p.NewPlayerUserCode, true, true)
			if err != nil {
				tx.Rollback()
				if err == sql.ErrNoRows {
					utils.HandleInvalidEntries(c, "User not found", err)
					return
				}
				utils.HandleError(c, "Failed to fetch user", err)
				return
			}

			if user.RoleSlug == "organization" {
				tx.Rollback()
				utils.HandleInvalidEntries(c, "Organizations are not eligible to register as participant", nil)
				return
			}

			if user.Details == nil || user.Details.Gender == nil {
				tx.Rollback()
				utils.HandleInvalidEntries(c, "Player gender is missing", nil)
				return
			}

			isAgeValid, isGenderValid, err := models.ValidateAgeAndGender(ehGameTypeId, age, *user.Details.Gender)
			if err != nil {
				tx.Rollback()
				utils.HandleError(c, "Failed to validate age or gender", err)
				return
			}
			if !isGenderValid {
				tx.Rollback()
				utils.HandleInvalidEntries(c, fmt.Sprintf("%s players are not allowed in this team", *user.Details.Gender), nil)
				return
			}
			if !isAgeValid {
				tx.Rollback()
				utils.HandleInvalidEntries(c, "Player does not meet age requirements", nil)
				return
			}

			tshirtSizes := map[int64]string{int64(user.ID): p.TshirtSize}
			err = models.InsertTeamMembers([]int64{int64(user.ID)}, tshirtSizes, eventID, gameID, teamID, ehGameTypeId, tx)
			if err != nil {
				tx.Rollback()
				utils.HandleError(c, "Failed to add new player: ", err)
				return
			}
			currentPlayerCount++
		}
	}

	if !(currentPlayerCount <= maxPlayer) {
		tx.Rollback()
		utils.HandleInvalidEntries(c, fmt.Sprintf("Cannot add player: team cannot exceed %d players", maxPlayer), nil)
		return
	}

	if !(currentPlayerCount >= minPlayer) {
		tx.Rollback()
		utils.HandleInvalidEntries(c, fmt.Sprintf("Cannot delete player: team must have at least %d players", minPlayer), nil)
		return
	}

	if err := tx.Commit(); err != nil {
		utils.HandleError(c, "Failed to commit transaction", err)
		return
	}

	var squadStats []models.SquadStats
	hasMatches := models.CheckIfEventHasMatches(eventID, gameID)

	if hasMatches {
		squadStats, err = models.GetSquadStats(eventID, gameID, []int64{gameTypeID}, []int64{ageGroupID}, teamID)
	} else {
		squadStats, err = models.GetSquadStatsWithoutMatches(eventID, gameID, []int64{gameTypeID}, []int64{ageGroupID})
	}

	if err != nil {
		utils.HandleError(c, "Failed to fetch updated squad stats", err)
		return
	}

	for i := range squadStats {
		for j := range squadStats[i].Players {
			squadStats[i].Players[j].PlayerEncID = crypto.NEncrypt(squadStats[i].Players[j].PlayerID)
		}
	}

	type EventData struct {
		EncEventId    string `json:"event_id"`
		EncGameId     string `json:"game_id"`
		EncGameTypeId string `json:"game_type_id"`
		EncAgeGroupId string `json:"age_group_id"`
	}
	event := EventData{
		EncEventId:    crypto.NEncrypt(eventID),
		EncGameId:     crypto.NEncrypt(gameID),
		EncGameTypeId: crypto.NEncrypt(gameTypeID),
		EncAgeGroupId: crypto.NEncrypt(ageGroupID),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"message":   "Player(s) updated successfully",
		"eventData": event,
		"data":      squadStats,
	})
}

func ChangeCaptain(c *gin.Context) {
	TeamId := DecryptParamId(c, "team_id", true)
	if TeamId == 0 {
		return
	}
	NewId := DecryptParamId(c, "new_id", true)
	if NewId == 0 {
		return
	}
	_, isTeamMeber, err := models.GetTeamPlayer(TeamId, NewId)
	if err != nil {
		utils.HandleError(c, "Error fetching new captain", err)
		return
	}
	if !isTeamMeber {
		utils.HandleError(c, "New captain must be part of the team")
		return
	}

	_, err = database.DB.Exec(`
            UPDATE event_has_teams
            SET team_captain = $1
            WHERE id = $2`, NewId, TeamId)
	if err != nil {
		utils.HandleError(c, "failed to transfer captain role", err)
		return
	}
	utils.HandleSuccess(c, "captain replaced sucessfully")
}
