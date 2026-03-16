package controllers

import (
	"fmt"
	"os"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// the api request is sent at only ..../v1/register ,the tokens are not added so
// we send the raw Query in payload
type pass struct {
	IsAdmin         bool   `json:"is_admin"`
	Code            string `json:"code" binding:"required"`
	Jwt             string `json:"jwt" binding:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
	RecaptchaToken  string `json:"recaptcha_token"`
}

// ResetPassword resets a user's password based on the provided JSON input.
// This function performs the following:
// 1. Validates the input fields (Code, Jwt, NewPassword, ConfirmPassword).
// 2. Parses the JWT token and verifies the user's identity.
// 3. Compares the provided code with the stored encrypted code.
// 4. Verifies the token's expiration and status.
// 5. Hashes the new password and updates the password in the database.
// 6. Deactivates the reset link by marking the entry in the forgot_users table as inactive.
//
// Params:
//   - c (gin.Context): The context of the HTTP request.
//
// Returns:
//   - None: The response is directly sent back to the client using Gin.
func ResetPassword(c *gin.Context) {
	var Pwds pass
	if err := c.ShouldBindJSON(&Pwds); err != nil {
		// Error in binding the JSON body to the struct
		// fmt.Println(err)
		utils.HandleError(c, "Invalid input", err)

		return
	}

	if err := utils.VerifyRecaptcha(c, Pwds.RecaptchaToken); err != nil {
		return
	}

	// Validate the input fields (e.g., ensure password requirements)
	errV := ValidateStruct(Pwds)
	if errV != "" {
		utils.HandleError(c, errV, nil)
		return
	}

	// Check if the new password and confirm password match
	if Pwds.NewPassword != Pwds.ConfirmPassword {
		utils.HandleInvalidEntries(c, "Passwords do not match")
		return
	}

	// Step 1: Get the user's email from the JWT token
	err := godotenv.Load()
	if err != nil {
		utils.HandleError(c, "Environment not loaded: ", err)
	}
	var secretKey = []byte(os.Getenv("JWT_SECRET"))
	token, err := jwt.Parse(Pwds.Jwt, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token's signing method matches
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil || !token.Valid {
		// Invalid or expired JWT token
		utils.HandleError(c, "Invalid token", err)
		return
	}

	// Step 2: Retrieve the user email from the claims in the JWT token
	var email string
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		email = fmt.Sprintf("%v", claims["email"])
	}

	// Step 3: Verify the token's expiry and user status
	ForgotUser, err := models.GetForgotUserByEmail(email)
	if err != nil {
		utils.HandleError(c, "No user found", err)
		return
	}

	// Compare the provided code with the stored encrypted code
	err = bcrypt.CompareHashAndPassword([]byte(ForgotUser.Code), []byte(Pwds.Code))
	if err != nil {
		utils.HandleError(c, "Hash compare failed: ", err)
		return
	}

	// Check if the reset link has expired
	if ForgotUser.Status == "Inactive" {
		utils.HandleInvalidEntries(c, "This link has expired.")
		return
	}

	// Adjust for timezone mismatch (default DB is UTC, but we need IST)
	//We're not storing timezone in DB, but time.now() gives india's time zone IST
	//and when expirey is loaded from DB it get the default timezone UTC
	//so the calculations are incorrect if we directly use time.now.After(ForgotUser.ExpireAt)
	//hence we have to set all the individual things in the following statement and set timzone of india
	exp := time.Date(
		ForgotUser.ExpireAt.Year(), ForgotUser.ExpireAt.Month(), ForgotUser.ExpireAt.Day(),
		ForgotUser.ExpireAt.Hour(), ForgotUser.ExpireAt.Minute(), ForgotUser.ExpireAt.Second(),
		ForgotUser.ExpireAt.Nanosecond(), time.Now().Location(),
	)
	if time.Now().After(exp) {
		utils.HandleInvalidEntries(c, "Token Expired")
		// Invalidate the reset link
		_, err = database.DB.Exec("UPDATE forgot_users SET status='Inactive' where email= $1", ForgotUser.Email)
		if err != nil {
			utils.HandleError(c, "Error updating status", err)
			return
		}
		return
	}

	// Step 4: Hash the new password and update it in the database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(Pwds.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.HandleError(c, "Error hashing password", err)
		return
	}
	Pwds.NewPassword = string(hashedPassword)

	// Update password based on user type (admin or normal user)
	if Pwds.IsAdmin {
		err = models.UpdatePassword("admin", ForgotUser.Email, Pwds.NewPassword)
	} else {
		err = models.UpdatePassword("users", ForgotUser.Email, Pwds.NewPassword)
	}

	// Step 5: Deactivate the reset link after the password has been updated
	if err != nil {
		utils.HandleError(c, "Error passing query", err)
		return
	}

	_, err = database.DB.Exec("UPDATE forgot_users SET status='Inactive' where email= $1", ForgotUser.Email)
	if err != nil {
		utils.HandleError(c, "Error wiping record", err)
		return
	}

	// Return success response
	utils.HandleSuccess(c, "Password reset successfully")
}
