package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"
	"strings"
	"time"

	// "sports-events-api/crypto"
	"github.com/gin-gonic/gin"
)

const EventLogoFolderPath = "public/event"

func CreateEvent(c *gin.Context) {
	var NewEvent models.Event

	err := c.Request.ParseMultipartForm(10 << 20)
	if err != nil {
		// fmt.Println("Error parsing form data:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	// NewEvent.Name = c.PostForm("name")
	NewEvent.Name = c.PostForm("name")

	if NewEvent.Name == "" {
		utils.HandleError(c, "Name is required", nil)
		return
	}
	if len(NewEvent.Name) < 2 {
		utils.HandleError(c, "Name must be at least 2 characters long", nil)
		return
	}
	if len(NewEvent.Name) > 100 {
		utils.HandleError(c, "Name cannot be longer than 100 characters", nil)
		return
	}

	// NewEvent.FromDate = c.PostForm("from_date")
	// NewEvent.ToDate = c.PostForm("to_date")
	NewEvent.CreatedByRole = c.PostForm("created_by_role")
	NewEvent.CreatedByEncId = c.PostForm("created_by_id")
	NewEvent.Venue = c.PostForm("venue")
	if NewEvent.Venue == "" {
		utils.HandleError(c, "Venue is required", nil)
		return
	}
	if len(NewEvent.Venue) < 3 {
		utils.HandleError(c, "Venue must be at least 3 characters long", nil)
		return
	}

	NewEvent.Fees = c.PostForm("fees")
	if NewEvent.Fees == "" {
		utils.HandleError(c, "Fees must be required", nil)
	}
	NewEvent.About = c.PostForm("about")
	NewEvent.SponsorTitle = c.PostForm("sponsor_title")
	// NewEvent.FacebookLink = c.PostForm("facebook_link")
	// NewEvent.InstagramLink = c.PostForm("instagram_link")
	// NewEvent.YoutubeLink = c.PostForm("linkedin_link")

	facebookLink := c.PostForm("facebook_link")
	if facebookLink != "" {
		if !strings.Contains(facebookLink, "facebook.com") {
			utils.HandleError(c, "Invalid facebook link. Please provide a valid facebook url", nil)
			return
		}
		NewEvent.FacebookLink = facebookLink
	}

	instagramLink := c.PostForm("instagram_link")
	if instagramLink != "" {
		if !strings.Contains(instagramLink, "instagram.com") {
			utils.HandleError(c, "Invalid instagram link. Please provide a valid instagram url", nil)
			return
		}
		NewEvent.InstagramLink = instagramLink
	}

	var linkedinLink string
	linkedinLink = c.PostForm("linkedin_link")
	if linkedinLink != "" {
		if !strings.Contains(linkedinLink, "linkedin.com") {
			utils.HandleError(c, "Invalid linkedin link. Please provide a valid linkedin url", nil)
			return
		}
		NewEvent.LinkedinLink = &linkedinLink
	} else {
		NewEvent.LinkedinLink = nil
	}

	levelOfCompetitionEnc := c.PostForm("level_of_competition")
	NewEvent.LevelOfCompetition, err = crypto.NDecrypt(levelOfCompetitionEnc)
	if err != nil {
		utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
		return
	}

	googleMapLink := c.PostForm("google_map_link")
	if googleMapLink != "" && !utils.IsValidGoogleMapsURL(googleMapLink) {
		utils.HandleError(c, "Invalid google map link.", err)
		return
	}
	NewEvent.GoogleMapLink = googleMapLink

	fromDateStr := c.PostForm("from_date")
	toDateStr := c.PostForm("to_date")
	startRegStr := c.PostForm("start_registration_date")
	lastRegStr := c.PostForm("last_registration_date")

	layout := "2006-01-02"
	today := time.Now().Truncate(24 * time.Hour)

	fromDate, err := time.Parse(layout, fromDateStr)
	if err != nil {
		utils.HandleError(c, "Invalid from_date format. Use YYYY-MM-DD", err)
		return
	}
	toDate, err := time.Parse(layout, toDateStr)
	if err != nil {
		utils.HandleError(c, "Invalid to_date format. Use YYYY-MM-DD", err)
		return
	}

	// from_date must be today or later
	if fromDate.Before(today) {
		utils.HandleError(c, "From date must be today or a future date", nil)
		return
	}

	// to_date must be after or wqual from_date
	if toDate.Before(fromDate) {
		utils.HandleError(c, "To date must be after or equal to from date", nil)
		return
	}

	// start_reg_date between today and from_date
	if startRegStr != "" {
		startReg, err := time.Parse(layout, startRegStr)
		if err != nil {
			utils.HandleError(c, "Invalid start_registration_date format. Use YYYY-MM-DD", err)
			return
		}
		if startReg.Before(today) || startReg.After(fromDate) {
			utils.HandleError(c, "Start registration date must be between today and from_date", nil)
			return
		}
		NewEvent.StartRegistrationDate = startRegStr
	}

	// last_reg_date between start_reg_date and from_date
	if lastRegStr != "" {
		lastReg, err := time.Parse(layout, lastRegStr)
		if err != nil {
			utils.HandleError(c, "Invalid last_registration_date format. Use YYYY-MM-DD", err)
			return
		}

		// need startReg to validate
		startReg, _ := time.Parse(layout, startRegStr)
		if lastReg.Before(startReg) || lastReg.After(fromDate) {
			utils.HandleError(c, "Last registration date must be between start_registration_date and from_date", nil)
			return
		}
		NewEvent.LastRegistrationDate = lastRegStr
	}

	// Assign final values
	NewEvent.FromDate = fromDateStr
	NewEvent.ToDate = toDateStr

	file, err := c.FormFile("logo")
	if err == nil {
		filename := fmt.Sprintf("event_%d_%s", time.Now().Unix(), file.Filename)
		filePath := filepath.Join(EventLogoFolderPath, filename)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			utils.HandleError(c, "Could not save file", err)
			return
		}

		NewEvent.Logo = filepath.ToSlash(filepath.Join("event", filename))
	}

	gameJson := c.PostForm("games")
	err = json.Unmarshal([]byte(gameJson), &NewEvent.Games)
	if err != nil {
		utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
		return
	}

	NewEvent.StateEncId = c.PostForm("state_id")
	NewEvent.StateId, err = crypto.NDecrypt(NewEvent.StateEncId)
	if err != nil {
		utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
		return
	}

	NewEvent.CreatedById, err = crypto.NDecrypt(NewEvent.CreatedByEncId)
	if err != nil {
		utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
		return
	}

	NewEvent.CityEncId = c.PostForm("city_id")
	NewEvent.CityId, err = crypto.NDecrypt(NewEvent.CityEncId)
	if err != nil {
		utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
		return
	}

	feeType := c.PostForm("fee_type")
	if feeType == "event_fee" {
		NewEvent.Fees = c.PostForm("fees")
	} else if feeType == "game_fee" {
		NewEvent.Fees = "0"

		gameFeesJson := c.PostForm("game_fees")
		var gameFeesMap map[string]float64
		if err := json.Unmarshal([]byte(gameFeesJson), &gameFeesMap); err == nil {
			for i := range NewEvent.Games {
				if fee, ok := gameFeesMap[NewEvent.Games[i].EncID]; ok {
					NewEvent.Games[i].Fees = fee
				}
			}
		}
	}
	// fmt.Println("Neww Event : ", NewEvent)
	for i, game := range NewEvent.Games {
		// fmt.Println("Game:", &NewEvent.Games)
		// fmt.Println("Game : dcsd", game.EncID)
		game.GameID, err = crypto.NDecrypt(game.EncID)
		// fmt.Println("Game ID", game.GameID)
		if err != nil {
			utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
			return
		}

		for j := range game.Type {
			game.Type[j].TypeID, err = crypto.NDecrypt(game.Type[j].EncId)
			// fmt.Println("gametypeid ", game.Type[j].TypeID)
			if err != nil {
				utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
				return
			}

			for k := range game.Type[j].AgeGroups {
				game.Type[j].AgeGroups[k].Id, err = crypto.NDecrypt(game.Type[j].AgeGroups[k].EncId)
				if err != nil {
					utils.HandleError(c, "Oops! something went wrong. Please try again later.", err)
					return
				}
			}
			// fmt.Println("Decrypted Type ID:", game.Type[j].TypeID)
		}
		NewEvent.Games[i] = game
	}

	tx, err := database.DB.Begin()
	if err != nil {
		utils.HandleError(c, "error creating a transaction", err)
		return
	}

	createdEvent, err := models.CreateEvent(&NewEvent, tx)
	if err != nil {
		utils.HandleError(c, "Unable to create event", err)
		tx.Rollback()
		return
	}
	tx.Commit()
	encID := crypto.NEncrypt(createdEvent.ID)
	createdEvent.EncID = encID

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event created successfully",
		"data": gin.H{
			"id":    createdEvent.EncID,
			"event": createdEvent,
			"logo":  fmt.Sprintf("/%s", createdEvent.Logo),
		},
	})
}

func UploadEventLogo(c *gin.Context) {
	eventID := c.Param("id")
	if eventID == "" {
		utils.HandleError(c, "Oops! something went wrong. Please try again later.")
		return
	}

	eventLogo, err := c.FormFile("event_logo")
	if err != nil {
		utils.HandleError(c, "No event logo provided", err)
		return
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("event_%s_%s.png", eventID, timestamp)

	filePath := filepath.Join(EventLogoFolderPath, filename)

	if err := c.SaveUploadedFile(eventLogo, filePath); err != nil {
		utils.HandleError(c, "Failed to save event logo", err)
		return
	}

	filePathDB := filepath.ToSlash(filepath.Join("public/event", filename))

	query := `UPDATE events SET logo = $1 WHERE id = $2`
	_, err = database.DB.Exec(query, filePathDB, eventID)
	if err != nil {
		utils.HandleError(c, "Failed to update event logo", err)
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "Event logo uploaded successfully",
		"data": gin.H{
			"event_id": eventID,
			"logo":     filePathDB,
		},
	})
}

// UpdateEvent handles updating an existing event
func UpdateEvent(c *gin.Context) {
	var updatedEvent models.Event

	err := c.Request.ParseMultipartForm(10 << 20)
	if err != nil {
		utils.HandleError(c, "Invalid form data", err)
		return
	}

	eventID := DecryptParamId(c, "id", true)
	if eventID == 0 {
		return
	}

	existingEvent, err := models.GetEventByID(int(eventID))
	if err != nil {
		utils.HandleError(c, "Event not found", err)
		return
	}

	if existingEvent.StartRegistrationDate != "" {
		startDate, err := time.Parse("2006-01-02", existingEvent.StartRegistrationDate)
		if err == nil && time.Now().After(startDate) {
			utils.HandleError(c, "Cannot update event after registration has started", err)
			return
		}
	}

	updatedEvent.ID = eventID
	updatedEvent.Name = c.PostForm("name")
	if updatedEvent.Name == "" {
		utils.HandleError(c, "Name is required", nil)
		return
	}
	if len(updatedEvent.Name) < 2 {
		utils.HandleError(c, "Name must be at least 2 characters long", nil)
		return
	}
	if len(updatedEvent.Name) > 100 {
		utils.HandleError(c, "Name cannot be longer than 100 characters", nil)
		return
	}

	updatedEvent.Slug = utils.GenerateSlug(updatedEvent.Name)
	fromDateStr := c.PostForm("from_date")
	toDateStr := c.PostForm("to_date")
	layout := "2006-01-02"
	today := time.Now().Truncate(24 * time.Hour)

	fromDate, err := time.Parse(layout, fromDateStr)
	if err != nil {
		utils.HandleError(c, "Invalid from_date format. Use YYYY-MM-DD", err)
		return
	}
	toDate, err := time.Parse(layout, toDateStr)
	if err != nil {
		utils.HandleError(c, "Invalid to_date format. Use YYYY-MM-DD", err)
		return
	}

	if fromDate.Before(today) {
		utils.HandleError(c, "From date must be today or a future date", nil)
		return
	}

	if toDate.Before(fromDate) {
		utils.HandleError(c, "To date must be after or equal to from date", nil)
		return
	}

	updatedEvent.FromDate = fromDateStr
	updatedEvent.ToDate = toDateStr
	updatedEvent.CreatedById = existingEvent.CreatedById
	updatedEvent.CreatedByRole = c.PostForm("created_by_role")
	updatedEvent.Venue = c.PostForm("venue")
	if updatedEvent.Venue == "" {
		utils.HandleError(c, "Venue is required", nil)
		return
	}
	if len(updatedEvent.Venue) < 3 {
		utils.HandleError(c, "Venue must be at least 3 characters long", nil)
		return
	}

	updatedEvent.Fees = c.PostForm("fees")
	if updatedEvent.Fees == "" {
		utils.HandleError(c, "Fees must be required", nil)
		return
	}

	updatedEvent.About = c.PostForm("about")
	updatedEvent.SponsorTitle = c.PostForm("sponsor_title")
	facebookLink := c.PostForm("facebook_link")
	if facebookLink != "" {
		if !strings.Contains(facebookLink, "facebook.com") {
			utils.HandleError(c, "Invalid Facebook link. Please provide a valid Facebook URL", nil)
			return
		}
		updatedEvent.FacebookLink = facebookLink
	} else if c.PostForm("facebook_link") == "" {
		updatedEvent.FacebookLink = ""
	} else {
		updatedEvent.FacebookLink = existingEvent.FacebookLink
	}

	instagramLink := c.PostForm("instagram_link")
	if instagramLink != "" {
		if !strings.Contains(instagramLink, "instagram.com") {
			utils.HandleError(c, "Invalid Instagram link. Please provide a valid Instagram URL", nil)
			return
		}
		updatedEvent.InstagramLink = instagramLink
	} else if c.PostForm("instagram_link") == "" {
		updatedEvent.InstagramLink = ""
	} else {
		updatedEvent.InstagramLink = existingEvent.InstagramLink
	}

	linkedinLink := c.PostForm("linkedin_link")
	if linkedinLink != "" {
		if !strings.Contains(linkedinLink, "linkedin.com") {
			utils.HandleError(c, "Invalid LinkedIn link. Please provide a valid LinkedIn URL", nil)
			return
		}
		updatedEvent.LinkedinLink = &linkedinLink
	} else if c.PostForm("linkedin_link") == "" {
		updatedEvent.LinkedinLink = nil
	} else {
		updatedEvent.LinkedinLink = existingEvent.LinkedinLink
	}

	levelOfCompetitionEnc := c.PostForm("level_of_competition")
	if levelOfCompetitionEnc != "" {
		levelID, err := crypto.NDecrypt(levelOfCompetitionEnc)
		if err != nil {
			utils.HandleError(c, "Invalid level of competition", err)
			return
		}
		updatedEvent.LevelOfCompetition = levelID
	} else {
		updatedEvent.LevelOfCompetition = existingEvent.LevelOfCompetition
	}

	googleMapLink := c.PostForm("google_map_link")
	if googleMapLink != "" && !utils.IsValidGoogleMapsURL(googleMapLink) {
		utils.HandleError(c, "Invalid google map link. Please provide a valid google map url", err)
		return
	}
	updatedEvent.GoogleMapLink = googleMapLink

	updatedEvent.LastRegistrationDate = c.PostForm("last_registration_date")
	if updatedEvent.LastRegistrationDate != "" {
		if _, err := time.Parse("2006-01-02", updatedEvent.LastRegistrationDate); err != nil {
			utils.HandleError(c, "Invalid last registration date format. Use YYYY-MM-DD", err)
			return
		}
	}

	updatedEvent.StartRegistrationDate = c.PostForm("start_registration_date")
	if updatedEvent.StartRegistrationDate != "" {
		if _, err := time.Parse("2006-01-02", updatedEvent.StartRegistrationDate); err != nil {
			utils.HandleError(c, "Invalid start registration date format. Use YYYY-MM-DD", err)
			return
		}
	}

	// Handle logo upload or deletion
	deleteLogo := c.PostForm("delete_logo")
	file, err := c.FormFile("logo")
	if deleteLogo == "true" {
		// Set logo to empty string to remove it
		updatedEvent.Logo = ""
	} else if err == nil && file.Size > 0 {
		filename := fmt.Sprintf("event_%d_%s", time.Now().Unix(), file.Filename)
		filePath := filepath.Join(EventLogoFolderPath, filename)
		if err := c.SaveUploadedFile(file, filePath); err == nil {
			updatedEvent.Logo = filepath.ToSlash(filepath.Join("event", filename))
		}
	} else {
		updatedEvent.Logo = existingEvent.Logo
	}

	updatedEvent.StateEncId = c.PostForm("state_id")
	updatedEvent.StateId, err = crypto.NDecrypt(updatedEvent.StateEncId)
	if err != nil {
		fmt.Println("Failed to decrypt state id:", err)
	}

	updatedEvent.CityEncId = c.PostForm("city_id")
	updatedEvent.CityId, err = crypto.NDecrypt(updatedEvent.CityEncId)
	if err != nil {
		fmt.Println("Failed to decrypt city id:", err)
	}

	feeType := c.PostForm("fee_type")
	if feeType == "event_fee" {
		updatedEvent.Fees = c.PostForm("fees")
	} else if feeType == "game_fee" {
		updatedEvent.Fees = "0"
		gameFeesJson := c.PostForm("game_fees")
		var gameFeesMap map[string]float64
		if err := json.Unmarshal([]byte(gameFeesJson), &gameFeesMap); err == nil {
			for i := range updatedEvent.Games {
				if fee, ok := gameFeesMap[updatedEvent.Games[i].EncID]; ok {
					updatedEvent.Games[i].Fees = fee
				}
			}
		}
	}

	err = models.UpdateEvent(&updatedEvent)
	if err != nil {
		utils.HandleError(c, "Failed to update event", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event updated successfully",
		"data": gin.H{
			"eventId": eventID,
			"event":   updatedEvent,
		},
	})
}

// UpdateEventLogo handles updating the event logo
func UpdateEventLogo(c *gin.Context) {
	eventID := c.Param("id")
	if eventID == "" {
		utils.HandleError(c, "event_id is required")
		return
	}

	eventLogo, err := c.FormFile("event_logo")
	if err != nil {
		utils.HandleError(c, "No event logo provided", err)
		return
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("event_%s_%s.png", eventID, timestamp)
	filePath := filepath.Join(EventLogoFolderPath, filename)

	if err := c.SaveUploadedFile(eventLogo, filePath); err != nil {
		utils.HandleError(c, "Failed to save event logo", err)
		return
	}

	filePathDB := filepath.ToSlash(filepath.Join("public/event", filename))
	query := `UPDATE events SET logo = $1 WHERE id = $2`
	_, err = database.DB.Exec(query, filePathDB, eventID)
	if err != nil {
		utils.HandleError(c, "Failed to update event logo in DB", err)
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "Event logo updated successfully",
		"data": gin.H{
			"event_id": eventID,
			"logo":     filePathDB,
		},
	})
}

// DeleteEventLogo handles deleting the event logo
func DeleteEventLogo(c *gin.Context) {
	eventID := c.Param("id")
	if eventID == "" {
		utils.HandleError(c, "event_id is required")
		return
	}

	// Decrypt event ID
	decryptedEventID, err := crypto.NDecrypt(eventID)
	if err != nil {
		fmt.Println("Failed to decrypt event ID:", eventID, "Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid event ID",
			"data":    fmt.Sprintf("Invalid event ID: %s", eventID),
		})
		return
	}

	// Verify event exists
	var existingLogo string
	err = database.DB.QueryRow(`SELECT logo FROM events WHERE id = $1`, decryptedEventID).Scan(&existingLogo)
	if err == sql.ErrNoRows {
		utils.HandleError(c, "Event not found", err)
		return
	}
	if err != nil {
		utils.HandleError(c, "Failed to fetch event", err)
		return
	}

	// Update the event record to remove the logo
	query := `UPDATE events SET logo = '' WHERE id = $1`
	_, err = database.DB.Exec(query, decryptedEventID)
	if err != nil {
		utils.HandleError(c, "Failed to delete event logo in DB", err)
		return
	}

	// Optionally, delete the logo file from the filesystem
	if existingLogo != "" {
		filePath := filepath.Join(EventLogoFolderPath, filepath.Base(existingLogo))
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			fmt.Println("Failed to delete logo file:", err)
			// Don't return error to client, as DB update was successful
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event logo deleted successfully",
		"data": gin.H{
			"event_id": decryptedEventID,
		},
	})
}

func UpdateEventStatus(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	currentStatus, err := models.GetEventStatusByID(decryptedId)
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

	err = models.UpdateEventStatusByID(decryptedId, newStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update status.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Events status updated successfully",
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func GetAllEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")
	dir := c.DefaultQuery("dir", "DESC")
	status := c.Query("status")
	offset := (page - 1) * limit
	from_date := c.Query("from_date")
	to_date := c.Query("to_date")
	type_of_event := c.Query("type_of_event")
	userEncId := c.Query("user_id")
	user_id, _ := crypto.NDecrypt(userEncId)

	isOrganized := false
	isParticipated := false
	if type_of_event == "organized" {
		isOrganized = true
	}
	if type_of_event == "participated" {
		isParticipated = true
	}

	// Fetch data from the model with status filtering
	totalRecords, events, err := models.GetEvents(search, sort, dir, status, from_date, to_date, int64(limit), int64(offset), &user_id, isOrganized, isParticipated)
	if err != nil {
		utils.HandleError(c, "Failed to fetch events.", err)
		return
	}

	// Encrypt the IDs of all events records
	encryptedEvents := make([]map[string]interface{}, 0)
	for _, event := range events {
		total_registered_fees, err := models.FetchFeesCountForOrg(event.CreatedById)
		if err != nil {
			utils.HandleError(c, "Error fetching total fees registered for the event", err)
			return
		}

		encryptedEventID := crypto.NEncrypt(event.ID)
		event.CreatedByEncId = crypto.NEncrypt(event.CreatedById)
		encryptedGames := GetEncryptedGames(&event)
		city, state, err := GetCityAndState(event.StateId, event.CityId)
		if err != nil {
			utils.HandleError(c, "Unable to fetch city and state", err)
			return
		}

		// Check if event logo is valid
		eventLogoPath := "public/" + event.LogoPath
		defaultLogoPath := "public/static/staticLogo.jpg"

		if event.LogoPath == "" || !fileExists(eventLogoPath) {
			eventLogoPath = defaultLogoPath
		}

		encryptedEvents = append(encryptedEvents, map[string]interface{}{
			"id":                      encryptedEventID,
			"name":                    event.Name,
			"created_by_id":           event.CreatedByEncId,
			"created_by_role":         event.CreatedByRole,
			"created_by_name":         event.CreatedByName,
			"from_date":               event.FromDate,
			"to_date":                 event.ToDate,
			"start_registration_date": event.StartRegistrationDate,
			"last_registration_date":  event.LastRegistrationDate,
			"state":                   state.Name,
			"city":                    city.Name,
			"venue":                   event.Venue,
			"fees":                    event.Fees,
			"about":                   event.About,
			"games":                   encryptedGames,
			"facebook_link":           event.FacebookLink,
			"instagram_link":          event.InstagramLink,
			"linkedin_link":           event.LinkedinLink,
			"logo":                    eventLogoPath,
			"team_count":              event.TeamCount,
			"status":                  event.Status,
			"created_at":              event.CreatedAt,
			"updated_at":              event.UpdatedAt,
			"total_registered_fees":   total_registered_fees,
		})
	}

	// Respond with paginated and filtered data
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"totalRecords": totalRecords,
			"events":       encryptedEvents,
		},
		"message": "Fetched all events successfully.",
	})
}

func GetCityAndState(stateId, cityId int64) (*models.City, *models.State, error) {
	city, err := models.GetCityById(cityId)
	if err != nil {
		fmt.Printf("unable to fetch city %+v", err)
	}

	state, err := models.GetStateById(stateId)
	if err != nil {
		fmt.Printf("unable to fetch state %+v", err)
	}
	return city, state, err
}

func GetEncryptedGames(event *models.Event) []gin.H {
	var encryptedGames []gin.H
	for _, game := range event.Games {
		event.TeamCount += game.TeamCount
		gameName, err := models.GetGamesById(game.GameID)
		if err != nil {
			fmt.Printf("Unable to fetch game name : %+v", err)
		}
		encryptedEventGameID := crypto.NEncrypt(game.ID)
		encryptedGameID := crypto.NEncrypt(gameName.ID)

		var encryptedGameTypes []gin.H
		for _, gameType := range game.Type {
			encryptedGameTypeID := crypto.NEncrypt(gameType.ID)
			EncTypeID := crypto.NEncrypt(gameType.TypeID)

			var encryptedAgeGroup []gin.H
			for _, ageGroup := range gameType.AgeGroups {
				encryptedAgeGroup = append(encryptedAgeGroup, gin.H{
					"id":                        crypto.NEncrypt(ageGroup.Id),
					"event_has_game_type_id":    crypto.NEncrypt(ageGroup.EventHasGameTypeId),
					"category":                  ageGroup.Category,
					"min_age":                   ageGroup.MinAge,
					"max_age":                   ageGroup.MaxAge,
					"slug":                      ageGroup.Slug,
					"max_registrations_reached": ageGroup.MaxRegistrationsReached,
					"active_team_count":         ageGroup.ActiveTeamCount,
					"min_player":                ageGroup.MinPlayer,
					"max_player":                ageGroup.MaxPlayer,
					"participated":              ageGroup.Participated,
				})
			}

			encryptedGameTypes = append(encryptedGameTypes, gin.H{
				"id":                        encryptedGameTypeID,
				"type_id":                   EncTypeID,
				"name":                      gameType.Name,
				"slug":                      gameType.Slug,
				"max_registrations_reached": gameType.MaxRegistrationsReached,
				"active_team_count":         gameType.ActiveTeamCount,
				"age_groups":                encryptedAgeGroup,
				// "min_player":                gameType.MinPlayer,
				// "maxPlayer":                 gameType.MaxPlayer,
				"participated": gameType.Participated,
			})
		}

		encryptedGames = append(encryptedGames, gin.H{
			"id":                        encryptedEventGameID,
			"game_id":                   encryptedGameID,
			"game":                      gameName.Name,
			"type_of_tournament":        game.TypeOfTournament,
			"max_registration":          game.MaxRegistration,
			"max_registrations_reached": game.MaxRegistrationsReached,
			"maximum_set_points":        game.MaxSetPoint,
			"number_of_players":         game.NumberOfPlayers,
			"team_size":                 game.TeamSize,
			"duration":                  game.Duration,
			"type":                      encryptedGameTypes,
			"sets":                      game.Sets,
			"fees":                      game.Fees,
			"is_tshirt_size_required":   game.IsTshirtSizeRequired,
			"number_of_overs":           game.NumberOfOvers,
			"ball_type":                 game.BallType,
			"team_count":                game.TeamCount,
			"created_at":                game.CreatedAt,
			"updated_at":                game.UpdatedAt,
			"option_disabled":           game.MaxRegistrationsReached,
			"participated":              game.Participated,
			"distance_category":         game.DistanceCategory,
			"cycle_type":                game.CycleType,
		})
	}
	return encryptedGames
}

func GetEventsById(c *gin.Context) {
	// Decrypt the encrypted ID
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}
	// Decrypt the encrypted ID
	decryptedUserId := DecryptParamId(c, "userId", false)
	var user struct {
		EncId string `form:"user_id"`
		Id    int64  `form:"-"`
	}
	err := c.ShouldBind(&user)
	if err != nil {
		utils.HandleError(c, "error binding form", err)
		return
	}

	var event *models.Event
	if user.EncId != "" {
		user.Id, err = crypto.NDecrypt(user.EncId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Failed to decrypt the user_id",
			})
			return
		}
		event, err = models.GetEventsById(decryptedId, user.Id)
	} else {
		event, err = models.GetEventsById(decryptedId)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "oops something went wrong",
			"data":    "could not get event-->" + err.Error(),
		})
		return
	}

	city, state, err := GetCityAndState(event.StateId, event.CityId)
	if err != nil {
		utils.HandleError(c, "Unable to fetch city and state", err)
		return
	}

	// Check if event logo is valid
	eventLogoPath := "public/" + event.LogoPath
	defaultLogoPath := "public/static/staticLogo.jpg"

	if event.LogoPath == "" || !fileExists(eventLogoPath) {
		eventLogoPath = defaultLogoPath
	}

	// Fetch event images using GetEventImages logic
	query := `SELECT image, image_original_name FROM event_has_image WHERE event_id = $1`
	rows, err := database.DB.Query(query, decryptedId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch event images", err)
		return
	}
	defer rows.Close()

	type EventImage struct {
		Path         string `json:"path"`
		OriginalName string `json:"original_name,omitempty"`
	}

	var images []EventImage
	for rows.Next() {
		var imagePath string
		var originalName sql.NullString

		if err := rows.Scan(&imagePath, &originalName); err != nil {
			utils.HandleError(c, "Failed to scan image row", err)
			return
		}

		images = append(images, EventImage{
			Path: strings.ReplaceAll(imagePath, "\\", "/"),
			OriginalName: func() string {
				if originalName.Valid {
					return originalName.String
				}
				return ""
			}(),
		})
	}

	// Fetch event sponsors
	sponsorQuery := `SELECT id, sponsor_title, sponsor_logo FROM event_has_sponsors WHERE event_id = $1`
	sponsorRows, err := database.DB.Query(sponsorQuery, decryptedId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch event sponsors", err)
		return
	}
	defer sponsorRows.Close()

	var sponsors []gin.H
	for sponsorRows.Next() {
		var sponsorID int
		var sponsorTitle, sponsorLogo string

		if err := sponsorRows.Scan(&sponsorID, &sponsorTitle, &sponsorLogo); err != nil {
			utils.HandleError(c, "Failed to scan sponsor row", err)
			return
		}

		sponsors = append(sponsors, gin.H{
			"sponsor_id":    sponsorID,
			"sponsor_title": sponsorTitle,
			"sponsor_logo":  sponsorLogo,
		})
	}

	var ExistingGameEntries []models.GameEntry
	if decryptedUserId > 0 {
		// Fetch all saved games in one query
		ExistingGameEntries, err = models.GetSavedGames(decryptedUserId, decryptedId)
		if err != nil {
			utils.HandleError(c, "Error fetching your saved participation details", err)
			return
		}
		for i := range ExistingGameEntries {
			ExistingGameEntries[i], err = EncryptGameEntry(ExistingGameEntries[i])
			if err != nil {
				utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error encrypting existing game entries-->%v", err))
			}
		}
		event.CreatedByEncId = crypto.NEncrypt(event.CreatedById)
		encryptedGames := GetEncryptedGames(event)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Event retrieved successfully",
			"data": gin.H{
				"id":                          crypto.NEncrypt(decryptedId),
				"name":                        event.Name,
				"created_by_id":               event.CreatedByEncId,
				"created_by_role":             event.CreatedByRole,
				"created_by_name":             event.CreatedByName,
				"created_by_email":            event.CreatedByEmail,
				"from_date":                   event.FromDate,
				"to_date":                     event.ToDate,
				"state":                       state.Name,
				"city":                        city.Name,
				"venue":                       event.Venue,
				"fees":                        event.Fees,
				"about":                       event.About,
				"games":                       encryptedGames,
				"facebook_link":               event.FacebookLink,
				"instagram_link":              event.InstagramLink,
				"linkedin_link":               event.LinkedinLink,
				"google_map_link":             event.GoogleMapLink,
				"last_registration_date":      event.LastRegistrationDate,
				"start_registration_date":     event.StartRegistrationDate,
				"logo":                        eventLogoPath,
				"status":                      event.Status,
				"created_at":                  event.CreatedAt,
				"updated_at":                  event.UpdatedAt,
				"sponsors":                    sponsors,
				"images":                      images,
				"title":                       event.LevelOfCompetitionTitle,
				"saved_participation_details": ExistingGameEntries,
			},
		})
		return
	}
	event.CreatedByEncId = crypto.NEncrypt(event.CreatedById)
	encryptedGames := GetEncryptedGames(event)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event retrieved successfully",
		"data": gin.H{
			"id":                      crypto.NEncrypt(decryptedId),
			"name":                    event.Name,
			"created_by_id":           event.CreatedByEncId,
			"created_by_role":         event.CreatedByRole,
			"created_by_name":         event.CreatedByName,
			"created_by_email":        event.CreatedByEmail,
			"from_date":               event.FromDate,
			"to_date":                 event.ToDate,
			"state":                   state.Name,
			"city":                    city.Name,
			"venue":                   event.Venue,
			"fees":                    event.Fees,
			"about":                   event.About,
			"games":                   encryptedGames,
			"facebook_link":           event.FacebookLink,
			"instagram_link":          event.InstagramLink,
			"linkedin_link":           event.LinkedinLink,
			"google_map_link":         event.GoogleMapLink,
			"last_registration_date":  event.LastRegistrationDate,
			"start_registration_date": event.StartRegistrationDate,
			"logo":                    eventLogoPath,
			"status":                  event.Status,
			"created_at":              event.CreatedAt,
			"updated_at":              event.UpdatedAt,
			"sponsors":                sponsors,
			"title":                   event.LevelOfCompetitionTitle,
			"images":                  images,
		},
	})
}

func GetParticipatedEventById(c *gin.Context) {
	// Decrypt the encrypted ID
	decryptedEventID := DecryptParamId(c, "id", true)
	if decryptedEventID == 0 {
		return
	}
	// Decrypt the encrypted ID
	decryptedUserID := DecryptParamId(c, "userId", true)
	if decryptedUserID == 0 {
		return
	}
	event, err := models.GetEventsById(decryptedEventID)
	if err != nil {
		utils.HandleError(c, "Could not fetch event", err)
		return
	}

	participatedGames, err := models.GetSavedGames(decryptedUserID, decryptedEventID)
	if err != nil {
		utils.HandleError(c, "Failed to fetch participated games", err)
		return
	}

	for i := range participatedGames {
		participatedGames[i], err = EncryptGameEntry(participatedGames[i])
		if err != nil {
			utils.HandleError(c, "Failed to encrypt game entry", err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Participated event data retrieved successfully",
		"data":    event,
	})
}

func DeleteEvent(c *gin.Context) {
	// Decrypt the encrypted ID
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	_, err := models.DeleteEventByID(decryptedId)
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
		"message": "Event deleted successfully",
		"status":  "success",
	})
}

func GetGamesList(c *gin.Context) {
	List, err := models.GetGamesList()
	for i := range List {
		List[i].EncId = crypto.NEncrypt(List[i].ID)
	}
	if err != nil {
		utils.HandleError(c, "Error getting all Games", err)
		return
	}
	utils.HandleSuccess(c, "Games fetched successfully", map[string]any{"games": List})
}

func GetGameConfig(c *gin.Context) {
	var ConfigArr []models.GameConfig
	var err error

	err = c.ShouldBindJSON(&ConfigArr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": fmt.Sprintln("Invalid Input:", err),
		})
		return
	}

	for i, currentGameConfig := range ConfigArr {
		// Decrypt encrypted ID
		currentGameConfig.ID, err = crypto.NDecrypt(currentGameConfig.EncId)
		if err != nil {
			utils.HandleError(c, "Error Decrypting Id", err)
			return
		}

		// Get Game Name
		gameName, err := models.GetGameNameById(currentGameConfig.ID)
		if err != nil {
			utils.HandleError(c, "Error retrieving game name", err)
			return
		}
		currentGameConfig.GameName = gameName

		// Get Game Type IDs
		typeIDs, err := models.GetTypeByGameId(currentGameConfig.ID)
		if err != nil {
			utils.HandleError(c, "Error retrieving types by given Id", err)
			return
		}

		tx, _ := database.DB.Begin()

		_, currentGameConfig.AgeGroups, err = models.GetAgeGroupsByGameId(currentGameConfig.ID, tx)
		if err != nil {
			tx.Rollback()
			utils.HandleError(c, "Error fetching age groups", err)
			return
		}

		tx.Commit()

		for _, typeID := range typeIDs {
			gameType, err := models.GetGameTypeById(typeID)
			if err != nil {
				utils.HandleError(c, "Error retrieving game type", err)
				return
			}

			gameType.EncID = crypto.NEncrypt(typeID)
			currentGameConfig.Types = append(currentGameConfig.Types, *gameType)
		}

		ConfigArr[i] = currentGameConfig
	}

	utils.HandleSuccess(c, "Game config retrieved successfully", ConfigArr)
}

func GetAllModeratorsForEvent(c *gin.Context) {
	// Extract and validate parameters
	EventId := DecryptParamId(c, "eventId", true)
	if EventId == 0 {
		return
	}

	valid, err := IsEventActive(EventId)
	if err != nil {
		utils.HandleError(c, "Oops somthing went wrong", err)
	}
	if !valid {
		utils.HandleInvalidEntries(c, "Event not found", fmt.Errorf("no active event found"))
		return
	}

	Moderators, err := models.GetAllModeratorsForEvent(EventId)
	if err != nil {
		utils.HandleError(c, "Error fetching moderators", err)
	}
	var EncModerators []string
	for i := range Moderators {
		enc := crypto.NEncrypt(Moderators[i])
		EncModerators = append(EncModerators, enc)
	}
	utils.HandleSuccess(c, "moderators retreived successfully", EncModerators)

}

func EncryptGameEntry(Entry models.GameEntry) (models.GameEntry, error) {
	//var err error
	Entry.EventHasGameEncId = crypto.NEncrypt(Entry.EventHasGameId)
	for i, Type := range Entry.Types {
		Type.TypeEncId = crypto.NEncrypt(Type.TypeId)
		for j, team := range Type.Teams {
			team.TeamEncId = crypto.NEncrypt(team.TeamId)
			team.TeamCaptainEncId = crypto.NEncrypt(team.TeamCaptainID)
			team.EventHasGameTypeEncId = crypto.NEncrypt(team.EventHasGameTypeId)
			team.AgeGroupEncId = crypto.NEncrypt(team.AgeGroupId)
			team.TeamMemberEncId = []string{}
			for j := range team.TeamMemberIDs {
				team.TeamMemberEncId = append(team.TeamMemberEncId, crypto.NEncrypt(team.TeamMemberIDs[j]))
			}
			Type.Teams[j] = team
		}
		Entry.Types[i] = Type
	}
	return Entry, nil
}

func IsEventActive(id int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM events
			where id= $1 and status='Active'
		)
	`
	var exists bool
	err := database.DB.QueryRow(query, id).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
