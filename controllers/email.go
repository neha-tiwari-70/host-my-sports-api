package controllers

import (
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

// SendVerificationLink handles sending a verification link email to a user.
//
// This function performs the following:
// 1. Accepts a JSON payload with an encrypted user ID.
// 2. Decrypts the user ID.
// 3. Fetches user details by ID.
// 4. Generates a verification token using the user's email and ID.
// 5. Constructs a verification link for setting a password.
// 6. Sends an email to the user containing the verification link.
//
// Params:
//   - c (*gin.Context): Gin context containing the request and response objects.
//
// Request Body:
//   - EncryptedID (string): Encrypted user ID (required).
//
// Returns:
//   - JSON response with success or error status, including message and details.
func SendVerificationLink(c *gin.Context) {
	var payload struct {
		EncryptedID    string `json:"user_id" binding:"required"`
		RecaptchaToken string `json:"recaptcha_token"`
		// Email string `json:"email" binding:"required,email"` // Optional, handled via ID
	}

	// Debug print (can be removed in production)
	// fmt.Println(payload)

	// Bind the incoming JSON payload to the struct
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.HandleError(c, "Invalid input", err)
		// fmt.Println(err)
		return
	}

	if err := utils.VerifyRecaptcha(c, payload.RecaptchaToken); err != nil {
		return
	}

	// Decrypt the encrypted user ID
	decryptedId, err := crypto.NDecrypt(payload.EncryptedID)
	// fmt.Println(decryptedId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Failed to decrypt the ID",
		})
		return
	}

	// Retrieve user information from the database
	user, err := models.GetUserByID(int(decryptedId))
	if err != nil {
		utils.HandleError(c, "User not found", err)
		return
	}

	user.Details, err = models.GetUserDetails(decryptedId)
	if err != nil {
		// User not found (e.g., the email does not exist in the database)
		utils.HandleError(c, "Error fetching user details", err)
		return
	}
	// Check for organization name duplication only if it's an organization
	if user.RoleSlug == "organization" {
		exists, err := models.IsOrganizationNameExists(*user.Details.OrganizationName, int(decryptedId))
		if err != nil && err.Error() != "user not found" {
			utils.HandleError(c, "Unable to check organization name's availability", err)
			return
		}
		if exists {
			utils.HandleInvalidEntries(c, "An organization with this name already exists. Please choose a different name.", nil)
			return
		}
	}
	// Generate a token to include in the verification link
	verificationToken, err := utils.GenerateToken(user.Email, 0.25, int(user.ID), int(decryptedId))
	if err != nil {
		utils.HandleError(c, "Failed to generate verification token", err)
		return
	}

	// Construct the verification link with query parameters
	verificationLink := fmt.Sprintf("%s/set-password?token=%s&email=%s", os.Getenv("ENDUSER_FRONT"), verificationToken, user.Email)

	// Send the email containing the verification link
	err = sendEmail(user.Email, "Email Verification", verificationLink, user.Name)
	if err != nil {
		utils.HandleError(c, "Failed to send verification email", err)
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Verification link sent successfully",
		"data":    "",
	})
}

// sendEmail sends an HTML email using SMTP with a customized verification message.
//
// This function performs the following:
// 1. Constructs an HTML email body with the recipient's name and verification link.
// 2. Uses SMTP credentials and environment variables to send the email.
//
// Params:
//   - to (string): Recipient's email address.
//   - subject (string): Subject line of the email.
//   - body (string): Verification link or main content to embed.
//   - name (string): Recipient's name (used in the greeting).
//
// Returns:
//   - error: If the email sending fails via SMTP.
func sendEmail(to, subject, body, name string) error {
	mailServer := os.Getenv("MAIL_SERVER")
	mailPort := os.Getenv("MAIL_PORT")
	username := os.Getenv("MAIL_USERNAME")
	password := os.Getenv("MAIL_PASSWORD")
	from := os.Getenv("MAIL_FROM_ADDRESS")
	fromName := "Host My Sports"

	fromHeader := fmt.Sprintf("%s <%s>", fromName, from)

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<title>Email Verification</title>
			<style>
				body {
					font-family: sans-serif;
					background-color: #ffffff;
					margin: 0;
					padding: 0;
				}
				.container {
					padding: 20px;
					text-align: center;
				}
				.header {
					font-size: 20px;
					font-weight: bold;
					color: #333;
					margin-bottom: 15px;
				}
				.email-content {
					background-color: #F0EFF4;
					padding: 20px;
					border-radius: 5px;
					box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
					max-width: 500px;
					margin: 0 auto;
					text-align: left;
				}
				.email-content h2 {
					color: #333;
				}
				.email-content p {
					font-size: 12px;
					color: #555;
				}
				.btn {
					display: inline-block;
					background-color: #2D3142;
					color: white !important;
					padding: 8px 18px;
					font-size: 12px;
					text-decoration: none;
					border-radius: 5px;
					margin-top: 5px;
				}
				.btn-container {
					text-align: center;
					margin-top: 10px;
				}
				.footer {
					margin-top: 20px;
					font-size: 12px;
					color: #777;
					text-align: center;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">Host My Sports</div>
				<div class="email-content">
					<h3>Hello!</h3>
					<p>Dear %s,</p>
					<p>Thank you for signing up with HostMySports.com.<br>
					Please complete the registration process by clicking the button below.</p>
					<div class="btn-container">
						<a href="%s" class="btn">Verify Email</a>
					</div>
					<p>Thanks,<br>Host My Sports</p>
				</div>
				<div class="footer">© 2025 Host My Sports. All rights reserved.</div>
			</div>
		</body>
		</html>`,
		name, body,
	)

	// Set email headers
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		fromHeader,
		to,
		subject,
		htmlBody,
	))

	auth := smtp.PlainAuth("", username, password, mailServer)

	err := smtp.SendMail(fmt.Sprintf("%s:%s", mailServer, mailPort), auth, from, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
