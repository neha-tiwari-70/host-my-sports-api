package controllers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// ForgotPassword handles the process of generating and sending a password reset link.
// This function performs the following:
// 1. Validates the incoming request payload (email & isAdmin).
// 2. Checks if the email exists in either the admin or user database based on isAdmin flag.
// 3. Generates a random reset code, encrypts and hashes it.
// 4. Stores or updates the code and metadata in the forgot_users table.
// 5. Builds a password reset link using the encrypted code.
// 6. Sends the reset link to the user via email.
//
// Params:
//   - c (*gin.Context): The HTTP request context containing the JSON payload.
//
// Returns:
//   - JSON response with either success message or error details.
func ForgotPassword(c *gin.Context) {
	var credentials models.ForgotUser

	// Parse and bind incoming JSON request
	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request", "data": err.Error()})
		sentry.CaptureException(err)
		return
	}

	// Step 1: Verify reCAPTCHA token
	if err := utils.VerifyRecaptcha(c, credentials.RecaptchaToken); err != nil {
		return
	}

	// Step 2: Validate input fields
	if errStr := ValidateStruct(credentials); errStr != "" {
		utils.HandleError(c, fmt.Sprintf("Validation Error: %v", errStr))
		return
	}

	// Step 3: Check if user/admin exists
	var userName *models.User
	var err error

	if credentials.IsAdmin {
		admin, err := models.GetAdminByEmail(credentials.Email)
		if err != nil {
			utils.HandleError(c, "Admin email not found", err)
			return
		}
		userName = &models.User{Name: admin.Name, Email: admin.Email} // Convert to common format
	} else {
		userName, err = models.GetUserByEmail(credentials.Email)

		if err != nil {
			if err.Error() == "user not found" {
				utils.HandleInvalidEntries(c, "User email not found", err)
				return
			}
			utils.HandleError(c, "User email not found", err)
			return
		}
	}

	// Step 4: Prepare status and code
	credentials.Status = "Active"
	code := string(codeGen())

	if err := godotenv.Load(); err != nil {
		utils.HandleError(c, "Error loading .env file", err)
		return
	}

	key := os.Getenv("AES_KEY")
	encryptedCode, err := encrypt(code, key)
	if err != nil {
		utils.HandleError(c, "Encryption error", err)
		return
	}

	hashedCode, err := bcrypt.GenerateFromPassword([]byte(encryptedCode), bcrypt.DefaultCost)
	if err != nil {
		utils.HandleError(c, "Error hashing code", err)
		return
	}
	credentials.Code = string(hashedCode)

	// Step 5: Save or update forgot user record
	existingForgotUser, _ := models.GetForgotUserByEmail(credentials.Email)
	if existingForgotUser != nil {
		query := `
			UPDATE forgot_users
			SET code=$1, updated_at=$2, expire_at=$3, status=$4
			WHERE email=$5
		`
		_, err = database.DB.Exec(query, credentials.Code, time.Now(), time.Now().Add(15*time.Minute), "Active", credentials.Email)
		if err != nil {
			utils.HandleError(c, "Failed to update forgot user", err)
			return
		}
	} else {
		if _, err = models.CreateForgotUser(&credentials); err != nil {
			utils.HandleError(c, "Failed to create forgot user", err)
			return
		}
	}

	// Step 6: Generate reset link
	resetLink, err := BuildLink(credentials, encryptedCode)
	if err != nil {
		utils.HandleError(c, "Error generating reset link", err)
		return
	}

	// Step 7: Send the reset email
	if err := Send_mail(userName.Name, resetLink, "HostMySports Password Reset", credentials); err != nil {
		utils.HandleError(c, "Failed to send reset email", err)
		return
	}

	utils.HandleSuccess(c, "Reset link sent successfully. Please check your email.")
}

// codeGen generates a 6-digit numeric code as a string.
func codeGen() string {
	x, _ := rand.Int(rand.Reader, big.NewInt(999999))
	Valid_code := fmt.Sprintf("%06d", x)
	return Valid_code
}

// encrypt encrypts the given text using AES encryption in GCM mode and encodes the result as a base64 string.
//
// Params:
//   - text (string): The plain text to encrypt.
//   - key (string): The AES key used for encryption.
//
// Returns:
//   - string: The base64 encoded encrypted text.
//   - error: If any error occurs during the encryption process.
func encrypt(text, key string) (string, error) {
	// Create an AES cipher block
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// Create a GCM (Galois Counter Mode) cipher mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize()) // Static nonce for now; should be randomly generated for better security

	ciphertext := aesGCM.Seal(nil, nonce, []byte(text), nil)

	// Encode ciphertext to a URL-safe format
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// BuildLink generates a password reset link with an embedded JWT token.
// This function performs the following:
// 1. Loads environment variables and retrieves the JWT secret key.
// 2. Creates a URL for the password reset page based on whether the user is an admin or not.
// 3. Creates a JWT token containing the user's email as a claim.
// 4. Appends the reset code and the JWT token as query parameters to the URL.
//
// Params:
//   - credentials (models.ForgotUser): The user's credentials, including email and admin status.
//   - code (string): The encrypted reset code to be included in the URL.
//
// Returns:
//   - *url.URL: The generated password reset link.
//   - error: If any error occurs during the process (e.g., environment variable loading or URL generation).
func BuildLink(credentials models.ForgotUser, code string) (*url.URL, error) {
	var claims utils.Claims
	claims.UserEmail = credentials.Email
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	var secretKey = []byte(os.Getenv("JWT_SECRET"))
	var link *url.URL
	if credentials.IsAdmin {
		link, _ = url.Parse(os.Getenv("ADMIN_FRONT") + "/auth/reset-password")
	} else {
		link, _ = url.Parse(os.Getenv("ENDUSER_FRONT") + "/reset-password")
	}

	// Create JWT token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	em, _ := token.SignedString(secretKey)

	// Add the encrypted reset code and JWT token to the URL as query parameters
	link.RawQuery = "token=" + code + "&em=" + em
	return link, nil
}

// Send_mail sends a password reset email to the user.
// This function performs the following:
// 1. Loads environment variables for SMTP server credentials.
// 2. Creates an HTML email body with a password reset link.
// 3. Sends the email to the user's registered email address using the SMTP server.
//
// Params:
//   - name (string): The name of the user to be addressed in the email.
//   - link (*url.URL): The password reset URL with query parameters.
//   - subject (string): The subject of the email.
//   - user (models.ForgotUser): The user's information including email.
//
// Returns:
//   - error: If any error occurs during email sending (e.g., failed SMTP connection or email sending).
func Send_mail(name string, link *url.URL, subject string, user models.ForgotUser) error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	// Retrieve SMTP credentials from environment variables
	emHost := os.Getenv("MAIL_SERVER")
	emPort := os.Getenv("MAIL_PORT")
	from := os.Getenv("MAIL_USERNAME")
	pass := os.Getenv("MAIL_PASSWORD")
	to := user.Email

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
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">Host My Sports</div>
				<div class="email-content">
					<h3>Hello!</h3>
					<p>Dear %s,</p>
					<p>Follow the link below to reset your password.</p>
					<div class="btn-container">
						<a href="%s" class="btn">Reset Password</a>
					</div>
					<p>Thanks,<br>Host My Sports</p>
				</div>
				<div class="footer">© 2025 Host My Sports. All rights reserved.</div>
			</div>
		</body>
		</html>`,
		name, link.String(),
	)

	// Create the email message with headers and body
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		from,
		to,
		subject,
		htmlBody,
	))

	// Authenticate and send the email via SMTP
	auth := smtp.PlainAuth("", from, pass, emHost)
	err = smtp.SendMail(fmt.Sprintf("%s:%s", emHost, emPort), auth, from, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
