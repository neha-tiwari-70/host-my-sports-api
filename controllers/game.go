package controllers

import (
	"fmt"
	"log"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// create games
func CreateGame(c *gin.Context) {
	var data struct {
		GameName   string   `json:"game_name" binding:"required"`
		GameTypeID []string `json:"game_type_id" binding:"required"`
		AgeGroupID []string `json:"age_group_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		utils.HandleError(c, "Invalid Input", err)
		return
	}

	slug := utils.GenerateSlug(data.GameName)

	newGame := models.Game{
		Name: data.GameName,
		Slug: slug,
	}

	createdGame, err := models.InsertGame(&newGame)
	if err != nil {
		utils.HandleError(c, "Failed To Create Game", err)
		return
	}

	for _, encryptedID := range data.GameTypeID {
		decryptedID, err := crypto.NDecrypt(encryptedID)
		if err != nil {
			utils.HandleError(c, "Failed To Decrypt Game Type ID", err)
			return
		}

		gameHasType := models.GameHasType{
			GameID:     createdGame.ID,
			GameTypeID: decryptedID,
		}

		if _, err = models.InsertGameHasType(&gameHasType); err != nil {
			utils.HandleError(c, "Failed To Link Game with Game Type", err)
			return
		}
	}

	for _, encryptedAgeGroupId := range data.AgeGroupID {
		decryptedID, err := crypto.NDecrypt(encryptedAgeGroupId)
		if err != nil {
			utils.HandleError(c, "Failed To Decrypt Age Group ID", err)
			return
		}

		gameHasAgeGroup := models.GameHasAgeGroup{
			GameId:     createdGame.ID,
			AgeGroupId: decryptedID,
		}

		if _, err = models.InsertGameHasAgeGroup(&gameHasAgeGroup); err != nil {
			utils.HandleError(c, "Failed To Link Game with Age group", err)
			return
		}
	}

	encryptedID := crypto.NEncrypt(createdGame.ID)
	utils.HandleSuccess(c, "Game created successfully", encryptedID)
}

// Listing all games
func GetAllGames(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")
	dir := c.DefaultQuery("dir", "DESC")
	offset := (page - 1) * limit

	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	totalRecords, games, err := models.GetGames(search, sort, dir, int64(limit), int64(offset))
	if err != nil {
		utils.HandleError(c, "Failed to fetch games.", err)
		return
	}

	encryptedGames := make([]map[string]interface{}, 0)
	for _, game := range games {
		encryptedID := crypto.NEncrypt(game.ID)

		encryptedGames = append(encryptedGames, map[string]interface{}{
			"id":         encryptedID,
			"game_name":  game.Name,
			"status":     game.Status,
			"created_at": game.CreatedAt,
			"updated_at": game.UpdatedAt,
		})
	}

	utils.HandleSuccess(c, "Fetched all games successfully.", gin.H{"games": encryptedGames, "totalRecords": totalRecords})
}

// View game
func GetGamesById(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	game, err := models.GetGamesById(decryptedId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch game with the provided ID.", err)
		return
	}

	if game.ID == 0 {
		utils.HandleError(c, "No game found with the provided ID.")
		return
	}

	// Encrypt the game ID and game type IDs
	encryptedGameID := crypto.NEncrypt(game.ID)
	var encryptedGameTypeID []string
	for _, id := range game.GameTypeID {
		encryptedGameTypeID = append(encryptedGameTypeID, crypto.NEncrypt(id))
	}
	// fmt.Println("IDS", game.AgeGroupID)
	var encryptedAgeGroupIds []string
	for _, id := range game.AgeGroupID {
		encryptedAgeGroupIds = append(encryptedAgeGroupIds, crypto.NEncrypt(id))
	}

	// Send the game data in the response
	utils.HandleSuccess(c, "Game retrieved successfully.", gin.H{
		"id":           encryptedGameID,
		"game_name":    game.Name,
		"slug":         game.Slug,
		"game_type_id": encryptedGameTypeID,
		"age_group_id": encryptedAgeGroupIds,
		"created_at":   game.CreatedAt,
		"updated_at":   game.UpdatedAt,
	})
}

// Delete games
func DeleteGame(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	err := models.DeleteGameById(decryptedId)
	if err != nil {
		if err.Error() == fmt.Sprintf("game with ID %d is not found", decryptedId) {
			utils.HandleError(c, "No game found with the provided ID.")
			return
		}
		utils.HandleError(c, "Failed to delete the game.", err)
		return
	}

	utils.HandleSuccess(c, "Game deleted successfully.")
}

// Update game
func UpdateGame(c *gin.Context) {

	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	var gameUpdate models.GameUpdate
	if err := c.ShouldBindJSON(&gameUpdate); err != nil {
		utils.HandleError(c, "Invalid input data.", err)
		return
	}

	err := models.UpdateGameById(decryptedId, gameUpdate)
	if err != nil {
		if err.Error() == fmt.Sprintf("game with ID %d is not found", decryptedId) {
			utils.HandleError(c, "No game found with the provided ID.")
			return
		}
		utils.HandleError(c, "Failed to update the game.", err)
		return
	}

	var gameTypeIDs []int64
	var gameTypeDetails []gin.H
	for _, encryptedID := range gameUpdate.GameTypeIDs {
		decryptedTypeID, err := crypto.NDecrypt(encryptedID)
		if err != nil {
			utils.HandleError(c, "Failed to decrypt game type ID.", err)
			return
		}

		gameTypeID := int64(decryptedTypeID)
		gameTypeIDs = append(gameTypeIDs, gameTypeID)

		gameType, err := models.GetGameTypeById(decryptedTypeID)
		if err != nil {
			utils.HandleError(c, "Failed to retrieve game type name.", err)
			return
		}
		encryptedTypeID := crypto.NEncrypt(decryptedTypeID)
		gameTypeDetails = append(gameTypeDetails, gin.H{"name": gameType.Name, "id": encryptedTypeID})
	}

	if len(gameTypeIDs) > 0 {
		err = models.UpdateGameTypeAssociations(decryptedId, gameTypeIDs)
		if err != nil {
			utils.HandleError(c, "Failed to update game type associations.", err)
			return
		}
	}

	updatedGame, err := models.GetGamesById(decryptedId)
	if err != nil {
		utils.HandleError(c, "Failed to retrieve updated game details.", err)
		return
	}

	encryptedGameID := crypto.NEncrypt(updatedGame.ID)

	responseData := gin.H{"id": encryptedGameID, "game_name": updatedGame.Name, "game_types": gameTypeDetails, "created_at": updatedGame.CreatedAt, "updated_at": updatedGame.UpdatedAt}

	utils.HandleSuccess(c, "Game updated successfully.", responseData)
}

// Update Game Status
func UpdateGameStatus(c *gin.Context) {

	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	currentStatus, err := models.GetGameStatusByID(decryptedId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to fetch current status.",
		})
		return
	}

	newStatus := "Inactive"
	if currentStatus == "Inactive" {
		newStatus = "Active"
	}

	err = models.UpdateGameStatus(decryptedId, newStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update status.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Game status updated successfully.",
	})
}

// Get all game Types
func GetAllTypes(c *gin.Context) {
	gamesTypes, err := models.GetAllGameTypes()
	if err != nil {
		log.Printf("Error fetching game types: %v", err)

		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"data":    "",
			"message": "Failed to fetch game types.",
		})
		return
	}

	encryptedGamesTypes := make([]map[string]interface{}, 0)
	for _, gameType := range gamesTypes {
		encryptedId := crypto.NEncrypt(gameType.ID)

		encryptedGamesTypes = append(encryptedGamesTypes, map[string]interface{}{
			"id":   encryptedId,
			"name": gameType.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"gamesTypes": encryptedGamesTypes,
		},
		"message": "Fetched all game types successfully.",
	})
}

// Get all game Types
func GetAllAgeGroup(c *gin.Context) {
	ageGroups, err := models.GetAllAgeGroup()
	if err != nil {
		log.Printf("Error fetching age group: %v", err)

		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"data":    "",
			"message": "Failed to fetch age group.",
		})
		return
	}

	encryptedAgeGroups := make([]map[string]interface{}, 0)
	for _, ageGroup := range ageGroups {
		encryptedId := crypto.NEncrypt(ageGroup.ID)

		encryptedAgeGroups = append(encryptedAgeGroups, map[string]interface{}{
			"id":   encryptedId,
			"name": ageGroup.Category,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ageGroups": encryptedAgeGroups,
		},
		"message": "Fetched all age group successfully.",
	})
}

func GetGamesInfoByGameIds(c *gin.Context) {
	var games struct {
		Ids []string `json:"game_ids"`
	}
	if err := c.ShouldBindJSON(&games); err != nil {
		utils.HandleError(c, "Invalid request body", err)
		return
	}

	if len(games.Ids) == 0 {
		utils.HandleError(c, "No game Ids found")
		return
	}

	var decGameIds []int64
	for _, encId := range games.Ids {
		id, err := crypto.NDecrypt(encId)
		if err != nil {
			utils.HandleError(c, "Failed to decrypt id.", err)
			return
		}
		decGameIds = append(decGameIds, id)
	}

	gameList, err := models.GetGamesInfoByGameIds(decGameIds)
	if err != nil {
		utils.HandleError(c, "Failed to fetch games. ", err)
		return
	}

	var result []gin.H
	for _, game := range gameList {
		var gameTypes []gin.H
		for _, gt := range game.GameTypes {
			gameTypes = append(gameTypes, gin.H{
				"id":   crypto.NEncrypt(gt.ID),
				"name": gt.Name,
				"slug": gt.Slug,
			})
		}

		var ageGroups []gin.H
		for _, ag := range game.AgeGroups {
			ageGroups = append(ageGroups, gin.H{
				"id":   crypto.NEncrypt(ag.ID),
				"name": ag.Category,
				"slug": utils.GenerateSlug(ag.Category),
			})
		}

		result = append(result, gin.H{
			"id":         crypto.NEncrypt(game.ID),
			"game_name":  game.Name,
			"slug":       game.Slug,
			"status":     game.Status,
			"game_types": gameTypes,
			"age_groups": ageGroups,
			"created_at": game.CreatedAt,
			"updated_at": game.UpdatedAt,
		})
	}
	utils.HandleSuccess(c, "Games retrieved successfully", result)
}
