package controllers

import (
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type ContactUs struct {
	Name    string `json:"name,omitempty" validate:"required,min=2"`
	Phone   string `json:"phone" binding:"required,len=10,numeric"`
	Email   string `json:"email,omitempty" validate:"required,email"`
	Subject string `json:"subject,omitempty" validate:"required"`
	Message string `json:"message,omitempty" validate:"required"`
}

func ContactUsFunc(c *gin.Context) {
	var contactUsData ContactUs

	if err := c.ShouldBindJSON(&contactUsData); err != nil {
		// fmt.Println(err)
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	errV := ValidateStruct(contactUsData)
	if errV != "" {
		utils.HandleError(c, errV)
		return
	}

	err := SendContactMail(contactUsData)
	if err != nil {
		utils.HandleError(c, "Failed to send message.", err)
		return
	}

	utils.HandleSuccess(c, "Thanks! We've received your message.")
}

func SendContactMail(contactUsDetails ContactUs) error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	// Retrieve SMTP contactUsData from environment variables
	emHost := os.Getenv("MAIL_SERVER")
	emPort := os.Getenv("MAIL_PORT")
	// emName := os.Getenv("MAIL_USERNAME")
	pass := os.Getenv("MAIL_PASSWORD")
	to := os.Getenv("MAIL_FROM_ADDRESS")
	// from := contactUsDetails.Email
	subject := contactUsDetails.Subject

	// Construct HTML email body with reset password link
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<title>Email Verification</title>
			<style>
				body {
					font-family: sans serif;
					background-color:rgb(255, 255, 255);
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
					background-color:#F0EFF4;
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
					text-align:center;
					display: inline-block;
					background-color:#2D3142;
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
				.message-box {
					background-color: #f0eff4;
					padding: 15px;
					border-left: 4px solid #2d3142;
					font-size: 12px;
					color: #444;
					white-space: pre-wrap;
					border-radius: 4px;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">Contact Us</div>
				<div class="email-content">
					 <h2 style="text-align: center">New Contact Us Message</h2>
					<p><strong>Name:</strong> %s</p>
					<p><strong>Email:</strong> %s</p>
					<p><strong>Phone:</strong> %s</p>
					<p><strong>Subject:</strong> %s</p>
					<p><strong>Message:</strong><br/></p>
					<div class="message-box">%s</div>
				</div>
				<div class="footer">© 2025 Host My Sports. All rights reserved.</div>
			</div>
		</body>
		</html>`, contactUsDetails.Name, contactUsDetails.Email, contactUsDetails.Phone, contactUsDetails.Subject, contactUsDetails.Message,
	)

	// Create the email message with headers and body
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		to,
		to,
		subject,
		htmlBody,
	))

	// Authenticate and send the email via SMTP
	auth := smtp.PlainAuth("", to, pass, emHost)
	err = smtp.SendMail(fmt.Sprintf("%s:%s", emHost, emPort), auth, to, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
