package models

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sports-events-api/database"
	"sports-events-api/utils"
	"strings"
	"time"
)

// User represents a user in the system.
// This structure holds personal and account-related information for the user,
// including their user code, name, email, and status.
type User struct {
	ID       int64  `json:"-"`                                        // Internal ID, not exposed in API responses
	UserCode int    `json:"user_code"`                                // New field for user code
	Name     string `json:"name,omitempty" validate:"required,min=2"` // User's name
	// OrganizationName  sql.NullString `json:"-"`
	OrganizationName  *string      `json:"organization_name"`
	Email             string       `json:"email,omitempty" validate:"required,email"`              // User's email, must be valid
	Password          string       `json:"-" validate:"omitempty,min=8"`                           // User's password, validated if present
	MobileNo          string       `json:"mobile_no,omitempty" validate:"required,len=10,numeric"` // User's mobile number, 10 digits required
	Role              string       `json:"role,omitempty"`
	RoleSlug          string       `json:"role_slug,omitempty"`          // Role of the user in the system (e.g., admin, user)
	OTPStatus         string       `json:"otp_status,omitempty"`         // OTP status for authentication
	EmailStatus       string       `json:"email_status,omitempty"`       // Email verification status
	Status            string       `json:"status,omitempty"`             // User's current account status
	CreatedAt         time.Time    `json:"created_at,omitempty"`         // Timestamp of user creation
	UpdatedAt         time.Time    `json:"updated_at,omitempty"`         // Timestamp of last update
	VerificationToken string       `json:"verification_token,omitempty"` // Token for email verification
	Details           *UserDetails `json:"details" validate:"omitempty"` // Additional user details, optional
}

// InterestedGames represents a game that a user is interested in.
// It contains the game information along with user-specific fields like user ID and game ID.
type InterestedGames struct {
	ID        int64  `json:"-"`         // Internal ID, not exposed in API responses
	EncID     string `json:"-"`         // Encoded internal ID, for secure transfer
	UserId    int64  `json:"-"`         // Internal user ID, not exposed
	UserEncId string `json:"-"`         // Encoded user ID, for secure transfer
	EncGameId string `json:"id"`        // Encoded game ID
	GameId    int64  `json:"-"`         // Internal game ID
	GameName  string `json:"game_name"` // Name of the game
	Slug      string `json:"slug"`      // Slug (URL-friendly identifier) of the game
}

// UserDetails contains additional personal information for a user,
// such as nationality, gender, skill level, address, and social media links.
type UserDetails struct {
	ID                 int               `json:"-"`                                      // Internal ID, not exposed
	User_id            int               `json:"-" validate:"omitempty,numeric"`         // User's internal ID
	EncUser_id         string            `json:"user_id" validate:"omitempty,numeric"`   // Encoded user ID
	Nationality        *string           `json:"nationality" validate:"omitempty,min=2"` // Nationality of the user
	Gender             *string           `json:"gender" validate:"omitempty"`            // Gender of the user
	Skill_level        *string           `json:"skill_level" validate:"omitempty"`       // Skill level of the user in sports
	Address            *string           `json:"address" validate:"omitempty"`           // User's address
	Facebook_link      *string           `json:"facebook_link" validate:"omitempty"`     // Facebook profile link
	Insta_link         *string           `json:"insta_link" validate:"omitempty"`        // Instagram profile link
	Games              []InterestedGames `json:"interested_games,omitempty"`             // List of games user is interested in
	ProfileRoles       []ProfileRoles    `json:"profile_roles,omitempty"`
	OrganizationName   *string           `json:"organization_name,omitempty"`
	Height             *string           `json:"height"`                                  // Height of the user
	Weight             *string           `json:"weight"`                                  // Weight of the user
	Current_team       *string           `json:"current_team" validate:"omitempty,min=2"` // Current team of the user
	State              int               `json:"-" validate:"omitempty"`                  // Internal state ID
	City               int               `json:"-" validate:"omitempty"`                  // Internal city ID
	EncState           string            `json:"state" validate:"omitempty"`              // Encoded state ID
	EncCity            string            `json:"city" validate:"omitempty"`               // Encoded city ID
	Aadhar_no          *string           `json:"aadhar_no" validate:"omitempty"`          // Aadhar number for identity verification
	DOB                *string           `json:"dob" validate:"omitempty"`                // Date of birth of the user
	Profile_Image_Path *string           `json:"profile_image" validate:"omitempty"`      // Path to user's profile image
	Status             string            `json:"status"`                                  // User's current status (active, inactive)
	CreatedAt          time.Time         `json:"created_at"`                              // Timestamp of creation
	UpdatedAt          time.Time         `json:"updated_at"`                              // Timestamp of last update
	StateName          string            `json:"state_name,omitempty"`
	CityName           string            `json:"city_name,omitempty"`
	CoachName          string            `json:"coachName,omitempty"`
}

// LoginUser represents the credentials used for user login.
type LoginUser struct {
	IsAdmin        bool   `json:"is_admin"`                                      // Flag indicating if the user is an admin
	Email          string `json:"email,omitempty" validate:"omitempty,email"`    // User's email for login
	Password       string `json:"password,omitempty" validate:"omitempty,min=8"` // User's password for login
	RecaptchaToken string `json:"recaptcha_token"`
}

// UpdatedUser represents a user whose information has been updated.
type UpdatedUser struct {
	IsAdmin  bool         `json:"is_admin"`                                                // Flag indicating if the user is an admin
	ID       string       `json:"id"`                                                      // User's ID
	Name     string       `json:"name,omitempty" validate:"omitempty,min=2"`               // Updated user name
	Email    string       `json:"email,omitempty" validate:"omitempty,email"`              // Updated email
	MobileNo string       `json:"mobile_no,omitempty" validate:"omitempty,len=10,numeric"` // Updated mobile number
	Details  *UserDetails `json:"details"`                                                 // Updated user details
}

type ProfileRoles struct {
	EncId string `json:"id"`
	Id    int64  `json:"-"`
	Role  string `json:"role"`
	Slug  string `json:"slug"`
}

// GetUserByEmailOrMobile fetches a user based on their email or mobile number.
// This function performs the following:
// 1. Executes a query to search for a user by their email or mobile number.
// 2. If a user is found, it returns the user details including the user's ID, name, email, and status.
//
// Params:
//   - email (string): The email of the user to search for.
//   - mobile (string): The mobile number of the user to search for.
//
// Returns:
//   - *User: The user object containing the user details.
//   - error: If any error occurs during the query or data retrieval.
func GetUserByEmailOrMobile(email, mobile string) (*User, error) {
	var user User
	query := `SELECT id, name, email, mobile_no, otp_status, email_status, status FROM users WHERE email=$1 OR mobile_no=$2`
	err := database.DB.QueryRow(query, email, mobile).Scan(
		&user.ID, &user.Name, &user.Email, &user.MobileNo, &user.OTPStatus, &user.EmailStatus, &user.Status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No user found for the given email or mobile
		}
		return nil, err // Return error if query fails
	}
	return &user, nil // Return the user object if found
}

// CreateUser inserts a new user record into the users table with default status values.
// This function performs the following:
// 1. Sets CreatedAt and UpdatedAt timestamps if not already set.
// 2. Assigns default values: Status = "Pending", OTPStatus = "Pending", EmailStatus = "Pending".
// 3. Generates a slug from the provided RoleSlug.
// 4. Inserts the user into the users table and retrieves the generated ID.
// 5. Calls CreateDetails to insert a placeholder user details record.
//
// Params:
//   - user (*User): Pointer to the user struct containing the user's data.
//
// Returns:
//   - *User: The user struct with the populated ID field.
//   - error: If any error occurs during the process.
func CreateUser(user *User) (*User, error) {
	// fmt.Println(user.Details.DOB)
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	// setting initial entries for some fields
	user.Status = "Pending"
	user.OTPStatus = "Pending"
	user.EmailStatus = "Pending"
	user.RoleSlug = utils.GenerateSlug(user.RoleSlug)

	query := `INSERT INTO users (name, user_code, email, password, mobile_no, role_slug, otp_status, email_status, status, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`

	var userID int64
	err := database.DB.QueryRow(query, user.Name, user.UserCode, user.Email, user.Password, user.MobileNo, user.RoleSlug, user.OTPStatus, user.EmailStatus, user.Status, user.CreatedAt, user.UpdatedAt).Scan(&userID)

	err = CreateDetails(userID, *user.Details)
	if err != nil {
		fmt.Printf("Error during database query: %v\n", err)
		return nil, fmt.Errorf("unable to create user: %v", err)
	}
	user.ID = userID
	return user, nil
}

// GenrateOTP generates and stores a 6-digit OTP for the specified user.
// This function performs the following:
// 1. Generates a random 6-digit OTP.
// 2. Sets the OTP to expire in 1 minute.
// 3. Inserts the OTP into the otp_verification table.
//
// Params:
//   - userId (int64): The ID of the user for whom the OTP is generated.
//
// Returns:
//   - string: The generated OTP.
//   - error: If an error occurs during insertion.
func GenrateOTP(userId int64) (string, error) {
	rand.Seed(time.Now().UnixNano())
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))

	expireAt := time.Now().Add(5 * time.Minute)

	query := `INSERT INTO otp_verification(user_id, otp, expire_at) VALUES($1, $2, $3)`
	_, err := database.DB.Exec(query, userId, otp, expireAt)

	if err != nil {
		return "", fmt.Errorf("failed to store OTP : %v", err)
	}
	return otp, nil
}

// UpdateOTP updates the OTP for the specified user with a new one and resets the expiry.
// This function performs the following:
// 1. Generates a new random 6-digit OTP.
// 2. Sets the new OTP to expire in 1 minute.
// 3. Updates the existing record in the otp_verification table.
//
// Params:
//   - userId (int64): The ID of the user whose OTP is being updated.
//
// Returns:
//   - string: The new generated OTP.
//   - error: If an error occurs during the update.
func UpdateOTP(userId int64) (string, error) {
	rand.Seed(time.Now().UnixNano())
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))

	expireAt := time.Now().Add(5 * time.Minute)
	// fmt.Println(userId)
	// fmt.Println("OTP SENT IS : ", otp)
	query := `UPDATE otp_verification SET otp=$1, expire_at=$2 WHERE user_id = $3`
	_, err := database.DB.Exec(query, otp, expireAt, userId)

	if err != nil {
		return "", fmt.Errorf("failed to store OTP : %v", err)
	}
	return otp, nil
}

// CreateDetails inserts an empty/default row in the user_details table for the specified user ID.
// This function is intended to initialize a placeholder entry for user details,
// which can later be updated with actual data.
//
// Params:
//   - userId (int64): The ID of the user for whom the details are being created.
//
// Returns:
//   - error: If an error occurs during insertion.
func CreateDetails(userId int64, details UserDetails) error {
	// var details UserDetails

	query := `INSERT INTO public.user_details (nationality, gender, skill_level, address, facebook_link, insta_link, height, weight, state, city, aadhar_no, dob, profile_image_path, user_id, organization_name)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15);`

	_, err := database.DB.Exec(query, details.Nationality, details.Gender, details.Skill_level, details.Address, details.Facebook_link, details.Insta_link, details.Height,
		details.Weight, details.State, details.City, details.Aadhar_no, details.DOB, details.Profile_Image_Path, userId, details.OrganizationName)
	// fmt.Println(err)
	if err != nil {
		return err
	}
	return nil
}

// GetUserByMobile fetches a user by mobile number.
// This function performs the following:
// 1. Executes a SELECT query on the users table using the provided mobile number.
// 2. Maps the result to a User struct.
//
// Params:
//   - mobileNo (string): The mobile number to search for.
//
// Returns:
//   - *User: The user object if found.
//   - error: If user not found or a database error occurs.
func GetUserByMobile(mobileNo string) (*User, error) {
	var user User

	err := database.DB.QueryRow("SELECT id, name, email, password, mobile_no, role_slug, otp_status, email_status, status, created_at, updated_at FROM users WHERE mobile_no = $1 AND status != 'Delete'", mobileNo).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.MobileNo,
		&user.RoleSlug,
		&user.OTPStatus,
		&user.EmailStatus,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		// fmt.Println("Error scanning user:", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %v", err)
	}

	return &user, nil
}

// GetUserByEmail fetches a user by email.
// This function performs the following:
// 1. Executes a SELECT * query on the users table using the provided email.
// 2. Maps all fields to a User struct.
//
// Params:
//   - email (string): The email to search for.
//
// Returns:
//   - *User: The user object if found.
//   - error: If user not found or a database error occurs.
func GetUserByEmail(email string) (*User, error) {
	var user User
	email = strings.ToLower(email)
	err := database.DB.QueryRow("SELECT * FROM users WHERE LOWER(email) = $1 AND status != 'Delete'", email).Scan(
		&user.ID,
		&user.UserCode,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.MobileNo,
		&user.RoleSlug,
		&user.OTPStatus,
		&user.EmailStatus,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		// fmt.Println("Error scanning user:", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %v", err)
	}

	return &user, nil
}

// GetUserByID fetches a user by their unique user ID.
// This function performs the following:
// 1. Executes a SELECT query using the user ID.
// 2. Maps the result into a User struct.
//
// Params:
//   - userID (int): The ID of the user to retrieve.
//
// Returns:
//   - *User: The user object if found.
//   - error: If user not found or a database error occurs.
func GetUserByID(userID int) (*User, error) {
	var user User

	err := database.DB.QueryRow("SELECT id, name, email, password, mobile_no, role_slug, otp_status, email_status, status, created_at, updated_at FROM users WHERE id = $1", userID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.MobileNo,
		&user.RoleSlug,
		&user.OTPStatus,
		&user.EmailStatus,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		// fmt.Println("Error scanning user:", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %v", err)
	}

	return &user, nil
}

// UpdateUser updates an existing user record in the database.
// This function performs the following:
// 1. Updates fields such as name, email, password, role, status, and timestamps.
// 2. Executes an UPDATE statement using the provided User struct.
//
// Params:
//   - user (*User): Pointer to the user object containing updated data.
//
// Returns:
//   - error: If the update fails due to a query or connection issue.
func UpdateUser(user *User, tx *sql.Tx) error {
	// fmt.Println("User got for update is : ", user)
	// fmt.Println("OTP status is : ", user.OTPStatus)

	query := `
		UPDATE users SET name = $1, email = $2, password = $3, mobile_no = $4, role_slug = $5, otp_status=$6, email_status=$7, status = $8, updated_at = $9 WHERE id = $10`

	_, err := tx.Exec(query, user.Name, user.Email, user.Password, user.MobileNo, user.RoleSlug, user.OTPStatus, user.EmailStatus, user.Status, time.Now(), user.ID)

	if err != nil {
		fmt.Printf("Error updating user: %v\n", err)
		return fmt.Errorf("unable to update user: %v", err)
	}

	return nil
}

func UpdateSignupDetails(details UserDetails, userId int64, tx *sql.Tx) error {
	query := `
        UPDATE user_details
        SET organization_name=$1, dob=$2, gender=$3
        WHERE user_id=$4
    `

	// Handle DOB properly
	var dob interface{}
	if details.DOB == nil || *details.DOB == "" {
		dob = nil // store NULL in DB
	} else {
		dob = *details.DOB // must be YYYY-MM-DD string
	}

	var gender interface{}
	if details.Gender == nil || *details.Gender == "" {
		gender = nil
	} else {
		gender = *details.Gender
	}

	_, err := tx.Exec(query,
		*details.OrganizationName,
		dob,
		gender,
		userId,
	)
	if err != nil {
		return fmt.Errorf("unable to update user_details: %v", err)
	}
	return nil
}

// OTPVerification struct
type OTPVerification struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	OTP      string    `json:"otp"`
	ExpireAt time.Time `json:"expire_at"`
}

// GetOTPByUserID fetches the latest OTP for the user.
// This function performs the following:
// 1. Executes a query to retrieve the most recent OTP associated with a given user ID.
// 2. If no OTP is found, returns nil without an error.
//
// Params:
//   - userID (int): The ID of the user.
//
// Returns:
//   - *OTPVerification: OTP details if found, or nil if not.
//   - error: If there is a database error.
func GetOTPByUserID(userID int) (*OTPVerification, error) {
	query := `SELECT id, user_id, otp, expire_at FROM otp_verification WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`
	var otpDetails OTPVerification

	err := database.DB.QueryRow(query, userID).Scan(&otpDetails.ID, &otpDetails.UserID, &otpDetails.OTP, &otpDetails.ExpireAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No OTP found
		}
		return nil, err
	}

	return &otpDetails, nil
}

// UpdateUserEmail updates a user's email address.
// This function performs the following:
// 1. Executes an UPDATE query to change the user's email.
// 2. Also updates the updated_at timestamp.
//
// Params:
//   - userID (int): The ID of the user.
//   - newEmail (string): The new email to set.
//
// Returns:
//   - error: If the update query fails.
func UpdateUserEmail(userID int, newEmail string) error {
	query := `UPDATE users SET email = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(query, newEmail, time.Now(), userID)

	if err != nil {
		fmt.Printf("Error updating user email: %v\n", err)
		return fmt.Errorf("unable to update user email: %v", err)
	}

	return nil
}

// UpdatePassword updates a user's password.
// This function performs the following:
// 1. Updates the password field for the user identified by email in the specified table.
// 2. Updates the updated_at field as well.
//
// Params:
//   - table (string): The name of the table containing the user record.
//   - email (string): The email of the user.
//   - newPassword (string): The new hashed password.
//
// Returns:
//   - error: If the update operation fails.
func UpdatePassword(table string, email string, newPassword string) error {
	query := `UPDATE ` + table + ` SET password = $1, updated_at = $2 WHERE email = $3`
	_, err := database.DB.Exec(query, newPassword, time.Now(), email)

	if err != nil {
		fmt.Printf("Error updating password: %v\n", err)
		return fmt.Errorf("unable to update password: %v", err)
	}

	return nil
}

// UpdateUserStatus updates the verification status of a user.
// This function performs the following:
// 1. Marks the user as verified (is_verified = TRUE).
// 2. Updates their account status and updated_at timestamp.
//
// Params:
//   - userID (int): The ID of the user.
//   - status (string): The new status value (e.g., "active").
//
// Returns:
//   - error: If the update fails.
func UpdateUserStatus(userID int, status string) error {
	query := `UPDATE users SET status = $1, is_verified = TRUE, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(query, status, time.Now(), userID)

	if err != nil {
		fmt.Println("Error updating user status:", err)
		return fmt.Errorf("unable to update user status: %v", err)
	}

	return nil
}

// GetUserDetails fetches detailed profile information for a user.
// This function performs the following:
// 1. Retrieves user_details based on the given userID.
// 2. If no data is found, returns an empty UserDetails struct.
// 3. Fetches associated interested games with names and slugs from the games table.
// 4. Assigns the interested games to the user profile object.
//
// Params:
//   - userID (int64): The ID of the user.
//
// Returns:
//   - *UserDetails: A populated UserDetails struct including basic info and interested games.
//   - error: If any error occurs during data retrieval.
func GetUserDetails(userID int64) (*UserDetails, error) {
	var user_details UserDetails

	err := database.DB.QueryRow("SELECT id, user_id, nationality, gender, skill_level, address, facebook_link, insta_link, height, weight, state, city, aadhar_no, dob, profile_image_path, status, organization_name, created_at, updated_at FROM user_details WHERE user_id = $1", userID).Scan(
		&user_details.ID,
		&user_details.User_id,
		&user_details.Nationality,
		&user_details.Gender,
		&user_details.Skill_level,
		&user_details.Address,
		&user_details.Facebook_link,
		&user_details.Insta_link,
		&user_details.Height,
		&user_details.Weight,
		&user_details.State,
		&user_details.City,
		&user_details.Aadhar_no,
		&user_details.DOB,
		&user_details.Profile_Image_Path,
		&user_details.Status,
		&user_details.OrganizationName,
		&user_details.CreatedAt,
		&user_details.UpdatedAt,
	)
	// fmt.Println("in get user_details, id:", user_details.ID, ", user_id:", user_details.User_id)
	if err != nil {
		// fmt.Println("Error scanning user details:", err)
		if err == sql.ErrNoRows {
			return &user_details, nil
		}
		return nil, fmt.Errorf("failed to fetch user_details: %v", err)
	}

	// Fetch interested games with game_name
	rows, err := database.DB.Query(`
		SELECT uhg.game_id, g.game_name, g.slug
		FROM user_has_interested_games uhg
		JOIN games g ON uhg.game_id = g.id
		WHERE uhg.user_id = $1`, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch interested games: %v", err)
	}
	defer rows.Close()

	var interestedGames []InterestedGames
	for rows.Next() {
		var game InterestedGames
		if err := rows.Scan(&game.GameId, &game.GameName, &game.Slug); err != nil {
			return nil, fmt.Errorf("error scanning game details: %v", err)
		}
		// Convert integer IDs to encoded strings

		interestedGames = append(interestedGames, game)
	}

	// Assign interested games to user details
	user_details.Games = interestedGames

	rows2, err := database.DB.Query(`
		SELECT uhpr.profile_role_id, pr.role, pr.slug
		FROM user_has_profile_roles uhpr
		JOIN profile_roles pr ON uhpr.profile_role_id = pr.id
		WHERE uhpr.user_id = $1`, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch interested games: %v", err)
	}
	defer rows2.Close()

	var profileRoles []ProfileRoles
	for rows2.Next() {
		var profileRole ProfileRoles
		if err := rows2.Scan(&profileRole.Id, &profileRole.Role, &profileRole.Slug); err != nil {
			return nil, fmt.Errorf("error scanning profile roles: %v", err)
		}
		// Convert integer IDs to encoded strings

		profileRoles = append(profileRoles, profileRole)
	}

	// Assign interested games to user details
	user_details.ProfileRoles = profileRoles
	// fmt.Println("details : ", user_details)
	return &user_details, nil
}

func GetAllProfileRoles() ([]ProfileRoles, error) {
	var profileRoles []ProfileRoles
	query := `SELECT id, role, slug FROM profile_roles WHERE status = 'Active';`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch profile roles : %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var profileRole ProfileRoles
		if err := rows.Scan(&profileRole.Id, &profileRole.Role, &profileRole.Slug); err != nil {
			return nil, fmt.Errorf("error scanning profile role : %v", err)
		}
		profileRoles = append(profileRoles, profileRole)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over age groups  : %v", err)
	}
	return profileRoles, nil
}

func IsUserModerator(userID int) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM organization_has_score_moderator
		WHERE moderator_id = $1 AND status = 'Active'
	`
	err := database.DB.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetProfileImageById retrieves the profile image path for a user.
// This function performs the following:
// 1. Executes a query to fetch the profile image path associated with the user.
// 2. If no profile image is found, returns an empty string.
//
// Params:
//   - userID (int): The ID of the user whose profile image path is to be fetched.
//
// Returns:
//   - string: The profile image path if found, or an empty string if no image is available.
//   - error: If there is an issue during the database query.
func GetProfileImageById(userID int) (string, error) {
	var ImagePath any
	err := database.DB.QueryRow("SELECT profile_image_path FROM user_details WHERE user_id = $1", userID).Scan(
		&ImagePath,
	)
	if err != nil {
		//fmt.Println("Error scanning Profile picture:", err)
		if err == sql.ErrNoRows {
			return "", nil // No profile image found
		}
		return "", fmt.Errorf("failed to fetch profile_image: %v", err)
	}
	return fmt.Sprintf("%v", ImagePath), nil
}

// GetMaxUserCode retrieves the highest user code from the database.
// This function performs the following:
// 1. Executes a query to get the highest user code from the users table, sorted in descending order.
// 2. Extracts the last 4 digits of the user code and converts them to an integer.
//
// Returns:
//   - int: The last numeric part of the highest user code.
//   - error: If any error occurs during the query or conversion process.
func GetMaxUserCode() (int, error) {
	var maxCode int
	query := "SELECT user_code FROM users ORDER BY user_code DESC LIMIT 1"
	err := database.DB.QueryRow(query).Scan(&maxCode)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No user code found
		}
		return 0, err
	}

	// codeNum, err := strconv.Atoi(maxCode[len(maxCode)-4:]) // Extract last 4 digits
	// if err != nil {
	// 	return 0, err
	// }

	return maxCode, nil
}

// GetUserByUserCode retrieves a user by their user code and optionally returns their age and additional details.
//
// Behavior:
//  1. Queries the `users` table to fetch the user record where user_code matches and status is 'Active'.
//  2. If getDetails or getAge is true, it fetches additional data from the `user_details` table.
//  3. If getAge is true and a valid DOB exists, it calculates the user's age.
//
// Parameters:
//   - userCode (string): The unique user code to search for.
//   - getAge (bool): If true, attempts to calculate the user's age from DOB.
//   - getDetails (bool): If true, fetches extended user details from the `user_details` table.
//
// Returns:
//   - *User: Pointer to the populated User struct (with or without details).
//   - int: Calculated age of the user (0 if not calculated).
//   - error: An error if something goes wrong during the database query or age calculation.
func GetUserByUserCode(userCode string, getAge bool, getDetails bool) (*User, int, error) {
	query := "SELECT * FROM users WHERE user_code=$1 AND status='Active'"
	var reqUser User
	// var dob sql.NullTime
	var age int
	err := database.DB.QueryRow(query, userCode).Scan(
		&reqUser.ID,
		&reqUser.UserCode,
		&reqUser.Name,
		&reqUser.Email,
		&reqUser.Password,
		&reqUser.MobileNo,
		&reqUser.RoleSlug,
		&reqUser.OTPStatus,
		&reqUser.EmailStatus,
		&reqUser.Status,
		&reqUser.CreatedAt,
		&reqUser.UpdatedAt,
	)
	if err != nil {
		return &reqUser, age, err // Error fetching user details
	}

	if !getDetails {
		return &reqUser, 0, nil
	}
	reqUser.Details, err = GetUserDetails(reqUser.ID)
	if err != nil {
		return &reqUser, age, fmt.Errorf("error fetching details:%v", err)
	}

	// If DOB is valid, calculate age
	if reqUser.Details.DOB != nil && getAge {
		// datestring := fmt.Sprintf("%d-%02d-%02d", dob.Time.Year(), int(dob.Time.Month()), dob.Time.Day())
		datestring, err := time.Parse(time.RFC3339, *reqUser.Details.DOB)
		if err != nil {
			return &reqUser, age, err
		}
		age, err = utils.CalculateAge(fmt.Sprintf("%d-%02d-%02d", datestring.Year(), int(datestring.Month()), datestring.Day()))
		// fmt.Println("age:", age, "error:", err)
		if err != nil {
			return &reqUser, age, err
		}
		if age == 0 {
			return &reqUser, age, fmt.Errorf("age calculation error") // In case age calculation returns 0
		}
	}
	return &reqUser, age, nil
}
func GetTshirtSizeByUserId(UserId int64, teamId int64) (string, error) {
	var TshirtSize sql.NullString
	query := `
		SELECT tshirt_size FROM event_has_users
		WHERE event_has_team_id=$1 AND user_id=$2
	`
	err := database.DB.QueryRow(query, teamId, UserId).Scan(&TshirtSize)
	if err != nil {
		return "", fmt.Errorf("databse error while retreiving t-shirt sizes-->%v", err)
	}
	return TshirtSize.String, nil
}

func ValidateAgeAndGender(EHGameTypeId int64, userAge int, gender string) (bool, bool, error) {
	var minAge, maxAge sql.NullInt32
	var gameTypeName string
	var AgeValid bool
	var GenderValid bool
	query := `
		SELECT ag.maxage, ag.minage, gt.name
		FROM event_has_game_types ehgt
		  JOIN age_group ag ON ehgt.age_group_id= ag.id
		  JOIN games_types gt ON ehgt.game_type_id= gt.id
		WHERE ehgt.id=$1 AND ag.status='Active'
	`
	err := database.DB.QueryRow(query, EHGameTypeId).Scan(&maxAge, &minAge, &gameTypeName)
	if err != nil {
		return false, false, fmt.Errorf("database error --->%v", err)
	}

	if userAge != 0 {
		if (userAge >= int(minAge.Int32) || !minAge.Valid) && (((minAge.Valid && userAge <= int(maxAge.Int32)) || userAge < int(maxAge.Int32)) || !maxAge.Valid) {
			AgeValid = true
		} else {
			AgeValid = false
		}
	} else {
		return false, false, fmt.Errorf("user age cannot be 0")
	}

	gameTypeName = strings.ToLower(gameTypeName)

	if strings.Contains(gameTypeName, "female") || strings.Contains(gameTypeName, "woman") || strings.Contains(gameTypeName, "women") {
		GenderValid = strings.ToLower(gender) == "female"
	} else if strings.Contains(gameTypeName, "male") || strings.Contains(gameTypeName, "man") || strings.Contains(gameTypeName, "men") {
		GenderValid = strings.ToLower(gender) == "male"
	} else {
		GenderValid = true
	}

	return AgeValid, GenderValid, nil
}

func IsOrganizationNameExists(name string, id int) (bool, error) {
	var count int
	query := `
		SELECT COUNT(ud.organization_name)
		FROM user_details ud
		JOIN users u ON u.id = ud.user_id
		WHERE LOWER(TRIM(ud.organization_name)) LIKE LOWER(TRIM($1))
		AND u.status = 'Active'
		AND u.id != $2;
 	`

	// Use QueryRow with Scan
	err := database.DB.QueryRow(query, name, id).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func SyncUserDetailStatusByUserId(userId int64) error {
	query := `
		UPDATE user_details
		SET status= u.status
		FROM users u
		where user_details.user_id= $1
	`
	_, err := database.DB.Exec(query, userId)
	if err != nil {
		return fmt.Errorf("failed to update user_details table: %v", err)
	}
	return nil
}

func GetFullUserByID(userID int) (*User, error) {
	var user User
	var details UserDetails
	var stateName, cityName sql.NullString

	query := `
		SELECT
			u.id, u.user_code, u.name, u.email, u.mobile_no, u.role_slug, u.status,
			u.otp_status, u.email_status, u.created_at, u.updated_at,
			ud.organization_name, ud.nationality, ud.skill_level, ud.address,
			ud.facebook_link, ud.insta_link, ud.height, ud.weight, ud.current_team,
			ud.gender, ud.state, ud.city, ud.aadhar_no, ud.dob, ud.profile_image_path,
			ud.status, ud.created_at, ud.updated_at,
			s.name AS state_name, c.city AS city_name
		FROM users u
		LEFT JOIN user_details ud ON ud.user_id = u.id
		LEFT JOIN states s ON s.id = ud.state
		LEFT JOIN cities c ON c.id = ud.city
		WHERE u.id = $1 AND u.status != 'Delete'
	`

	err := database.DB.QueryRow(query, userID).Scan(
		&user.ID, &user.UserCode, &user.Name, &user.Email, &user.MobileNo,
		&user.RoleSlug, &user.Status, &user.OTPStatus, &user.EmailStatus,
		&user.CreatedAt, &user.UpdatedAt,

		&details.OrganizationName, &details.Nationality, &details.Skill_level,
		&details.Address, &details.Facebook_link, &details.Insta_link,
		&details.Height, &details.Weight, &details.Current_team,
		&details.Gender, &details.State, &details.City, &details.Aadhar_no,
		&details.DOB, &details.Profile_Image_Path, &details.Status,
		&details.CreatedAt, &details.UpdatedAt,

		&stateName, &cityName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %v", err)
	}

	details.User_id = int(user.ID)
	if stateName.Valid {
		details.StateName = stateName.String
	}
	if cityName.Valid {
		details.CityName = cityName.String
	}

	user.Details = &details
	return &user, nil
}

func GetUserCountAttachedToMobileNumber(mobileNo string, userId int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM users
		WHERE mobile_no= $1 AND status='Active' AND id!=$2
	`
	count := 0
	err := database.DB.QueryRow(query, mobileNo, userId).Scan(&count)
	if err != nil {
		return count, err
	}
	return count, nil
}
