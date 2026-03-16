package controllers

import (
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// create level of competition
func CreateLevelOfCompetition(c *gin.Context) {
	var input struct {
		Title string `json:"title" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	level := models.LevelOfCompetition{
		Title:  input.Title,
		Status: "Active",
	}

	savedLevel, err := models.InsertLevelOfCompetition(&level)
	if err != nil {
		utils.HandleError(c, "Failed to create level of competition", err)
		return
	}

	encryptedID := crypto.NEncrypt(int64(savedLevel.ID))
	utils.HandleSuccess(c, "Level of Competition created successfully", encryptedID)

}

// View level of competition
func GetAllLevelsOfCompetition(c *gin.Context) {
	// Extract query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")
	dir := c.DefaultQuery("dir", "DESC")
	status := c.Query("status")
	offset := (page - 1) * limit

	// Fetch from DB
	totalRecords, levels, err := models.GetLevelsOfCompetition(search, sort, dir, status, int64(limit), int64(offset))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"data":    "",
			"message": "Failed to fetch levels of competition.",
		})
		return
	}

	// Prepare encrypted data
	encryptedLevels := make([]map[string]interface{}, 0)
	for _, level := range levels {
		encryptedId := crypto.NEncrypt(int64(level.ID))

		encryptedLevels = append(encryptedLevels, map[string]interface{}{
			"id":         encryptedId,
			"title":      level.Title,
			"status":     level.Status,
			"created_at": level.CreatedAt,
			"updated_at": level.UpdatedAt,
		})
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"totalRecords":        totalRecords,
			"levelsOfCompetition": encryptedLevels,
		},
		"message": "Fetched all levels of competition successfully.",
	})
}

// view level of competition by particular id
func GetLevelofCompetitionById(c *gin.Context) {
	// Decrypt the encrypted ID
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	LevelOfCompetition, err := models.GetLevelofCompetitionById(decryptedId)
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
		"message": "Level of competition retrieved successfully",
		"data": gin.H{
			"id":   crypto.NEncrypt(decryptedId),
			"name": LevelOfCompetition.Title,
			// "slug":       LevelOfCompetition.Slug,
			"status":     LevelOfCompetition.Status,
			"created_at": LevelOfCompetition.CreatedAt,
			"updated_at": LevelOfCompetition.UpdatedAt,
		},
	})
}

// Delete level of competition
func Deletelevelofcompetition(c *gin.Context) {
	// Decrypt the encrypted ID
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	_, err := models.DeleteLevelofcompetitionByID(decryptedId)
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
		"message": "Level of Competition deleted successfully",
		"status":  "success",
	})
}

// update level of competition
func UpdateLevelofCompetitionById(c *gin.Context) {
	// Decrypt the encrypted ID
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	// Bind the request body to the LevelOfCompetition struct
	var data models.LevelOfCompetition
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

	// Convert int64 to int for assignment
	data.ID = int(decryptedId)

	// Call the model function to update the level of competition
	updatedLevelofCompetition, err := models.UpdateLevelOfCompetition(&data)
	if err != nil {
		utils.HandleError(c, "level of Competition Already Exist.", err)
		return
	}

	// Convert int to int64 for encryption
	eid := crypto.NEncrypt(int64(updatedLevelofCompetition.ID))

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Level of Competition updated successfully.",
		"data": gin.H{
			"id":         eid,
			"name":       updatedLevelofCompetition.Title,
			"status":     updatedLevelofCompetition.Status,
			"created_at": updatedLevelofCompetition.CreatedAt,
			"updated_at": updatedLevelofCompetition.UpdatedAt,
		},
	})
}

// update status level of competition
func UpdateLevelofCompetitionStatus(c *gin.Context) {
	// Extract and decrypt the game type ID from the URL parameter
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	// Fetch the current status from the database
	currentStatus, err := models.GetLevelofCompetitionStatusByID(decryptedId)
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
	err = models.UpdateLevelofCompetitionStatusByID(decryptedId, newStatus)
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
		"message": "Level of competition status updated successfully.",
	})
}

// level of competition config
func GetAllLevelConfigs(c *gin.Context) {
	levels, err := models.GetAllLevelOfCompetition()
	if err != nil {
		utils.HandleError(c, "Error fetching level of competition data", err)
		return
	}

	var response []models.LevelConfigRequest
	for _, level := range levels {
		encID := crypto.NEncrypt(int64(level.ID))

		response = append(response, models.LevelConfigRequest{
			ID:        int64(level.ID),
			EncID:     encID,
			Title:     level.Title,
			Status:    level.Status,
			CreatedAt: level.CreatedAt,
			UpdatedAt: level.UpdatedAt,
		})
	}

	utils.HandleSuccess(c, "Levels fetched successfully", map[string]any{"levels": response})
}
