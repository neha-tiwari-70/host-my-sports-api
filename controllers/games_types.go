package controllers

import (
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func CreateGamesTypes(c *gin.Context) {
	var data models.AddGame
	if err := c.ShouldBindBodyWithJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid Input",
			"data":    err.Error(),
		})
		return
	}
	validationErrors := ValidateStruct(data)
	if validationErrors != "" {
		c.JSON(http.StatusOK, gin.H{
			"data":    "",
			"status":  "error",
			"message": validationErrors,
		})
		return
	}

	var input_games_types = models.Games_Types{
		Name: data.Name,
	}

	creategamesTypes, err := models.InsertGamesTypes(&input_games_types)
	if err != nil {
		utils.HandleError(c, "Game type already exist.", err)
		return
	}

	eid := crypto.NEncrypt(creategamesTypes.ID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Game type created successfully",
		"data":    eid,
	})
}

func GetGamesTypesById(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	gamesTypes, err := models.GetGamesTypesById(decryptedId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Games types retrieved successfully",
		"data": gin.H{
			"id":         crypto.NEncrypt(decryptedId),
			"name":       gamesTypes.Name,
			"slug":       gamesTypes.Slug,
			"status":     gamesTypes.Status,
			"created_at": gamesTypes.CreatedAt,
			"updated_at": gamesTypes.UpdatedAt,
		},
	})
}

func DeleteGamesTypes(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	_, err := models.DeleteGamesTypesByID(decryptedId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    nil,
		"message": "Game type deleted successfully",
		"status":  "success",
	})
}

func GetAllGamesTypes(c *gin.Context) {
	// Extract query parameters for pagination, search, sorting, and status
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")
	dir := c.DefaultQuery("dir", "DESC")
	status := c.Query("status") // Fetch status from query params
	offset := (page - 1) * limit

	// Fetch data from the model with status filtering
	totalRecords, gamesTypes, err := models.GetGamesTypes(search, sort, dir, status, int64(limit), int64(offset))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"data":    "",
			"message": "Failed to fetch games types.",
		})
		return
	}

	// Encrypt the IDs of all games_types records
	encryptedGamesTypes := make([]map[string]interface{}, 0)
	for _, gameType := range gamesTypes {
		encryptedId := crypto.NEncrypt(gameType.ID)

		// Append the encrypted game type to the list
		encryptedGamesTypes = append(encryptedGamesTypes, map[string]interface{}{
			"id":         encryptedId,
			"name":       gameType.Name,
			"slug":       gameType.Slug,
			"status":     gameType.Status,
			"created_at": gameType.CreatedAt,
			"updated_at": gameType.UpdatedAt,
		})
	}

	// Respond with paginated and filtered data
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"totalRecords": totalRecords,
			"gamesTypes":   encryptedGamesTypes,
		},
		"message": "Fetched all games types successfully.",
	})
}

func UpdateGamesTypesById(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	// Bind the request body to the Games_Types struct
	var data models.Games_Types
	if err := c.ShouldBindBodyWithJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid input.",
			"data":    err.Error(),
		})
		return
	}

	validationErrors := ValidateStruct(data)
	if validationErrors != "" {
		c.JSON(http.StatusOK, gin.H{
			"data":    "",
			"status":  "error",
			"message": validationErrors,
		})
		return
	}

	// Populate the ID in the games_types struct
	data.ID = decryptedId

	// Call the model function to update the games type
	updatedGameType, err := models.UpdateGamesTypes(&data)
	if err != nil {
		utils.HandleError(c, "Games Types Already Exist.", err)
		return
	}

	// Encrypt the updated ID for the response
	eid := crypto.NEncrypt(updatedGameType.ID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Game type updated successfully.",
		"data": gin.H{
			"id":         eid,
			"name":       updatedGameType.Name,
			"slug":       updatedGameType.Slug,
			"status":     updatedGameType.Status,
			"created_at": updatedGameType.CreatedAt,
			"updated_at": updatedGameType.UpdatedAt,
		},
	})
}

func UpdateGameTypeStatus(c *gin.Context) {
	// Extract the game type ID from the URL parameter
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	// Fetch the current status from the database
	currentStatus, err := models.GetGameTypeStatusByID(decryptedId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to fetch current status.",
		})
		return
	}

	// Determine the new status based on the current status
	newStatus := "Inactive"
	if currentStatus == "Inactive" {
		newStatus = "Active"
	}

	// Update the status in the database
	err = models.UpdateGameTypeStatusByID(decryptedId, newStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update status.",
		})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Game type status updated successfully.",
	})
}
