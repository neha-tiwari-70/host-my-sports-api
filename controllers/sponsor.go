package controllers

import (
	"database/sql"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const SponsorFolderPath = "public/event"

func UploadMultipleSponsorLogos(c *gin.Context) {
	decryptedID := DecryptParamId(c, "id", true)
	if decryptedID == 0 {
		return
	}

	var exists int
	err := database.DB.QueryRow("SELECT COUNT(1) FROM events WHERE id = $1", decryptedID).Scan(&exists)
	if err != nil || exists == 0 {
		utils.HandleError(c, "Invalid event_id: Event does not exist", err)
		return
	}

	sponsorCountStr := c.PostForm("length")

	sponsorCount, err := strconv.ParseInt(sponsorCountStr, 10, 64)
	if err != nil {
		utils.HandleError(c, "invalid sponsor length", err)
		return
	}
	type Sponsor struct {
		title string
		logo  *multipart.FileHeader
	}
	SponsorMap := map[string]Sponsor{}
	for i := 0; i < int(sponsorCount); i++ {
		key := fmt.Sprintf("Sponsor%v", i)
		titleKey := fmt.Sprintf("sponsor_titles_%v", i)
		logoKey := fmt.Sprintf("logo_%v", i)

		//extract title
		title := c.PostForm(titleKey)

		//extract logo
		logo, err := c.FormFile(logoKey)

		if err != nil && err.Error() != "http: no such file" {
			utils.HandleError(c, "error extracting logo for "+key, err)
			return
		}

		SponsorMap[key] = Sponsor{
			title: title,
			logo:  logo,
		}

	}

	var insertedSponsors []gin.H
	for i := range SponsorMap {
		// timestamp := time.Now().Format("20060102150405")
		filename := ""
		filePath := ""

		if SponsorMap[i].logo != nil {
			filename = fmt.Sprintf("%v_%d_%d.png", i, decryptedID, time.Now().UnixNano())
			filePath = filepath.Join(SponsorFolderPath, filename)
			err := c.SaveUploadedFile(SponsorMap[i].logo, filePath)
			if err != nil {
				utils.HandleError(c, "Failed to save file", err)
				return
			}
		}

		var sponsorID int
		query := `INSERT INTO event_has_sponsors (event_id, sponsor_title, sponsor_logo) VALUES ($1, $2, $3) RETURNING id`
		err = database.DB.QueryRow(query, decryptedID, SponsorMap[i].title, filePath).Scan(&sponsorID)
		if err != nil {
			utils.HandleError(c, "Failed to insert into sponsor table", err)
			return
		}

		insertedSponsors = append(insertedSponsors, gin.H{
			"sponsor_id":    sponsorID,
			"sponsor_title": SponsorMap[i],
			"sponsor_logo":  filePath,
		})
	}

	utils.HandleSuccess(c, "Sponsor logos uploaded successfully", gin.H{
		"upload_logos_count": len(insertedSponsors),
		"sponsors":           insertedSponsors,
	})
}

func UpdateMultipleSponsors(c *gin.Context) {
	decryptedID := DecryptParamId(c, "id", true)
	if decryptedID == 0 {
		return
	}

	var exists int
	err := database.DB.QueryRow("SELECT COUNT(1) FROM events WHERE id = $1", decryptedID).Scan(&exists)
	if err != nil || exists == 0 {
		utils.HandleError(c, "Invalid event_id: Event does not exist", err)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		utils.HandleError(c, "Failed to parse form", err)
		return
	}

	existingSponsors := make(map[int]bool)
	existingSponsorLogos := make(map[int]string)
	rows, err := database.DB.Query("SELECT id, sponsor_logo FROM event_has_sponsors WHERE event_id = $1", decryptedID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var logoPath string
			if err := rows.Scan(&id, &logoPath); err == nil {
				existingSponsors[id] = false
				existingSponsorLogos[id] = logoPath
			}
		}
	}

	sponsorIDs := form.Value["sponsor_ids"]
	sponsorTitles := form.Value["sponsor_titles"]

	if len(sponsorTitles) == 0 && len(form.File) == 0 {
		utils.HandleError(c, "No sponsor data provided")
		return
	}

	var updatedSponsors []gin.H

	sponsorFiles := make(map[int]*multipart.FileHeader)
	for i := range sponsorTitles {
		fileKey := fmt.Sprintf("files[%d]", i)
		if fileHeaders, ok := form.File[fileKey]; ok && len(fileHeaders) > 0 {
			sponsorFiles[i] = fileHeaders[0]
		}
	}

	for i := range sponsorTitles {
		title := sponsorTitles[i]
		file, hasFile := sponsorFiles[i]

		if title == "" && !hasFile {
			continue
		}

		var sponsorID int
		var filePath string
		var isUpdate bool

		if i < len(sponsorIDs) && sponsorIDs[i] != "" {
			id, err := strconv.Atoi(sponsorIDs[i])
			if err == nil {
				sponsorID = id
				isUpdate = true
				existingSponsors[sponsorID] = true

				if existingLogoPath, exists := existingSponsorLogos[sponsorID]; exists {
					filePath = existingLogoPath
				}
			}
		}

		if hasFile {
			if isUpdate && filePath != "" {
				_ = utils.DeleteFile(filePath)
			}
			filename := fmt.Sprintf("sponsor_%d_%d_%d.png", decryptedID, i, time.Now().UnixNano())
			filePath = filepath.Join(SponsorFolderPath, filename)
			err := c.SaveUploadedFile(file, filePath)
			if err != nil {
				utils.HandleError(c, "Failed to save file", err)
				return
			}
		} else if !isUpdate {
			filePath = ""
		}

		if isUpdate {
			query := `UPDATE event_has_sponsors SET sponsor_title = $1, sponsor_logo = $2 WHERE id = $3`
			_, err = database.DB.Exec(query, title, filePath, sponsorID)
			if err != nil {
				utils.HandleError(c, "Failed to update sponsor", err)
				return
			}
		} else {
			query := `INSERT INTO event_has_sponsors (event_id, sponsor_title, sponsor_logo) VALUES ($1, $2, $3) RETURNING id`
			err = database.DB.QueryRow(query, decryptedID, title, filePath).Scan(&sponsorID)
			if err != nil {
				utils.HandleError(c, "Failed to insert sponsor", err)
				return
			}
		}

		updatedSponsors = append(updatedSponsors, gin.H{
			"sponsor_id":    sponsorID,
			"sponsor_title": title,
			"sponsor_logo":  filePath,
		})
	}

	deletedCount := 0
	for id, found := range existingSponsors {
		if !found {
			if logoPath, exists := existingSponsorLogos[id]; exists && logoPath != "" {
				_ = utils.DeleteFile(logoPath)
			}
			_, err = database.DB.Exec("DELETE FROM event_has_sponsors WHERE id = $1", id)
			if err != nil {
				utils.HandleError(c, "Failed to delete removed sponsor", err)
				return
			}
			deletedCount++
		}
	}

	utils.HandleSuccess(c, "Sponsors updated successfully", gin.H{
		"updated_sponsors_count": len(updatedSponsors),
		"deleted_sponsors_count": deletedCount,
		"sponsors":               updatedSponsors,
	})
}

func DeleteSponsorLogo(c *gin.Context) {
	encryptedID := c.Param("id") // sponsor_id
	if encryptedID == "" {
		utils.HandleError(c, "sponsor_id is required")
		return
	}

	// Decrypt sponsor ID
	decryptedSponsorID, err := crypto.NDecrypt(encryptedID)
	if err != nil {
		plainID, err := strconv.ParseInt(encryptedID, 10, 64)
		if err != nil || plainID <= 0 {
			utils.HandleError(c, "Invalid sponsor_id format", err)
			return
		}
		decryptedSponsorID = plainID
	}

	// Verify sponsor exists
	var existingLogo string
	err = database.DB.QueryRow(`SELECT sponsor_logo FROM event_has_sponsors WHERE id = $1`, decryptedSponsorID).Scan(&existingLogo)
	if err == sql.ErrNoRows {
		utils.HandleError(c, "Sponsor not found", err)
		return
	}
	if err != nil {
		utils.HandleError(c, "Failed to fetch sponsor", err)
		return
	}

	// Update DB to clear sponsor_logo
	query := `UPDATE event_has_sponsors SET sponsor_logo = '' WHERE id = $1`
	_, err = database.DB.Exec(query, decryptedSponsorID)
	if err != nil {
		utils.HandleError(c, "Failed to delete sponsor logo in DB", err)
		return
	}

	// Delete logo file if exists
	if existingLogo != "" {
		if err := os.Remove(existingLogo); err != nil && !os.IsNotExist(err) {
			fmt.Println("Failed to delete sponsor logo file:", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Sponsor logo deleted successfully",
		"data": gin.H{
			"sponsor_id": decryptedSponsorID,
		},
	})
}
