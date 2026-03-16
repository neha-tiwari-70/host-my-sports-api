package controllers

import (
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// VerifyEmail verifies the user's email address using a provided token.
// This function performs the following:
// 1. Binds the incoming JSON payload containing the token and email.
// 2. Verifies the token and extracts its claims.
// 3. Ensures the email in the token matches the email provided in the payload.
// 4. Retrieves the user from the database using the email address.
// 5. Updates the user's verification status.
// 6. Responds with either success or failure depending on the outcome.
//
// Params:
//   - c (*gin.Context): The Gin context to handle the HTTP request and response.
//
// Returns:
//   - HTTP response with a success or error message.
func VerifyEmail(c *gin.Context) {
	// Step 1: Define the payload structure to capture the incoming JSON data
	var payload struct {
		Token string `json:"token" binding:"required"`       // The verification token sent by the user
		Email string `json:"email" binding:"required,email"` // The email address to verify
	}

	// Step 2: Bind the incoming JSON to the payload struct
	if err := c.ShouldBindJSON(&payload); err != nil {
		// Handle binding errors (invalid JSON input)
		utils.HandleError(c, "Invalid input", err)
		return
	}

	// Step 3: Verify the token and extract claims
	claims, expErr := utils.VerifyToken(payload.Token)
	if expErr != nil && !strings.Contains(expErr.Error(), "Token is expired") {
		// Token verification failed (e.g., invalid or expired token)
		utils.HandleError(c, "Invalid token", expErr)
		return
	}

	// Step 4: Ensure that the email in the claims matches the email from the payload
	if claims["email"] != payload.Email {
		// Email mismatch between token and payload
		utils.HandleError(c, "Email does not match the token")
		return
	}

	userIdString := claims["user_id"].(string)
	UserId, err := crypto.NDecrypt(userIdString)
	if err != nil {
		utils.HandleError(c, "Decryption error")
		return
	}

	// Step 5: Retrieve the user from the database using the provided email
	user, err := models.GetUserByID(int(UserId))
	if err != nil {
		// User not found (e.g., the email does not exist in the database)
		utils.HandleError(c, "User not found")
		return
	}
	if user.Status == "Delete" {
		// User not found (e.g., the email does not exist in the database)
		utils.HandleInvalidEntries(c, "User not found")
		return
	}
	if user.Status == "Active" {
		utils.HandleInvalidEntries(c, "Link already used")
		return
	}
	if expErr != nil {
		utils.HandleInvalidEntries(c, "Link expired", err)
		return
	}

	user.Details, err = models.GetUserDetails(UserId)
	if err != nil {
		// User not found (e.g., the email does not exist in the database)
		utils.HandleError(c, "Error fetching user details", err)
		return
	}
	// Check for organization name duplication only if it's an organization
	if user.RoleSlug == "organization" {
		exists, err := models.IsOrganizationNameExists(*user.Details.OrganizationName, int(UserId))
		if err != nil && err.Error() != "user not found" {
			utils.HandleError(c, "Unable to check organization name's availability", err)
			return
		}
		if exists {
			// utils.HandleInvalidEntries(c, "An organization with this name already exists. Please choose a different name.", nil)

			c.JSON(http.StatusOK, gin.H{"status": "error",
				"message": "An organization with this name already exists. Please choose a different name.",
				"data":    crypto.NEncrypt(UserId)})
			return
		}
	}

	// get count for users with same mobile number as this that don't have the same id as existing user
	mobileCount, err := models.GetUserCountAttachedToMobileNumber(user.MobileNo, user.ID)
	if err != nil {
		utils.HandleError(c, "Error verifying mobile number", err)
		return
	}
	//if it's greater than or equal to 5 then return whith limit reached with this numeber error
	if mobileCount >= 5 {
		c.JSON(http.StatusOK, gin.H{"status": "error",
			"message": "This mobile number is no longer valid for use. Please use another one.",
			"data":    crypto.NEncrypt(UserId)})
		return
	}

	// Step 6: Update the user's verification status in the database
	// Uncomment the next line if you want to set the user's verification status to false after verification
	// user.IsVerified = true
	tx, err := database.DB.Begin()
	if err != nil {
		utils.HandleError(c, "database error")
		return
	}
	if err := models.UpdateUser(user, tx); err != nil {
		// Failed to update the user in the database
		utils.HandleError(c, "status not changed", err)
		tx.Rollback()
		return
	}
	tx.Commit()

	// Step 7: Encrypt the user ID to return a secure, encrypted response
	encryptedId := crypto.NEncrypt(int64(user.ID))

	// Step 8: Respond with a success message containing the encrypted user ID
	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"message":    "Email verified successfully",
		"data":       encryptedId,
		"isVerified": true,
	})
}
