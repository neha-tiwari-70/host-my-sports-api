package controllers

import (
	"database/sql"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

// GetModeratorByUserCode fetches moderator data by user code for a given organization.
// This function performs the following:
//  1. Extracts and decrypts route parameters (userCode, organizationId).
//  2. Retrieves the user associated with the userCode.
//  3. Checks if the user is already registered as a moderator for the given organization.
//  4. Returns user details and moderator status.
//
// Params:
//   - c (*gin.Context): HTTP context from Gin.
//
// Returns:
//   - JSON response with moderator details and their status.
func GetModeratorByUserCode(c *gin.Context) {
	// Extract route parameters
	userCode, exists := c.Params.Get("userCode")
	if !exists {
		utils.HandleError(c, "Invalid request-->missing userCode param")
		return
	}

	// Extract and decrypt the encrypted organization ID
	organizationId := DecryptParamId(c, "organizationId", true)
	if organizationId == 0 {
		return
	}
	// Extract and decrypt the encrypted event ID
	EventId := DecryptParamId(c, "eventId", true)
	if EventId == 0 {
		return
	}

	// Fetch user by userCode (without age and details)
	user, _, err := models.GetUserByUserCode(userCode, false, false)
	if err == sql.ErrNoRows {
		utils.HandleInvalidEntries(c, "Could not find any user with the user code you entered", err)
		return
	} else if err != nil {
		utils.HandleError(c, "Could not find any user with the user code you entered", err)
		return
	}
	EncId := crypto.NEncrypt(int64(user.ID))

	//fetch event
	Event, err := models.GetEventByID(int(EventId))
	if err != nil {
		utils.HandleError(c, "Could not find any event", err)
		return
	}
	Event.EncID = crypto.NEncrypt(int64(Event.ID))

	// Check if user is already a moderator for this organization in this game or event
	tx, _ := database.DB.Begin()
	moderatorAlreadyRegistered, err := models.CheckOrganizationHasModerator(organizationId, int64(user.ID), EventId, tx)
	if err != nil {
		utils.HandleError(c, "Oops something went wrong", err)
		return
	}
	if moderatorAlreadyRegistered {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "You already have " + user.Name + " as a moderator for '" + Event.Name + "' event",
			"data":    map[string]string{"id": EncId},
		})
		return
	}
	tx.Commit()

	utils.HandleSuccess(c, "user found!", gin.H{"id": EncId, "name": user.Name, "isExistingModerator": moderatorAlreadyRegistered})
}

// GetAllModeratorsForOrg retrieves all moderators linked to a given organization.
// This function performs the following:
//  1. Extracts and decrypts the organization ID.
//  2. Retrieves all moderators associated with the organization.
//  3. Encrypts the moderator IDs.
//  4. Returns the list of encrypted moderators.
//
// Params:
//   - c (*gin.Context): HTTP context from Gin.
//
// Returns:
//   - JSON response with list of moderators and their encrypted IDs.
func GetAllModeratorsForOrg(c *gin.Context) {
	// Extract and decrypt the encrypted organization ID
	organizationId := DecryptParamId(c, "organizationId", true)
	if organizationId == 0 {
		return
	}

	// Fetch all moderators
	resArr, err := models.GetAllModeratorsForOrg(organizationId)
	if err != nil {
		utils.HandleError(c, "Could not fetch all moderators", err)
		return
	}

	type encData struct {
		Name       string `json:"name"`
		EncId      string `json:"id"`
		EncEventId string `json:"event_id"`
		Status     string `json:"status"`
	}

	// Encrypt moderator IDs and build response array
	var encArr []encData
	for _, entry := range resArr {
		enc := crypto.NEncrypt(entry.Id)
		encEvent := crypto.NEncrypt(entry.EventId)
		encArr = append(encArr, encData{Name: entry.Name, EncId: enc, EncEventId: encEvent, Status: entry.Status})
	}

	utils.HandleSuccess(c, "successfully fetched all moderators", encArr)
}

// AddModerator adds a user as a moderator for a specific organization.
// This function performs the following:
//  1. Extracts and decrypts the organization and moderator IDs from the route.
//  2. Checks if the user is already a moderator in that organization.
//  3. Adds the user as a moderator if not already assigned.
//
// Params:
//   - c (*gin.Context): HTTP context from Gin.
//
// Returns:
//   - JSON response indicating success or failure.
func AddModerator(c *gin.Context) {
	// Extract and decrypt the encrypted organization ID
	OrganizationId := DecryptParamId(c, "organizationId", true)
	if OrganizationId == 0 {
		return
	}

	// Extract and decrypt the encrypted moderator ID
	ModeratorId := DecryptParamId(c, "moderatorId", true)
	if ModeratorId == 0 {
		return
	}

	EventId := DecryptParamId(c, "eventId", true)
	if EventId == 0 {
		return
	}

	// Check if already assigned
	tx, _ := database.DB.Begin()
	OrganizerHasModerator, err := models.CheckOrganizationHasModerator(OrganizationId, ModeratorId, EventId, tx)
	tx.Commit()
	if err != nil {
		utils.HandleError(c, "Error chcking relationship", err)
		return
	}

	if OrganizerHasModerator {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "This user is already assigned as a moderator",
			"data":    map[string]bool{"OrganizerHasModerator": true},
		})
		return
	}

	// Add the moderator
	err = models.AddModerator(OrganizationId, ModeratorId, EventId)
	if err != nil {
		utils.HandleError(c, "Failed to add this moderator", err)
		return
	}

	utils.HandleSuccess(c, "Moderator added successfuly")
}

// UpdateModerator updates or deletes a moderator assignment for an organization.
// This function performs the following:
// 1. Extracts and validates the "action" parameter from the route.
// 2. Decrypts the organization and moderator encrypted IDs.
// 3. Verifies the moderator is currently assigned to the organization.
// 4. Calls the appropriate update or delete operation on the relationship.
//
// Supported Actions:
//   - "update": Updates moderator information.
//   - "delete": Removes the moderator assignment.
//
// Route Params:
//   - action (string): The action to perform ("update" or "delete").
//   - OrganizationId (string): Encrypted ID of the organization.
//   - ModeratorId (string): Encrypted ID of the moderator.
//
// Returns:
//   - 200 OK: If the operation is successful.
//   - 400/500: If parameters are missing, invalid, or a database error occurs.

func UpdateModerator(c *gin.Context) {
	// Extract the action to perform (update/delete)
	Action, exists := c.Params.Get("action")
	if !exists {
		utils.HandleError(c, "Invalid request-->missing action param")
		return
	} else if Action != "update" && Action != "delete" {
		utils.HandleError(c, "Invalid request-->action not supported")
		return
	}

	// Extract and decrypt the encrypted organization ID
	OrganizationId := DecryptParamId(c, "organizationId", true)
	if OrganizationId == 0 {
		return
	}

	EventId := DecryptParamId(c, "eventId", true)
	if EventId == 0 {
		return
	}

	// Extract and decrypt the encrypted moderator ID
	ModeratorId := DecryptParamId(c, "moderatorId", true)
	if ModeratorId == 0 {
		return
	}

	// Check if the moderator is currently assigned to the organization
	tx, _ := database.DB.Begin()
	OrganizerHasModerator, err := models.CheckOrganizationHasModerator(OrganizationId, ModeratorId, EventId, tx)
	tx.Commit()
	if err != nil {
		utils.HandleError(c, "Error checking relationship", err)
		return
	}
	if !OrganizerHasModerator {
		utils.HandleError(c, "The specified moderator is not currently assigned to this organization or one of them is inactive", err)
		return
	}

	// Perform the requested action (update/delete)
	err = models.UpdateModerator(OrganizationId, ModeratorId, EventId, Action)
	if err != nil {
		utils.HandleError(c, "Error "+Action+"ing.", err)
		return
	}

	utils.HandleSuccess(c, Action+"ed Successfully")
}

func GetEventByOrganizationId(c *gin.Context) {
	// Extract and decrypt the encrypted organization ID
	organizationId := DecryptParamId(c, "id", true)
	if organizationId == 0 {
		return
	}

	// Fetch all moderators
	resArr, err := models.GetEventByOrganizationId(organizationId)
	if err != nil {
		utils.HandleError(c, "Could not fetch all events", err)
		return
	}
	// Encrypt moderator IDs and build response array
	var encArr []models.EncData
	for _, entry := range resArr {
		enc := crypto.NEncrypt(entry.Id)
		encArr = append(encArr, models.EncData{Name: entry.Name, EncId: enc})
	}

	utils.HandleSuccess(c, "successfully fetched all events", encArr)
}
