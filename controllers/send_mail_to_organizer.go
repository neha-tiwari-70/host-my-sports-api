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

func SendMailToOrganizer(c *gin.Context) {
	idParam := c.Param("user_id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "User Id parameter is missing in the request.",
		})
		return
	}

	user_id, err := crypto.NDecrypt(idParam)
	if err != nil {
		utils.HandleError(c, "Unable to decrypt the user id.", err)
		return
	}

	userDetails, err := models.GetUserByID(int(user_id))
	if err != nil {
		utils.HandleError(c, "Failed to fetch the user's details", err)
		return
	}
	profileRedirection := fmt.Sprintf(`%s/profile`, os.Getenv("ENDUSER_FRONT"))

	err = SendBankDetailsMail(userDetails.Email, "Request to fill the bank details", profileRedirection, userDetails.Name)
	if err != nil {
		utils.HandleError(c, "Failed to send email", err)
		return
	}

	utils.HandleSuccess(c, "Email sent successfully")

}

func SendBankDetailsMail(to, subject, body, name string) error {
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
				<p>We hope you're having a great day!</p>
				<p>To ensure smooth and timely processing of your event-related payments, we kindly request you to complete your bank details in your Host My Sports profile.</p>
				<p>Adding your bank details is essential for us to transfer the registration fees collected from participants directly to your account.</p>

				<div class="btn-container">
					<a href="%s" class="btn">Complete Bank Details</a>
				</div>

				<p>If you have already filled in your bank details, you can safely ignore this message.</p>

				<p>Thank you for choosing Host My Sports.</p>
				<p>Warm regards,<br><strong>Team Host My Sports</strong></p>
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
