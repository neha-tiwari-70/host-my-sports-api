// Package models contains database models and related functions for the sports events platform.
package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"
	"time"
)

// ForgotUser represents a user requesting a password reset.
// This table stores temporary data for verifying the user's identity and handling expiration.
type ForgotUser struct {
	ID             int       `json:"id"`                                        // Unique identifier (auto-incremented)
	IsAdmin        bool      `json:"is_admin"`                                  // Indicates if the user is an admin
	Email          string    `json:"email,omitempty" validate:"required,email"` // User's email address
	Code           string    `json:"-"`                                         // Verification code (not exposed in JSON)
	Status         string    `json:"status,omitempty"`                          // Status of the reset process (e.g., pending, used)
	CreatedAt      time.Time `json:"created_at"`                                // Timestamp when the record was created
	UpdatedAt      time.Time `json:"updated_at"`                                // Timestamp of the last update
	ExpireAt       time.Time `json:"expire_at"`                                 // Expiration time for the code (typically 15 minutes from generation)
	RecaptchaToken string    `json:"recaptcha_token"`
}

// CreateForgotUser inserts a new ForgotUser record into the database.
// It also sets CreatedAt, UpdatedAt, and ExpireAt timestamps.
func CreateForgotUser(user *ForgotUser) (*ForgotUser, error) {
	// Set default values for CreatedAt and UpdatedAt if not provided
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now() // Use current time if not set
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now() // Use current time if not set
	}

	// Expire the reset link/code 15 minutes after it's created
	user.ExpireAt = user.UpdatedAt.Add(15 * time.Minute)

	// Insert user into the database and return the newly inserted ID
	query := `INSERT INTO forgot_users (is_admin,email, code, status, created_at, updated_at, expire_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	var userID int
	err := database.DB.QueryRow(
		query,
		user.IsAdmin,
		user.Email,
		user.Code,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
		user.ExpireAt,
	).Scan(&userID)

	if err != nil {
		fmt.Printf("Error during database query: %v\n", err)
		return nil, fmt.Errorf("unable to create user: %v", err)
	}

	// Set the returned user ID to the user struct
	user.ID = userID

	// Return the created user with the assigned ID
	return user, nil
}

// GetForgotUserByEmail fetches a ForgotUser record using the email address.
// This is typically used during password reset flows to verify if a reset was requested.
func GetForgotUserByEmail(email string) (*ForgotUser, error) {
	var user ForgotUser

	// Query to fetch all columns for the user with the given email
	err := database.DB.QueryRow("SELECT * FROM forgot_users WHERE email = $1", email).Scan(
		&user.ID,        // Auto-incremented ID
		&user.IsAdmin,   // Role flag
		&user.Email,     // User email
		&user.Code,      // Reset verification code
		&user.Status,    // Reset status
		&user.CreatedAt, // Timestamp when record was created
		&user.UpdatedAt, // Timestamp when record was last updated
		&user.ExpireAt,  // When the reset code expires
	)

	// Debugging - Print the error if it exists
	if err != nil {
		fmt.Println("Error scanning user:", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %v", err)
	}

	return &user, nil
}
