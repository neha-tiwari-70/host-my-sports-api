package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sports-events-api/crypto"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// / Secret key used for signing and verifying the JWT
var secretKey = []byte(os.Getenv("JWT_SECRET"))

// Claims defines the structure of JWT payload for authentication.
// Includes user email, a token string, and standard JWT claims like expiration and issuer.
type Claims struct {
	UserEmail string `json:"email"`
	UserId    string `json:"user_id"`
	Token     string `json:"token"`
	jwt.StandardClaims
}

type RecaptchaResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Score       float64  `json:"score"`
	Action      string   `json:"action"`
}

// GenerateToken generates a JWT token with the user's email embedded in the claims.
// This function performs the following:
//  1. Constructs the JWT claims with expiration and issuer.
//  2. Signs the token using HS256 algorithm and the secret key.
//
// Params:
//   - userEmail (string): Email to embed in the token.
//   - user_id (...int): Optional user ID for future extensibility.
//
// Returns:
//   - string: The signed JWT token string.
//   - error: If token signing fails.
func GenerateToken(userEmail string, HoursTillExpiry float64, user_id ...int) (string, error) {
	claims := Claims{
		UserEmail: userEmail,
		UserId:    crypto.NEncrypt(int64(user_id[0])),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(HoursTillExpiry * float64(time.Hour))).Unix(), // Token valid for 24 hours
			Issuer:    os.Getenv("APP_URL"),                                                       // Set your app base URL or name here
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// VerifyToken verifies a JWT and returns its claims if valid.
// This function performs the following:
//  1. Parses the token string using the secret key.
//  2. Validates the signing method.
//  3. Returns the claims if token is valid.
//
// Params:
//   - tokenString (string): The JWT token to be verified.
//
// Returns:
//   - jwt.MapClaims: A map of token claims if valid.
//   - error: If the token is invalid or verification fails.
func VerifyToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token's signing method matches
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if (err != nil || !token.Valid) && !strings.Contains(err.Error(), "Token is expired") {
		return nil, err
	}

	// Return the claims (you can modify this part based on your token structure)
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, err
	}

	return nil, fmt.Errorf("invalid token")
}

func GetSessionUserId(c *gin.Context) int64 {
	id, ok := c.Get("user_id")
	if !ok {
		// user_id not found in context
		return 0
	}

	// Try type assertion to int64
	userID, ok := id.(int64)
	if !ok {
		// If it's not a int64 (e.g., float64), handle accordingly
		return 0
	}

	return userID
}

// GenerateVerificationToken creates a secure random token for email verification.
// Generates a 32-character hexadecimal string.
//
// Returns:
//   - string: The generated token string.
//   - error: If the random byte generation fails.
func GenerateVerificationToken() (string, error) {
	token := make([]byte, 16) // 16 bytes for a 32-character hex token
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}

// HandleError formats and sends a standardized error response via Gin context.
//
// Params:
//   - c (*gin.Context): The Gin context for the request.
//   - msg (string): The message to include in the error response.
//   - err (...error): Optional error to include in the response.
func HandleError(c *gin.Context, msg string, err ...error) {
	var ErrorString string
	if err == nil {
		ErrorString = fmt.Sprintf("%v", msg)
	} else {
		ErrorString = fmt.Sprintf("%v: %v", msg, err)
	}
	c.JSON(http.StatusOK, gin.H{"status": "error", "message": msg, "data": ErrorString})
	sentry.CaptureException(fmt.Errorf("%v", ErrorString))
}

// same as HandleError but does not alert sentry
func HandleInvalidEntries(c *gin.Context, msg string, err ...error) {
	var ErrorString string
	if err == nil {
		ErrorString = fmt.Sprintf("%v", msg)
	} else {
		ErrorString = fmt.Sprintf("%v: %v", msg, err)
	}
	c.JSON(http.StatusOK, gin.H{"status": "error", "message": msg, "data": ErrorString})
}

// HandleSuccess formats and sends a standardized success response via Gin context.
//
// Params:
//   - c (*gin.Context): The Gin context for the request.
//   - msg (string): Success message.
//   - data (...any): Optional data to return in the response.
func HandleSuccess(c *gin.Context, msg string, data ...any) {
	if data != nil {
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": msg, "data": data[0]})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": msg, "data": ""})
	}
}

// GenerateSlug creates a URL-friendly slug from a given string.
// Converts text to lowercase, replaces spaces with hyphens, and removes invalid characters.
//
// Params:
//   - name (string): The input string to convert.
//
// Returns:
//   - string: The generated slug.
func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	slug = re.ReplaceAllString(slug, "")
	slug = strings.Trim(slug, "-")
	return slug
}

// CreateFolder creates a directory if it does not already exist.
//
// Params:
//   - FolderPath (string): The full path to the folder to create.
//
// Returns:
//   - string: Message if a folder is created.
//   - error: If folder creation fails.
func CreateFolder(FolderPath string) (string, error) {
	// Check if the folder already exists
	if _, err := os.Stat(FolderPath); os.IsNotExist(err) {
		// Create the folder
		err := os.MkdirAll(FolderPath, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("failed to create folder: %v", err)
		}
		return ",New Folder Created for the image", nil
	}
	return "", nil
}

// CalculateAge calculates the age from a date of birth string.
//
// Params:
//   - Dob (string): Date of birth in "YYYY-MM-DD" format.
//
// Returns:
//   - int: Calculated age.
//   - error: If date parsing fails.
func CalculateAge(Dob string) (int, error) {
	birthTime, err := time.Parse("2006-01-02", Dob)
	if err != nil {
		return 0, err
	}

	// Get the current time
	now := time.Now()

	// Calculate age
	age := now.Year() - birthTime.Year()

	// Adjust if birthday hasn't occurred yet this year
	if now.YearDay() < birthTime.YearDay() {
		age--
	}
	return age, nil
}

// IsValidGoogleMapsURL checks if a given URL is a valid Google Maps URL.
//
// Params:
//   - urlString (string): The URL to validate.
//
// Returns:
//   - bool: True if valid, false otherwise.
func IsValidGoogleMapsURL(urlString string) bool {
	u, err := url.ParseRequestURI(urlString)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())
	path := strings.ToLower(u.Path)

	// Allow google.com / google.co.in / maps.google.com
	if strings.Contains(host, "google.") {
		validPaths := []string{
			"/maps", "/maps/", "/maps/place/", "/maps/search/",
			"/maps/dir/", "/maps/embed", "/maps/@",
		}
		for _, p := range validPaths {
			if strings.HasPrefix(path, p) {
				return true
			}
		}
	}

	// Accept goo.gl/maps/... or maps.app.goo.gl/... (with optional query params)
	shortPattern := regexp.MustCompile(`^(https?://)?(goo\.gl/maps|maps\.app\.goo\.gl)/[\w\-]+(\?.*)?$`)
	if shortPattern.MatchString(urlString) {
		return true
	}

	return false
}

// IsValidDate checks if a given date string is in valid "YYYY-MM-DD" format.
//
// Params:
//   - dateString (string): Date string to validate.
//
// Returns:
//   - bool: True if valid date format, false otherwise.
func IsValidDate(dateString string) bool {
	_, err := time.Parse("2006-01-02", dateString)
	return err == nil
}

// PrintDBJson prints the result of a SQL select query in pretty JSON format.
// Helpful for debugging queries within a transaction.
//
// Params:
//   - title (string): Label to prefix the JSON output.
//   - tableName (string): Name of the table to query.
//   - tx (*sql.Tx): Active SQL transaction.
//   - conditionString (...string): Optional WHERE condition.
func PrintDBJson(title string, tableName string, tx *sql.Tx, conditionString ...string) {
	querytest := `SELECT json_agg(t) FROM ( SELECT * FROM ` + tableName
	if conditionString != nil {
		querytest += ` WHERE ` + conditionString[0]
	}
	querytest += `) AS t;`

	var dataJson sql.NullString
	err := tx.QueryRow(querytest).Scan(&dataJson)
	if err != nil {
		fmt.Println("error:", err)
	}

	if !dataJson.Valid {
		fmt.Println(title, ": No data found.")
		return
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, []byte(dataJson.String), "", "   ")
	if err != nil {
		fmt.Println("Failed to format JSON:", err)
	}
	fmt.Println(title, ": ", prettyJSON.String())
}

// ParseDDMMYYYY parses a date string in "dd-mm-yyyy" format into a time.Time object.
//
// Params:
//   - dateStr (string): Date string in "dd-mm-yyyy" format.
//
// Returns:
//   - time.Time: Parsed date.
//   - error: If parsing fails.
func ParseDDMMYYYY(dateStr string) (time.Time, error) {
	return time.Parse("02-01-2006", dateStr)
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func VerifyRecaptcha(c *gin.Context, token string) error {
	// secret := "RECAPTCHA_SECRET_KEY"
	secret := os.Getenv("RECAPTCHA_SECRET_KEY")

	client := &http.Client{Timeout: 10 * time.Second}

	data := url.Values{}
	data.Set("secret", secret)
	data.Set("response", token)

	resp, err := client.PostForm("https://www.google.com/recaptcha/api/siteverify", data)
	if err != nil {
		HandleError(c, err.Error())
		return err
	}
	defer resp.Body.Close()

	var recaptchaResp RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&recaptchaResp); err != nil {
		HandleError(c, err.Error())
		return err
	}

	if !recaptchaResp.Success {
		fmt.Println("Recaptcha token received:", token)
		HandleInvalidEntries(c, "recaptcha verification failed")
		return errors.New("recaptcha verification failed")
	}

	if recaptchaResp.Score < 0.5 {
		HandleInvalidEntries(c, "recaptcha score too low")
		return errors.New("recaptcha score too low")
	}

	return nil
}

func GetSHA256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("could not parse claims")
	}

	return claims, nil
}
