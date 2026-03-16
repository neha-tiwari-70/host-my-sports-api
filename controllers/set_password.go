package controllers

import (
	"fmt"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type setPass struct {
	EncID           string `json:"id" binding:"required"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
	RecaptchaToken  string `json:"recaptcha_token"`
}

// SetPassword handles the password reset process for a user.
// This function performs the following:
// 1. Validates the incoming user data (password and confirmation password).
// 2. Decrypts the user ID received in the request to identify the user.
// 3. Verifies that the password and confirmation password match.
// 4. Hashes the password for secure storage.
// 5. Updates the password in the database for the given user.
//
// Params:
//   - c (*gin.Context): The Gin context to handle the HTTP request and response.
//
// Returns:
//   - error: If any error occurs at any stage (e.g., validation failure, decryption failure, database update error).
func SetPassword(c *gin.Context) {
	var Pwds setPass

	// Step 1: Bind the incoming JSON data to the Pwds struct
	if err := c.ShouldBindJSON(&Pwds); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := utils.VerifyRecaptcha(c, Pwds.RecaptchaToken); err != nil {
		return
	}

	// Step 2: Validate the user input (password and confirmation password)
	errV := ValidateStruct(Pwds)
	if errV != "" {
		utils.HandleError(c, errV)
		return
	}

	// Step 3: Decrypt the encrypted user ID received in the request
	decryptedId, err := crypto.NDecrypt(Pwds.EncID)
	if err != nil {
		// Handle error if decryption fails
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Failed to decrypt the ID",
		})
		return
	}

	// Step 4: Verify that the password and confirmation password match
	if Pwds.Password != Pwds.ConfirmPassword {
		utils.HandleInvalidEntries(c, "Passwords do not match")
		return
	}

	// Step 5: Check if the user exists in the database using the decrypted ID
	user, err := models.GetUserByID(int(decryptedId))
	if err != nil {
		utils.HandleError(c, "User not loaded using id", err)
		return
	}
	switch user.Status {
	case "Delete":
		utils.HandleInvalidEntries(c, "user does not exist")
		return
	case "Active":
		utils.HandleInvalidEntries(c, "link already used")
		return
	}

	// Step 6: Hash the new password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(Pwds.Password), bcrypt.DefaultCost)
	if err != nil {
		// Handle error if hashing the password fails
		utils.HandleError(c, "Error hashing password", err)
		return
	}

	// Update the password in the Pwds struct with the hashed password
	Pwds.Password = string(hashedPassword)

	// Step 7: Update the user's password in the database
	query := `UPDATE users SET password= $1, status=$2, email_status=$3 WHERE id=$4`
	_, err = database.DB.Exec(query, Pwds.Password, "Active", "Verified", decryptedId)
	if err != nil {
		// Handle error if the database update fails
		utils.HandleError(c, "Database Error", err)
		return
	}
	// Step 7.2: Update the user_details status in the database
	err = models.SyncUserDetailStatusByUserId(decryptedId)
	if err != nil {
		utils.HandleError(c, "Database Error", err)
		return
	}

	// Step 8: Return a success message after the password is updated successfully
	utils.HandleSuccess(c, "Password set successfully")
}
