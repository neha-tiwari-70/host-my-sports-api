package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"
	"time"
)

type Admin struct {
	ID        int       `json:"id"`
	Name      string    `json:"name,omitempty" validate:"omitempty,min=2"`
	Email     string    `json:"email,omitempty" validate:"omitempty,email"`
	Password  string    `json:"-" validate:"omitempty,min=8"`
	MobileNo  *string   `json:"mobile_no,omitempty" validate:"omitempty,len=10,numeric"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type LoginAdmin struct {
	Type     string `json:"type"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Password string `json:"password,omitempty" validate:"omitempty,min=8"`
}

type UpdatedAdmin struct {
	Type     string `json:"type"`
	ID       int    `json:"id"`
	Name     string `json:"name,omitempty" validate:"omitempty,min=2"`
	MobileNo string `json:"mobile_no,omitempty" validate:"omitempty,len=10,numeric"`
}

// CreateAdmin creates a new admin with default values for CreatedAt and UpdatedAt if not provided.
// It inserts the admin record into the database and returns the created admin with its assigned ID.
//
// Params:
//   - admin (*Admin): The admin struct with the necessary fields to create a new admin.
//
// Returns:
//   - *Admin: The newly created admin with the assigned ID.
//   - error: An error if the database insertion fails.
func CreateAdmin(admin *Admin) (*Admin, error) {
	// Set default values for CreatedAt and UpdatedAt if not provided
	if admin.CreatedAt.IsZero() {
		admin.CreatedAt = time.Now() // Use current time if CreatedAt is not set
	}
	if admin.UpdatedAt.IsZero() {
		admin.UpdatedAt = time.Now() // Use current time if UpdatedAt is not set
	}

	// SQL query to insert the new admin into the database
	query := `INSERT INTO admin (name, email, password, mobile_no, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	// Prepare statement to execute the SQL query and retrieve the admin ID
	var adminID int
	err := database.DB.QueryRow(query, admin.Name, admin.Email, admin.Password, admin.MobileNo, admin.CreatedAt, admin.UpdatedAt).Scan(&adminID)

	if err != nil {
		// Error handling: Print error and return a formatted error message
		fmt.Printf("Error during database query: %v\n", err)
		return nil, fmt.Errorf("unable to create admin: %v", err)
	}

	// Set the returned admin ID to the admin struct
	admin.ID = adminID

	// Return the created admin with the assigned ID
	return admin, nil
}

// GetAdminByID retrieves an admin by their unique ID from the database.
// It returns the admin details if found, or an error if something goes wrong.
//
// Params:
//   - adminID (int): The unique ID of the admin to retrieve.
//
// Returns:
//   - *Admin: The admin object if found, otherwise nil.
//   - error: An error if the admin is not found or there's an issue with the query.
func GetAdminByID(adminID int) (*Admin, error) {
	var admin Admin

	// Query to fetch all columns for the admin with the specified adminID
	err := database.DB.QueryRow("SELECT * FROM admin WHERE id = $1", adminID).Scan(
		&admin.ID,        // Auto-incremented ID
		&admin.Name,      // Admin name
		&admin.Email,     // Admin email
		&admin.Password,  // Admin password
		&admin.MobileNo,  // Admin mobile number
		&admin.CreatedAt, // Admin account creation timestamp
		&admin.UpdatedAt, // Admin account last updated timestamp
	)

	// Debugging - Print the error if it exists
	if err != nil {
		// If no rows are found, return a specific error
		fmt.Println("Error scanning admin:", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("admin not found")
		}
		// Return a generic error if the fetch fails for any other reason
		return nil, fmt.Errorf("failed to fetch admin: %v", err)
	}

	// Return the admin struct with populated data
	return &admin, nil
}

// GetAdminByEmail retrieves an admin by their email address from the database.
// It returns the admin details if found, or an error if something goes wrong.
//
// Params:
//   - email (string): The email address of the admin to retrieve.
//
// Returns:
//   - *Admin: The admin object if found, otherwise nil.
//   - error: An error if the admin is not found or there's an issue with the query.
func GetAdminByEmail(email string) (*Admin, error) {
	var admin Admin

	// Query to fetch all columns for the admin with the specified email
	err := database.DB.QueryRow("SELECT * FROM admin WHERE email = $1", email).Scan(
		&admin.ID,        // Auto-incremented ID
		&admin.Name,      // Admin name
		&admin.Email,     // Admin email
		&admin.Password,  // Admin password
		&admin.MobileNo,  // Admin mobile number
		&admin.CreatedAt, // Admin account creation timestamp
		&admin.UpdatedAt, // Admin account last updated timestamp
	)

	// Debugging - Print the error if it exists
	if err != nil {
		// If no rows are found, return a specific error
		fmt.Println("Error scanning admin:", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("admin not found")
		}
		// Return a generic error if the fetch fails for any other reason
		return nil, fmt.Errorf("failed to fetch admin: %v", err)
	}

	// Return the admin struct with populated data
	return &admin, nil
}

// UpdateAdminPassword updates the password of an admin based on their email.
// It returns an error if the update fails.
//
// Params:
//   - email (string): The email of the admin whose password is to be updated.
//   - password (string): The new password to be set for the admin.
//
// Returns:
//   - error: An error if the update operation fails.
func UpdateAdminPassword(email string, password string) error {

	// SQL query to update the admin's password based on their email
	query := `UPDATE admin SET password= $1 WHERE email=$2`

	// Execute the query with the provided password and email
	_, err := database.DB.Exec(query, password, email)

	// Return any error that occurs during execution
	if err != nil {
		return err
	}

	// Return nil if the password update was successful
	return nil
}
