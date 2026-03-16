package controllers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const EventFolderPath = "public/event"

func UploadMultipleEventFiles(c *gin.Context) {
	decryptedID := DecryptParamId(c, "id", true)
	if decryptedID == 0 {
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		utils.HandleError(c, "Failed to parse form", err)
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		utils.HandleInvalidEntries(c, "No files provided")
		return
	}

	var savedPaths []string
	for _, file := range files {
		// Get original file extension
		ext := filepath.Ext(file.Filename)
		if ext == "" {
			ext = ".bin"
		}
		encryptedEvId := crypto.NEncrypt(decryptedID)
		filename := fmt.Sprintf("event_%v_%d%s", encryptedEvId, time.Now().UnixNano(), ext)
		filePath := filepath.Join(EventFolderPath, filename)

		// Open uploaded file
		src, err := file.Open()
		if err != nil {
			utils.HandleError(c, "Error opening uploaded file", err)
			return
		}
		defer src.Close()

		// Create destination file
		dst, err := os.Create(filePath)
		if err != nil {
			utils.HandleError(c, "Error creating file on server", err)
			return
		}
		defer dst.Close()

		// Copy contents
		_, err = io.Copy(dst, src)
		if err != nil {
			utils.HandleError(c, "Error writing file to disk", err)
			return
		}

		// Normalize and store path
		normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
		savedPaths = append(savedPaths, normalizedPath)

		// Insert record in DB
		query := `INSERT INTO event_has_image (event_id, image, image_original_name) VALUES ($1, $2, $3)`
		_, err = database.DB.Exec(query, decryptedID, normalizedPath, file.Filename)
		if err != nil {
			utils.HandleError(c, "Failed to insert file record", err)
			return
		}
	}

	// Normalize file paths for response
	for i := range savedPaths {
		savedPaths[i] = strings.ReplaceAll(savedPaths[i], "\\", "/")
	}

	utils.HandleSuccess(c, "Event files uploaded successfully", gin.H{
		"uploaded_files_count": len(savedPaths),
		"file_paths":           savedPaths,
	})
}

func UpdateEventImages(c *gin.Context) {
	decryptedID := DecryptParamId(c, "id", true)
	if decryptedID == 0 {
		return
	}
	// Step 1: Fetch existing image hashes from DB
	rows, err := database.DB.Query("SELECT image, image_original_name FROM event_has_image WHERE event_id = $1", decryptedID)
	if err != nil {
		utils.HandleError(c, "Failed to fetch existing images", err)
		return
	}
	defer rows.Close()

	existingHashes := make(map[string]bool)
	for rows.Next() {
		var imagePath string
		if err := rows.Scan(&imagePath); err == nil {
			if data, err := os.ReadFile(imagePath); err == nil {
				hash := utils.GetSHA256Hash(data)
				existingHashes[hash] = true
			}
		}
	}

	// Step 2: Handle new uploads (additive only)
	form, err := c.MultipartForm()
	if err != nil {
		utils.HandleError(c, "Failed to parse form", err)
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		utils.HandleInvalidEntries(c, "No files provided")
		return
	}

	var savedPaths []string
	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			utils.HandleError(c, "Failed to open uploaded file", err)
			return
		}
		defer src.Close()

		fileBytes, err := io.ReadAll(src)
		if err != nil {
			utils.HandleError(c, "Failed to read uploaded file", err)
			return
		}

		fileHash := utils.GetSHA256Hash(fileBytes)
		if existingHashes[fileHash] {
			continue // skip duplicates
		}

		ext := filepath.Ext(file.Filename)
		if ext == "" {
			ext = ".bin"
		}

		encryptedEvId := crypto.NEncrypt(decryptedID)
		filename := fmt.Sprintf("event_%s_%d%s", encryptedEvId, time.Now().UnixNano(), ext)
		filePath := filepath.Join(EventFolderPath, filename)

		if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
			utils.HandleError(c, "Failed to save file", err)
			return
		}

		normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
		savedPaths = append(savedPaths, normalizedPath)

		// _, err = database.DB.Exec(`INSERT INTO event_has_image (event_id, image) VALUES ($1, $2)`, decryptedID, normalizedPath)
		_, err = database.DB.Exec(`INSERT INTO event_has_image (event_id, image, image_original_name) VALUES ($1, $2, $3)`, decryptedID, normalizedPath, file.Filename)
		if err != nil {
			utils.HandleError(c, "Failed to insert into event_has_image", err)
			return
		}
	}

	utils.HandleSuccess(c, "Event images updated successfully", gin.H{
		"uploaded_images_count": len(savedPaths),
		"file_paths":            savedPaths,
	})
}

/*
	func DeleteEventImage(c *gin.Context) {
		encryptedID := c.Param("id")
		if encryptedID == "" {
			utils.HandleError(c, "event_id is required")
			return
		}

		decryptedID, err := crypto.NDecrypt(encryptedID)
		if err != nil {
			utils.HandleError(c, "Failed to decrypt event_id", err)
			return
		}

		imagePath := c.Query("image")
		if imagePath == "" {
			utils.HandleError(c, "File path is required")
			return
		}

		_, err = database.DB.Exec(
			"DELETE FROM event_has_image WHERE event_id = $1 AND image = $2",
			decryptedID, imagePath,
		)
		if err != nil {
			utils.HandleError(c, "Failed to delete file from DB", err)
			return
		}

		_ = utils.DeleteFile(imagePath)

		utils.HandleSuccess(c, "File deleted successfully", gin.H{
			"deleted_image": imagePath,
		})
	}
*/
func DeleteEventImage(c *gin.Context) {
	decryptedID := DecryptParamId(c, "id", true)
	if decryptedID == 0 {
		return
	}
	// fmt.Println("Decrypted Event ID:", decryptedID)

	imagePath := c.Query("image")
	if imagePath == "" {
		// fmt.Println("ERROR: File path is required")
		utils.HandleError(c, "File path is required")
		return
	}
	// fmt.Println("Original imagePath from query:", imagePath)

	// Normalize the path to handle different formats
	normalizedPath := strings.ReplaceAll(imagePath, "\\", "/")
	// fmt.Println("Normalized path:", normalizedPath)

	// Ensure the path starts with the correct directory
	// Handle different possible path formats
	if !strings.HasPrefix(normalizedPath, "public/") {
		if strings.HasPrefix(normalizedPath, "event/") {
			normalizedPath = "public/" + normalizedPath
		} else {
			normalizedPath = "public/event/" + normalizedPath
		}
	}
	// fmt.Println("Final normalizedPath for DB:", normalizedPath)

	// First, check if the file exists in the database
	var dbImagePath string
	err := database.DB.QueryRow(
		"SELECT image FROM event_has_image WHERE event_id = $1 AND image = $2",
		decryptedID, normalizedPath,
	).Scan(&dbImagePath)

	if err != nil {
		// fmt.Println("ERROR: File not found in database for path:", normalizedPath)
		// fmt.Println("Database error:", err)
		utils.HandleError(c, "File not found in database")
		return
	}
	// fmt.Println("Found in database with path:", dbImagePath)

	// Delete from database
	result, err := database.DB.Exec(
		"DELETE FROM event_has_image WHERE event_id = $1 AND image = $2",
		decryptedID, normalizedPath,
	)
	if err != nil {
		// fmt.Println("ERROR: Failed to delete file from DB:", err)
		utils.HandleError(c, "Failed to delete file from DB", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	// fmt.Println("Database rows affected:", rowsAffected)

	if rowsAffected == 0 {
		// fmt.Println("WARNING: No rows affected - file might have been already deleted")
	}

	// Delete physical file
	// fmt.Println("Attempting to delete physical file:", normalizedPath)

	// Check if file exists before trying to delete
	if _, err := os.Stat(normalizedPath); os.IsNotExist(err) {
		// fmt.Println("WARNING: Physical file does not exist:", normalizedPath)
		// Continue with success since DB record is deleted
	} else {
		if err := utils.DeleteFile(normalizedPath); err != nil {
			// fmt.Println("WARNING: Failed to delete physical file:", normalizedPath, "Error:", err)
			// Log the error but don't fail the request since DB record is already deleted
		}
	}

	utils.HandleSuccess(c, "File deleted successfully", gin.H{
		"deleted_image":    normalizedPath,
		"db_rows_affected": rowsAffected,
	})
}
