package controllers

import (
	"net/http"
	"sports-events-api/database"

	"github.com/gin-gonic/gin"
)

func GetTotalEventUsers(c *gin.Context) {
	var count int
	// Update the SQL query to count distinct user_ids
	err := database.DB.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM event_has_users`).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to fetch total event users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"total_users": count,
	})
}

func GetTotalUsers(c *gin.Context) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE status ='Active'`).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to fetch total users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"total_users": count,
	})
}
