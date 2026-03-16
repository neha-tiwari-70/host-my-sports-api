package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

// SignUpData defines the structure for user sign-up payloads.
// Validation tags are used with go-playground/validator to enforce rules based on IsAdmin flag.
type SignUpData struct {
	IsAdmin          bool   `json:"is_admin"`                                                                           // Determines if the user is an admin (true) or not (false)
	Name             string `json:"name,omitempty" validate:"required,min=2"`                                           // Name is always required and must be at least 2 characters
	MobileNo         string `json:"mobile_no,omitempty" validate:"required_if=IsAdmin false ,omitempty,len=10,numeric"` // Required for non-admins; must be 10-digit numeric
	Email            string `json:"email,omitempty" validate:"required_if=IsAdmin true ,omitempty,email"`               // Required for admins; must be a valid email format
	Password         string `json:"password,omitempty" validate:"required_if=IsAdmin true ,omitempty"`                  // Required for admins
	RoleSlug         string `json:"role_slug,omitempty" validate:"required_if=IsAdmin false"`                           // Required for non-admins; describes the user's role
	DOB              string `json:"dob,omitempty" validate:"omitempty"`
	Gender           string `json:"gender,omitempty" validate:"omitempty"`
	OrganizationName string `json:"organization_name,omitempty" validate:"omitempty,required_if=RoleSlug organization"`
	RecaptchaToken   string `json:"recaptcha_token"`
}

// Initialize the validator instance from go-playground/validator.
var validate = validator.New()

// customMessages maps specific validation errors to user-friendly error messages.
var customMessages = map[string]string{
	"ID.required":              "User does not exist",
	"Name.required":            "Name is required.",
	"Name.min":                 "Name must be at least 2 characters long.",
	"Email.required":           "Email is required.",
	"Email.required_if":        "Email is required.",
	"Email.email":              "Email must be a valid email address.",
	"Password.required":        "Password is required.",
	"Password.required_if":     "Password is required.",
	"Password.min":             "Password must be at least 8 characters long.",
	"NewPassword.required":     "New Password is required.",
	"NewPassword.min":          "Password must be at least 8 characters long.",
	"ConfirmPassword.required": "Please Confirm the password",
	"MobileNo.required":        "Mobile number is required.",
	"MobileNo.required_if":     "Mobile number is required.",
	"MobileNo.len":             "Mobile number must be exactly 10 digits.",
	"MobileNo.numeric":         "Mobile number must contain only digits.",
	"GameTypeID.required":      "Pick at least 1 game-type",
	"FromDate.required":        "From date required",
	"ToDate.required":          "To date required",
	"EventId.required":         "Event is required",
	"Venue.required":           "Venue is required",
	"Fees.required":            "Fees is required",
	"Games.required":           "Games is required",
	"Teams.required":           "Teams are required",
	"Teams.min":                "Not Enough Teams To Schedule Matches",
	"Phone.required":           "Phone is required",
	"Phone.len":                "Phone must be exactly 10 digits.",
	"Phone.numeric":            "Phone must contain only digits.",
	"Subject.required":         "Subject is required.",
	"Message.required":         "Message is required.",
	// "ScheduledDate.required":   "Scheduled Date is required",
	// "ScheduledDate.datetime":   "Scheduled Date must be in YYYY-MM-DD format",
	// "Venue.min":                "Venue must be at least 3 characters long",
	// "VenueLink.required":       "Venue Link is required",
	// "StartTime.required":       "Start time is required",
	// "StartTime.datetime":       "Start time must be in HH:MM format",
	// "EndTime.datetime":         "End time must be in HH:MM format",
	// "EndTime.required":         "End time is required",
}

// ValidateStruct validates a struct using go-playground/validator
// and returns a human-readable error message (customized if available).
func ValidateStruct(s interface{}) string {
	// Perform the actual validation.
	err := validate.Struct(s)
	if err == nil {
		// No validation errors
		return ""
	}

	// Assert the error to ValidationErrors type
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// Return a generic message if casting fails
		return "Invalid input"
	}

	// Get the first validation error from the list
	ve := validationErrors[0]

	// Construct the key for custom message lookup, e.g., "Email.required"
	key := ve.Field() + "." + ve.Tag()

	// Check if a custom message is defined for this error
	if message, exists := customMessages[key]; exists {
		return message
	}

	// Fallback to default error message if no custom one is found
	return ve.Error()
}

// GenerateUserCode creates an 8-character unique alphanumeric user code
func GenerateUserCode() string {
	bytes := make([]byte, 4) // 4 bytes = 8 characters in hex
	_, err := rand.Read(bytes)
	if err != nil {
		return "USR12345" // Fallback user code
	}
	return hex.EncodeToString(bytes) // Convert bytes to hex string
}

// Register handles the registration process for both Admins and regular Users.
//
// For Admins:
//   - Requires name, email, and password.
//   - Password is hashed and stored securely.
//   - Creates a new admin record in the database.
//
// For Users:
//   - Requires name, mobile number, email (optional), and role_slug.
//   - Checks if the user already exists by email or mobile number.
//   - Handles various verification states (OTP pending, email pending, active).
//   - Generates a unique user code and OTP for verification.
//   - Creates a new user record in the database.
//
// Request Body: JSON payload matching SignUpData struct.
// Response: JSON object with status, message, and optional data.
func Register(c *gin.Context) {
	var data SignUpData

	// Step 1: Bind incoming JSON payload to the SignUpData struct
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	if err := utils.VerifyRecaptcha(c, data.RecaptchaToken); err != nil {
		return
	}

	// Step 2: Validate the struct using custom rules and messages
	errV := ValidateStruct(data)
	if errV != "" {
		utils.HandleError(c, errV, nil)
		return
	}

	// Step 3: Handle Admin Registration
	if data.IsAdmin {
		var admin = models.Admin{
			Name:     data.Name,
			Email:    data.Email,
			Password: data.Password,
		}

		// Hash the password before saving
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.HandleError(c, "Error hashing password", err)
			return
		}
		admin.Password = string(hashedPassword)

		// Save admin to database
		_, err = models.CreateAdmin(&admin)
		if err != nil {
			utils.HandleError(c, "Failed to register admin", err)
			return
		}

		// Encrypt admin ID before returning it
		eid := crypto.NEncrypt(int64(admin.ID))

		// Return success response
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Admin registered successfully",
			"data": gin.H{
				"id":         eid,
				"name":       admin.Name,
				"email":      admin.Email,
				"created_at": admin.CreatedAt,
				"updated_at": admin.UpdatedAt,
			},
		})
		return
	}

	// Step 4: Handle Non-Admin (User) Registration
	// Check if user already exists by Email
	userId := int64(0)
	existingUserByEmail, err := models.GetUserByEmail(data.Email)
	if err != nil && err.Error() != "user not found" {
		utils.HandleError(c, "Database error while checking user existence using email", err)
		return
	}

	if existingUserByEmail != nil {
		userId = existingUserByEmail.ID
		existingUserByEmail.Details, err = models.GetUserDetails(existingUserByEmail.ID)
		if err != nil {
			utils.HandleError(c, "Error fetching details", err)
			return
		}
	}

	// Use any of the existing records for further checks
	existingUser := existingUserByEmail

	// get count for users with same mobile number as this that don't have the same id as existing user
	mobileCount, err := models.GetUserCountAttachedToMobileNumber(data.MobileNo, userId)
	if err != nil {
		utils.HandleError(c, "Error verifying mobile number", err)
		return
	}
	//if it's greater than or equal to 5 the return whith limit reached with this numeber error
	if mobileCount >= 5 {
		utils.HandleInvalidEntries(c, "This mobile number is no longer valid for use. Please use another one.")
		return
	}

	// Step 5: New User Registration
	// If the user is an organization, skip the DOB and Gender fields
	var formattedDOB *string
	var gender *string

	if data.RoleSlug == "individual" {
		// Convert "25/05/1999" to "1999-05-25"
		dobInput := data.DOB
		if dobInput != "" {
			parsedDOB, err := time.Parse("02/01/2006", dobInput) // Ensure the format is DD/MM/YYYY
			if err != nil {
				utils.HandleInvalidEntries(c, "Invalid DOB format. Use DD/MM/YYYY")
				return
			}
			var x string
			formattedDOB = &x
			*formattedDOB = parsedDOB.Format("2006-01-02")
		}
		gender = &data.Gender
	}

	// Check for organization name duplication only if it's an organization
	if data.RoleSlug == "organization" {
		userId := 0
		if existingUser != nil {
			userId = int(existingUser.ID)
		}
		exists, err := models.IsOrganizationNameExists(data.OrganizationName, userId)
		if err != nil && err.Error() != "user not found" {
			utils.HandleError(c, "Unable to check organization name's availability", err)
			return
		}
		if exists {
			utils.HandleInvalidEntries(c, "An organization with this name already exists. Please choose a different name.", nil)
			return
		}
	}

	// Handle scenarios where user already exists but hasn't completed verification
	if existingUser != nil {
		if existingUser.Status == "Active" {
			c.JSON(http.StatusOK, gin.H{
				"user_status": "Active",
				"status":      "error",
				"message":     "User already exists. Please log in.",
			})
			return
		}
		tx, err := database.DB.Begin()
		if err != nil {
			utils.HandleError(c, "database error", fmt.Errorf("error initializing transaction: %v", err))
			return
		}
		isMobileChanged := data.MobileNo != existingUser.MobileNo
		if isMobileChanged {
			existingUser.OTPStatus = "Pending"
		}
		existingUser.Name = data.Name
		existingUser.MobileNo = data.MobileNo
		existingUser.RoleSlug = data.RoleSlug
		existingUser.Details.OrganizationName = &data.OrganizationName
		existingUser.Details.DOB = formattedDOB
		existingUser.Details.Gender = &data.Gender
		err = models.UpdateUser(existingUser, tx)
		if err != nil {
			utils.HandleError(c, "Error updating user", err)
			tx.Rollback()
			return
		}
		err = models.UpdateSignupDetails(*existingUser.Details, existingUser.ID, tx)
		if err != nil {
			utils.HandleError(c, "Error updating user", err)
			tx.Rollback()
			return
		}

		// If OTP is pending or expired, regenerate and send OTP
		if existingUser.OTPStatus == "Pending" || existingUser.OTPStatus == "Expired" {
			EncId := crypto.NEncrypt(existingUser.ID)
			otp, err := models.UpdateOTP(existingUser.ID)
			if err != nil {
				utils.HandleError(c, "Failed to generate OTP!", err)
				tx.Rollback()
				return
			}

			//NOTE - comment for testing
			err = utils.SendSMS(data.MobileNo, otp)
			if err != nil {
				utils.HandleError(c, "Failed to send OTP SMS", err)
				tx.Rollback()
				return
			}

			msg := "Mobile number changed"
			if !isMobileChanged {
				msg = "OTP verification pending"
			}
			c.JSON(http.StatusOK, gin.H{
				"data": map[string]interface{}{
					"id": EncId,
					// "otp": otp, //NOTE - uncomment for testing
				},
				"otp_status": existingUser.OTPStatus,
				"user":       existingUser,
				"status":     "error",
				"message":    msg + ". Please verify your OTP.",
			})
			tx.Commit()
			return
		}

		// If email verification is still pending
		if existingUser.EmailStatus == "Pending" {
			encryptedID := crypto.NEncrypt(int64(existingUser.ID))
			c.JSON(http.StatusOK, gin.H{
				"data": map[string]interface{}{
					"id": encryptedID,
				},
				"otp_status":   existingUser.OTPStatus,
				"email_status": existingUser.EmailStatus,
				"status":       "error",
				"message":      "Email verification pending. Please verify your email.",
			})
			tx.Commit()
			return
		}
	}

	// Generate a unique user code
	maxUserCode, err := models.GetMaxUserCode()
	if err != nil {
		utils.HandleError(c, "Failed to generate user code", err)
		return
	}

	user := models.User{
		Name:     data.Name,
		MobileNo: data.MobileNo,
		Email:    data.Email,
		RoleSlug: data.RoleSlug,
		UserCode: maxUserCode + 1,
		Details: &models.UserDetails{
			DOB:              formattedDOB,
			Gender:           gender,
			OrganizationName: &data.OrganizationName,
		},
	}

	// Save user to database
	createdUser, err := models.CreateUser(&user)
	if err != nil {
		utils.HandleError(c, "Failed to register user", err)
		return
	}
	// fmt.Println("Created User : ", createdUser)
	// Generate OTP for user verification
	otp, err := models.GenrateOTP(createdUser.ID)
	if err != nil {
		utils.HandleError(c, "Failed to generate OTP!", err)
		return
	}

	//NOTE - comment for testing
	err = utils.SendSMS(data.MobileNo, otp)
	if err != nil {
		utils.HandleError(c, "Failed to send OTP SMS", err)
		return
	}
	// Encrypt user ID before returning it
	eid := crypto.NEncrypt(int64(createdUser.ID))

	// Return success response with OTP
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "OTP has been sent to your mobile number.",
		"data": map[string]interface{}{
			"id":        eid,
			"user_code": user.UserCode,
			// "otp":       otp, //NOTE -uncomment for testing
		},
	})
}

// VerifyOTP checks the OTP for a registered user
func VerifyOTP(c *gin.Context) {
	var input struct {
		ID             string `json:"ID" binding:"required"`
		OTP            string `json:"otp" binding:"required"`
		RecaptchaToken string `json:"recaptcha_token"`
	}

	// Bind JSON input
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	if err := utils.VerifyRecaptcha(c, input.RecaptchaToken); err != nil {
		return
	}
	decId, err := crypto.NDecrypt(input.ID)
	if err != nil {
		utils.HandleError(c, "Decryption error", err)
		return
	}

	// Retrieve the user by mobile number
	user, err := models.GetUserByID(int(decId))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "User not found", "data": ""})
		sentry.CaptureException(fmt.Errorf("user not found: %v", err))
		return
	}

	// Retrieve the OTP details
	otpDetails, err := models.GetOTPByUserID(int(user.ID))
	if err != nil {
		utils.HandleError(c, "OTP not found", err)
		return
	}
	// fmt.Printf("Now : %v, expire: %v", time.Now(), otpDetails.ExpireAt)
	// fmt.Println(time.Now().After(otpDetails.ExpireAt))
	exp := time.Date(
		otpDetails.ExpireAt.Year(), otpDetails.ExpireAt.Month(), otpDetails.ExpireAt.Day(),
		otpDetails.ExpireAt.Hour(), otpDetails.ExpireAt.Minute(), otpDetails.ExpireAt.Second(),
		otpDetails.ExpireAt.Nanosecond(), time.Now().Location(),
	)
	// Check if OTP is expired
	if time.Now().After(exp) {
		// Generate a new OTP and update it in the database
		user.OTPStatus = "Expired"
		// fmt.Println("User : ", user)
		tx, err := database.DB.Begin()
		if err != nil {
			utils.HandleError(c, "database error")
			return
		}
		if err := models.UpdateUser(user, tx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": " ", "data": ""})
			tx.Rollback()
			return
		}
		tx.Commit()
		utils.HandleInvalidEntries(c, "Your otp is expired. Please click on Resend OTP.")
		return
	}

	// Verify the OTP
	if otpDetails.OTP != input.OTP {
		utils.HandleInvalidEntries(c, "Invalid OTP", err)
		return
	}

	// Update user status to 'active' (or any other appropriate status)
	user.OTPStatus = "Verified"
	tx, err := database.DB.Begin()
	if err != nil {
		utils.HandleError(c, "database error")
		return
	}
	if err := models.UpdateUser(user, tx); err != nil {
		utils.HandleError(c, "Failed to update status", err)
		tx.Rollback()
		return
	}
	tx.Commit()
	encryptedID := crypto.NEncrypt(int64(user.ID))
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "OTP verified successfully",
		"data": map[string]interface{}{
			"id": encryptedID,
		}})
}

func ResendOTP(c *gin.Context) {
	var input struct {
		Id string `json:"id" binding:"required"`
	}

	// Bind JSON input
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	decId, err := crypto.NDecrypt(input.Id)
	if err != nil {
		utils.HandleError(c, "Decryption error", err)
		return
	}
	// fmt.Println("Mobile no came is : ", input.MobileNumber)
	// Retrieve the user by mobile number
	user, err := models.GetUserByID(int(decId))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "User not found", "data": ""})
		return
	}

	// Generate a new OTP and update it in the database
	newOTP, err := models.UpdateOTP(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to generate OTP", "data": ""})
		return
	}

	//NOTE - comment for testing
	err = utils.SendSMS(user.MobileNo, newOTP)
	if err != nil {
		utils.HandleError(c, "Failed to send OTP SMS", err)
		return
	}
	// fmt.Println("otp updated and resent : ", newOTP)

	// Send the new OTP via SMS/Email (implement this in utils or external service)
	// err = utils.SendOTP(input.MobileNumber, newOTP)
	// if err != nil {
	// utils.HandleError(c, "Failed to send OTP", err)
	//     return
	// }

	//NOTE - uncomment for testing
	// c.JSON(http.StatusOK, gin.H{"status": "success",
	// "message": "OTP resent successfully",
	// "data":    map[string]interface{}{"otp": newOTP}})

	//NOTE - comment for testing
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "OTP resent successfully"})
}

// Login authenticates a user or admin based on the provided credentials
//
// For Admins:
//   - Requires email and password.
//   - Verifies the credentials and returns a JWT token on success.
//   - Responds with basic admin info and the token.
//
// For Users:
//
//   - Requires email and password.
//
//   - Verifies the credentials and checks user status (e.g. Pending, Deleted).
//
//   - Fetches user details and encrypts sensitive fields.
//
//   - Responds with complete user info and a JWT token.
//
// Request Body: JSON payload matching models.LoginUser struct.
//
// Response: JSON with login status, user/admin details, and JWT token.
func Login(c *gin.Context) {
	var loginData models.LoginUser

	if err := c.ShouldBindJSON(&loginData); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	//  Bypass reCAPTCHA in development/testing
	// if os.Getenv("BYPASS_RECAPTCHA") != "true" {
	// 	if err := utils.VerifyRecaptcha(loginData.RecaptchaToken); err != nil {
	// 		// fmt.Println("recaptcha")
	// 		 "reCAPTCHA failed", "data": err.Error()})
	// 		return
	// 	}
	// }
	if err := utils.VerifyRecaptcha(c, loginData.RecaptchaToken); err != nil {
		return
	}

	// Validate input fields
	validationErrors := ValidateStruct(loginData)
	if validationErrors != "" {
		c.JSON(http.StatusOK, gin.H{"data": "", "status": "error", "message": validationErrors})
		return
	}

	// Admin Login Flow
	if loginData.IsAdmin {
		// Find the admin by email
		admin, err := models.GetAdminByEmail(loginData.Email)
		if err != nil {
			utils.HandleInvalidEntries(c, "Invalid credentials", err)
			return
		}

		// Compare hashed password with the one provided
		err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(loginData.Password))
		if err != nil {
			utils.HandleInvalidEntries(c, "Invalid credentials", err)
			return
		}

		// Generate JWT token
		token, err := utils.GenerateToken(admin.Email, 24, admin.ID)
		if err != nil {
			utils.HandleError(c, "Failed to generate token", err)
			return
		}

		// Encrypt the admin ID before responding
		encryptedID := crypto.NEncrypt(int64(admin.ID))

		// Return success response with token and admin info
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Logged in successfully",
			"data": gin.H{
				"id":         encryptedID,
				"name":       admin.Name,
				"email":      admin.Email,
				"created_at": admin.CreatedAt,
				"updated_at": admin.UpdatedAt,
			},
			"token": token,
		})
		return
	}

	// User Login Flow
	user, err := models.GetUserByEmail(loginData.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password)) != nil {
		utils.HandleInvalidEntries(c, "Invalid credentials", err)
		return
	}

	if user.Status == "Delete" {
		utils.HandleInvalidEntries(c, "User doesn't exist.")
		return
	}
	if user.Status == "Pending" {
		utils.HandleInvalidEntries(c, "User registration incompleted.")
		return
	}

	user.Details, err = models.GetUserDetails(user.ID)
	if err != nil {
		utils.HandleError(c, "Error fetching user details", err)
		return
	}

	isModerator := false
	if isMod, err := models.IsUserModerator(int(user.ID)); err == nil && isMod {
		isModerator = true
	}

	encryptedID := crypto.NEncrypt(int64(user.ID))
	user.Details.EncState = crypto.NEncrypt(int64(user.Details.State))
	user.Details.EncUser_id = crypto.NEncrypt(int64(user.Details.User_id))
	user.Details.EncCity = crypto.NEncrypt(int64(user.Details.City))
	for i := range user.Details.Games {
		user.Details.Games[i].EncGameId = crypto.NEncrypt(user.Details.Games[i].GameId)
	}
	for i := range user.Details.ProfileRoles {
		user.Details.ProfileRoles[i].EncId = crypto.NEncrypt(user.Details.ProfileRoles[i].Id)
	}

	token, err := utils.GenerateToken(user.Email, 24, int(user.ID))
	if err != nil {
		utils.HandleError(c, "Failed to generate token", err)
		return
	}

	// fmt.Println("response", user.Details)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logged in successfully",
		"data": gin.H{
			"id":           encryptedID,
			"name":         user.Name,
			"user_code":    user.UserCode,
			"email":        user.Email,
			"mobile_no":    user.MobileNo,
			"otp_status":   user.OTPStatus,
			"email_status": user.EmailStatus,
			"status":       user.Status,
			"created_at":   user.CreatedAt,
			"updated_at":   user.UpdatedAt,
			"details":      user.Details,
			"role_slug":    user.RoleSlug,
			"is_moderator": isModerator,
		},
		"token": token,
	})
}

// ProfileUpdateController handles profile updates for both Admin and regular users.
// It binds incoming JSON data, validates it, and calls the appropriate update function
// based on the IsAdmin flag.
//
// Params:
//   - c (*gin.Context): Gin context carrying the HTTP request and response.
//
// Behavior:
//   - Validates incoming JSON for required fields.
//   - If IsAdmin is true, updates admin profile via updateAdminProfile.
//   - Otherwise, updates user profile via updateUserProfile.
//   - Sends appropriate success or error responses back to the client.
func ProfileUpdateController(c *gin.Context) {
	// Step 1: Get the updated data
	var NewData models.UpdatedUser

	// Bind the JSON payload to the NewData struct
	if err := c.ShouldBindJSON(&NewData); err != nil {
		utils.HandleError(c, "invalid payload", err)
		return
	}

	// Step 2: Check for validation and send the custom validation message
	validationErrors := ValidateStruct(NewData)
	if validationErrors != "" {
		utils.HandleError(c, "Validation Error", fmt.Errorf("%v", validationErrors))
		return
	}

	// Step 3: Update profile based on whether the user is an admin or not
	if NewData.IsAdmin {
		// If user is admin, update admin profile
		err := updateAdminProfile(&NewData)
		if err != nil {
			utils.HandleError(c, "Error updating admin", err)
			return
		}
	} else {
		// If user is not admin, update user profile
		err := updateUserProfile(&NewData)
		if err != nil {
			utils.HandleError(c, "Error updating user", err)
			return
		}
	}

	// Step 4: Respond with success
	utils.HandleSuccess(c, "Profile updated successfully")
}

// updateAdminProfile updates an admin's profile information in the database.
// It decrypts the admin ID before applying changes and updates the `name` and `mobile_no` fields.
// Params:
//   - NewData (*models.UpdatedUser): Struct containing the encrypted ID and updated profile fields.
//
// Returns:
//   - error: Returns nil if update is successful, otherwise returns an error.
func updateAdminProfile(NewData *models.UpdatedUser) error {
	// Implement the query on the table

	// Decrypt the encrypted admin ID received from the frontend
	decryptedId, err := crypto.NDecrypt(NewData.ID)
	if err != nil {
		return fmt.Errorf("failed to decrypt id")
	}

	// Prepare SQL query to update name and mobile number using the decrypted ID
	query := `
        UPDATE public.admin SET
            name = $1,
            mobile_no = $2
        WHERE id = $3;
    `

	// Execute the update query with the new name, mobile number, and decrypted ID
	_, err = database.DB.Exec(query, NewData.Name, NewData.MobileNo, decryptedId)
	if err != nil {
		// Return the error if the query fails
		return err
	}

	// Return nil to indicate success
	return nil
}

// updateUserProfile updates a regular user's profile details and their interested games.
// It updates data across `users`, `user_details`, and `user_has_interested_games` tables.
//
// Params:
//   - NewData (*models.UpdatedUser): Struct containing decrypted and encrypted data to be updated.
//
// Returns:
//   - error: If any database operation or decryption fails.
func updateUserProfile(NewData *models.UpdatedUser) error {
	// Decrypt the user ID from encrypted string
	decryptedId, err := crypto.NDecrypt(NewData.ID)
	if err != nil {
		fmt.Println("Failed to decrypt id")
	}

	// Step 1: Update the main user record (name and timestamp)
	query := `
        UPDATE public.users SET
            name = $1,
			updated_at = CURRENT_TIMESTAMP
        WHERE id = $2;
    `
	_, err = database.DB.Exec(query, NewData.Name, decryptedId)
	if err != nil {
		return err
	}

	// Step 2: Decrypt state and city IDs before updating user_details
	state, err := crypto.NDecrypt(NewData.Details.EncState)
	if err != nil {
		return err
	}
	city, err := crypto.NDecrypt(NewData.Details.EncCity)
	if err != nil {
		return err
	}

	// Step 3: Update user's detailed profile information
	// fmt.Println("New Data in query : ", NewData.Details.Gender)
	// fmt.Println("New Data in query : ", NewData.Details.Game)
	query = `UPDATE public.user_details SET
			nationality = $1,
			gender = $2,
			skill_level = $3,
			address = $4,
			facebook_link = $5,
			insta_link = $6,
			height = $7,
			weight = $8,
			state = $9,
			city = $10,
			aadhar_no = $11,
			dob = $12,
			coach_name= $13
		WHERE user_id = $14;`
	_, err = database.DB.Exec(query,
		NewData.Details.Nationality,
		NewData.Details.Gender,
		NewData.Details.Skill_level,
		NewData.Details.Address,
		NewData.Details.Facebook_link,
		NewData.Details.Insta_link,
		NewData.Details.Height,
		NewData.Details.Weight,
		state,
		city,
		NewData.Details.Aadhar_no,
		NewData.Details.DOB,
		NewData.Details.CoachName,
		decryptedId,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Step 4: Remove all existing interested games for the user
	_, err = database.DB.Exec(`DELETE FROM public.user_has_interested_games WHERE user_id = $1`, decryptedId)
	if err != nil {
		fmt.Println("Error deleting old games : ", err)
		return err
	}

	// fmt.Println("Games : ", NewData.Details.Games)

	// Step 5: Insert new interested games
	for _, game := range NewData.Details.Games {
		insertQuery := `
        INSERT INTO public.user_has_interested_games(user_id, game_id)
        VALUES($1, $2);
    `
		decGameId, err := crypto.NDecrypt(game.EncGameId)
		if err != nil {
			fmt.Println("Unable to decrypt game id")
		}
		_, err = database.DB.Exec(insertQuery, decryptedId, decGameId)
		if err != nil {
			fmt.Println("Error inserting into user_has_intereseted_games : ", err)
			return err
		}
	}

	// Step 4: Remove all existing profile roles
	_, err = database.DB.Exec(`DELETE FROM public.user_has_profile_roles WHERE user_id = $1`, decryptedId)
	if err != nil {
		fmt.Println("Error deleting old profile roles : ", err)
		return err
	}

	// Step 5: Insert new profile roles
	for _, profile_role := range NewData.Details.ProfileRoles {
		insertQuery := `
			INSERT INTO public.user_has_profile_roles(user_id, profile_role_id)
			VALUES($1, $2);
		`
		profile_role.Id, err = crypto.NDecrypt(profile_role.EncId)
		if err != nil {
			fmt.Println("Unable to decrypt profile role id")
		}

		_, err = database.DB.Exec(insertQuery, decryptedId, profile_role.Id)
		if err != nil {
			fmt.Println("Error inserting into user_has_profile role : ", err)
			return err
		}
	}

	// Return nil if all operations succeed
	return nil
}

func GetAllCoaches(c *gin.Context) {
	query := `
		SELECT u.id, u.name
		FROM users u
		JOIN user_has_profile_roles ur ON u.id = ur.user_id
		JOIN profile_roles pr ON pr.id = ur.profile_role_id
		WHERE pr.role ILIKE 'Coach' AND u.status = 'Active';
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		utils.HandleError(c, "Failed to fetch coaches", err)
		return
	}
	defer rows.Close()

	var coaches []map[string]interface{}

	for rows.Next() {
		var name string
		var id int64
		if err := rows.Scan(&id, &name); err != nil {
			utils.HandleError(c, "Error scanning coach data", err)
			return
		}

		coaches = append(coaches, map[string]interface{}{
			"name": name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Coaches fetched successfully",
		"data":    coaches,
	})
}

// ChangePassword handles the change password request for both users and admins.
// It validates the old password, checks if the new passwords match,
// hashes the new password, and updates it in the database.
//
// Params:
//   - c (*gin.Context): The Gin context that holds the request and response objects.
//
// This function handles two cases:
// 1. Admin password change: Updates the password for an admin.
// 2. User password change: Updates the password for a regular user.
func ChangePassword(c *gin.Context) {
	// Struct to bind incoming JSON data
	var passwordData struct {
		EncID           string `json:"id" binding:"required"`                 // Encrypted user or admin ID
		OldPassword     string `json:"old_password" binding:"required"`       // Old password
		NewPassword     string `json:"new_password" binding:"required,min=8"` // New password with minimum length of 8 characters
		ConfirmPassword string `json:"confirm_password" binding:"required"`   // Confirm new password
		IsAdmin         bool   `json:"is_admin"`                              // Boolean flag to indicate if it's an admin password change
	}

	// Bind the incoming JSON to passwordData struct
	if err := c.ShouldBindJSON(&passwordData); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	// Debugging log: Decrypt the user/admin ID
	decryptedId, err := crypto.NDecrypt(passwordData.EncID)
	if err != nil {
		fmt.Println("Failed to decrypt id")
	}

	// Step 1: If it's an admin, handle password change
	if passwordData.IsAdmin {
		// Retrieve admin by ID
		admin, err := models.GetAdminByID(int(decryptedId))
		if err != nil {
			utils.HandleError(c, "Admin not found", err)
			return
		}

		// Verify old password
		if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(passwordData.OldPassword)); err != nil {
			utils.HandleError(c, "Old password is incorrect", err)
			return
		}

		// Check if new passwords match
		if passwordData.NewPassword != passwordData.ConfirmPassword {
			utils.HandleError(c, "New password and confirm password do not match", err)
			return
		}

		// Step 2: Hash and update the new password
		hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(passwordData.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			utils.HandleError(c, "Error hashing new password", err)
			return
		}

		// Update password in the database for admin
		admin.Password = string(hashedNewPassword)
		if err := models.UpdatePassword("admin", admin.Email, admin.Password); err != nil {
			utils.HandleError(c, "Failed to update admin password", err)
			return
		}
	} else {
		// Step 3: If it's a user, handle password change
		// Retrieve user by ID
		user, err := models.GetUserByID(int(decryptedId))
		if err != nil {
			utils.HandleError(c, "User not found", err)
			return
		}

		// Check if new password is the same as old password
		if passwordData.OldPassword == passwordData.NewPassword {
			utils.HandleError(c, "New password cannot be same as old password", err)
			return
		}

		// Verify old password
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(passwordData.OldPassword)); err != nil {
			utils.HandleError(c, "Incorrect old password", err)
			return
		}

		// Check if new passwords match
		if passwordData.NewPassword != passwordData.ConfirmPassword {
			utils.HandleError(c, "New password and confirm password do not match", err)
			return
		}

		// Step 4: Hash and update the new password
		hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(passwordData.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			utils.HandleError(c, "Error hashing new password", err)
			return
		}

		// Update password in the database for user
		user.Password = string(hashedNewPassword)
		if err := models.UpdatePassword("users", user.Email, user.Password); err != nil {
			utils.HandleError(c, "Failed to update user password", err)
			return
		}
	}

	// Step 5: Respond with success message
	utils.HandleSuccess(c, "Password changed successfully")
	//c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Password changed successfully", "data": ""})
}

func OrganizationExistCheck(c *gin.Context) {
	query := strings.TrimSpace(c.Query("query"))
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query too short"})
		return
	}

	// Query the 'users' table where role_slug is 'organization'
	sqlQuery := `
        SELECT DISTINCT ON (LOWER(ud.organization_name))
		ud.organization_name
		FROM user_details ud
		JOIN users u ON u.id = ud.user_id
        WHERE LOWER(organization_name) LIKE LOWER($1)
		AND u.status = 'Active'
        ORDER BY LOWER(ud.organization_name)
        LIMIT 10
    `

	rows, err := database.DB.Query(sqlQuery, query+"%")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		fmt.Println(err)
		return
	}
	defer rows.Close()

	var suggestions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading results"})
			return
		}
		suggestions = append(suggestions, name)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Row iteration error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// func GetSession(c *gin.Context) any {
// 	// var sessionData models.User
// 	auth, _ := c.Get("ID")
// 	// if ok {
// 	// 	sessionData = auth.(models.User)
// 	// }
// 	// fmt.Println("session data : ", sessionData)
// 	return auth
// }
