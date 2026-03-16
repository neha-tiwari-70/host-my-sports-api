package controllers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// profileImage struct represents the structure for handling the profile image data.
// It contains the encrypted user ID, the file being uploaded, the name of the image, and the image path.
type profileImage struct {
	EncID string                `json:"id" binding:"required"` // Encrypted user ID for the profile image
	File  *multipart.FileHeader `json:"file"`                  // Uploaded file (profile image)
	Name  string                `json:"name"`                  // Name of the image
	Path  string                `json:"path"`                  // Path where the image will be stored
}

const FolderPath = "public/uploads" // Folder path where the profile images will be stored

// EmptyProfileImage removes the profile image associated with the user.
// This function performs the following:
// 1. Creates a folder in the backend project if it doesn't exist.
// 2. Binds the incoming request data (profile image) to the `profileImage` struct.
// 3. Updates the database to remove the profile image.
// 4. Handles errors and success responses accordingly.
//
// Params:
//   - c (*gin.Context): The Gin context to handle the HTTP request and response.
//
// Returns:
//   - error: If any error occurs during the process (e.g., file deletion or database update).
func EmptyProfileImage(c *gin.Context) {
	// Step 1: Define the backend project folder path
	FolderIndicator, err := utils.CreateFolder(FolderPath)
	if err != nil {
		utils.HandleError(c, fmt.Sprintf("%v", err))
	}

	// Step 2: Bind the incoming payload to the `profileImage` struct
	var Image profileImage
	if err := c.ShouldBindJSON(&Image); err != nil {
		// fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 3: Handle the database updates to remove the profile image
	_, err = DatabaseUpdate(&Image)
	if err != nil {
		utils.HandleError(c, "Database error while updating profile image", err)
	}
	utils.HandleSuccess(c, "Profile image removed successfully."+FolderIndicator, Image.Path)
}

// UpdateProfileImage updates the user's profile image.
// This function performs the following:
// 1. Creates the required folder for storing the image if it doesn't exist.
// 2. Binds the incoming file data to the `profileImage` struct and updates the database.
// 3. Saves the uploaded file to the server.
// 4. Handles errors and success responses accordingly.
//
// Params:
//   - c (*gin.Context): The Gin context to handle the HTTP request and response.
//
// Returns:
//   - error: If any error occurs during the process (e.g., file upload or database update).
func UpdateProfileImage(c *gin.Context) {
	// Step 1: Define the frontend project folder path
	FolderIndicator, err := utils.CreateFolder(FolderPath)
	if err != nil {
		utils.HandleError(c, fmt.Sprintf("%v", err))
	}

	// Step 2: Bind the incoming file data to the `profileImage` struct
	var Image profileImage
	Image.EncID = c.DefaultPostForm("id", "") // Get the encrypted user ID
	Image.File, _ = c.FormFile("file")        // Get the uploaded file from the request
	Image, err = DatabaseUpdate(&Image)       // Update the database with the image details
	if err != nil {
		utils.HandleError(c, "Database error while updating profile image", err)
		return
	}

	// Step 3: Save the uploaded file to the specified path
	err = c.SaveUploadedFile(Image.File, Image.Path)
	if err != nil {
		utils.HandleError(c, "Could Not Upload The File", err)
		return
	}

	// Step 4: Send a success response
	utils.HandleSuccess(c, "Profile picture set successfully"+FolderIndicator, Image.Path)
}

// DatabaseUpdate updates the user's profile image details in the database.
// This function performs the following:
// 1. Decrypts the encrypted user ID to identify the user.
// 2. Fetches the user details and checks if a previous profile image exists.
// 3. Deletes the old profile image if it exists (except for the default image).
// 4. Saves the new image file path to the database.
// 5. Returns the updated image details or any error encountered during the process.
//
// Params:
//   - Image (*profileImage): The profile image data to be updated.
//
// Returns:
//   - profileImage: The updated profile image data (including file path).
//   - error: If any error occurs during the process (e.g., decryption, database update).
func DatabaseUpdate(Image *profileImage) (profileImage, error) {
	// Step 1: Decrypt the user ID
	DecryptedId, err := crypto.NDecrypt(Image.EncID)
	if err != nil {
		return *Image, fmt.Errorf("failed to decrypt id")
	}

	// Step 2: Get user details from the database using the decrypted user ID
	user, err := models.GetUserByID(int(DecryptedId))
	if err != nil {
		return *Image, err
	}

	// Step 3: Delete old image if it exists
	oldPath, err := models.GetProfileImageById(int(DecryptedId))
	if err != nil {
		return *Image, fmt.Errorf("failed fetch to Former Image %v", err)
	}
	if oldPath != "" {
		Path := filepath.Join(FolderPath, filepath.Base(oldPath))
		if filepath.Base(oldPath) != "static.png" { // Do not delete the default image
			_ = os.Remove(Path)
			// fmt.Println("Image deleted from path", Path)
		}
	}

	// Step 4: Check if the image file is present and create the image name and path
	if Image.File != nil {
		timestamp := time.Now().Format("20060102150405")
		Image.Name = strings.ReplaceAll((user.Name + timestamp + ".png"), " ", "")
		Image.Path = filepath.Join(FolderPath, Image.Name)
	}

	// Step 5: Update the database with the new image path
	query := `
	    UPDATE public.user_details SET
	        profile_image_path = $1
	    WHERE user_id = $2;
	`
	_, err = database.DB.Exec(query, Image.Path, DecryptedId)
	if err != nil {
		return *Image, err
	}

	// Return the updated image details
	return *Image, nil
}
