package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AddBankInfo(c *gin.Context) {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		utils.HandleError(c, "Invalid form data", err)
		return
	}

	// Get form fields
	encUserID := c.PostForm("user_id")
	upiID := c.PostForm("upi_id")
	accountName := c.PostForm("account_name")
	accountNo := c.PostForm("account_no")
	accountType := c.PostForm("account_type")
	branchName := c.PostForm("branch_name")
	ifscCode := c.PostForm("ifsc_code")
	removeQRCode := c.PostForm("remove_qr_code") == "true"

	// Decrypt user ID
	userID, err := crypto.NDecrypt(encUserID)
	if err != nil {
		utils.HandleError(c, "Failed to decrypt user ID", err)
		return
	}

	user, err := models.GetUserByID(int(userID))
	if err != nil {
		utils.HandleError(c, "Unable to find user", err)
		return
	}

	existingBankInfo, _ := models.GetBankDetailsByUserID(userID)
	var qrCodePath string

	if removeQRCode {
		// Delete old QR code file if exists
		if existingBankInfo != nil && existingBankInfo.QRCode != "" {
			_ = os.Remove(existingBankInfo.QRCode) // Ignore error if file doesn't exist
		}
		qrCodePath = "" // Remove from DB
	} else {
		// Try to get the uploaded file
		file, header, err := c.Request.FormFile("qr_code")
		if err == nil && file != nil {
			defer file.Close()
			fileExt := filepath.Ext(header.Filename)
			safeName := strings.ReplaceAll(user.Name, " ", "_")
			fileName := fmt.Sprintf("qr_%s_%d%s", safeName, time.Now().Unix(), fileExt)
			uploadPath := filepath.Join("public/uploads", fileName)

			out, err := os.Create(uploadPath)
			if err != nil {
				utils.HandleError(c, "Failed to save QR code", err)
				return
			}
			defer out.Close()

			_, err = io.Copy(out, file)
			if err != nil {
				utils.HandleError(c, "Failed to save QR code", err)
				return
			}

			qrCodePath = uploadPath
		} else {
			// No new file, keep the existing one
			if existingBankInfo != nil {
				qrCodePath = existingBankInfo.QRCode
			}
		}
	}

	bankInfo := models.PaymentInfo{
		UserId:      userID,
		UPIId:       upiID,
		AccountName: accountName,
		AccountNo:   accountNo,
		AccountType: accountType,
		BranchName:  branchName,
		IFSCCode:    ifscCode,
		QRCode:      qrCodePath,
	}

	updatedBankDetails, err := models.EditBankDetails(&bankInfo)
	if err != nil {
		utils.HandleError(c, "Unable to update bank details", err)
		return
	}

	err = models.DeactivateFundAccountByOrganizerID(userID)
	if err != nil {
		utils.HandleError(c, "Failed to deactivate existing fund account", err)
		return
	}

	updatedBankDetails.EncUserId = crypto.NEncrypt(updatedBankDetails.UserId)
	updatedBankDetails.InfoEncId = crypto.NEncrypt(updatedBankDetails.InfoId)

	utils.HandleSuccess(c, "Bank details updated successfully", gin.H{
		"user_id":      updatedBankDetails.EncUserId,
		"info_id":      updatedBankDetails.InfoEncId,
		"upi_id":       updatedBankDetails.UPIId,
		"account_name": updatedBankDetails.AccountName,
		"account_no":   updatedBankDetails.AccountNo,
		"account_type": updatedBankDetails.AccountType,
		"branch_name":  updatedBankDetails.BranchName,
		"ifsc_code":    updatedBankDetails.IFSCCode,
		"qr_code":      updatedBankDetails.QRCode,
	})
}

func GetBankInfo(c *gin.Context) {
	userID := DecryptParamId(c, "user_id", true)
	if userID == 0 {
		return
	}

	user, err := models.GetUserByID(int(userID))
	if err != nil {
		utils.HandleError(c, "Unable to fetch user's data.", err)
		return
	}

	// Fetch bank details from DB
	bankInfo, err := models.GetBankDetailsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": err.Error(),
			"data":    nil,
		})
		return
	}
	bankInfo.EncUserId = crypto.NEncrypt(bankInfo.UserId)
	bankInfo.InfoEncId = crypto.NEncrypt(bankInfo.InfoId)
	// bankQRCode := bankInfo.QRCode
	// defaultLogoPath := "public/static/staticTeamLogo.png"
	// if team.TeamLogoPath == "" || !fileExists(teamLogoPath) {
	// 	teamLogoPath = defaultLogoPath
	// }

	utils.HandleSuccess(c, "Bank details fetched successfully", gin.H{
		"user_id":      bankInfo.EncUserId,
		"info_id":      bankInfo.InfoEncId,
		"upi_id":       bankInfo.UPIId,
		"account_name": bankInfo.AccountName,
		"account_no":   bankInfo.AccountNo,
		"account_type": bankInfo.AccountType,
		"branch_name":  bankInfo.BranchName,
		"ifsc_code":    bankInfo.IFSCCode,
		"qr_code":      bankInfo.QRCode,
		"user":         user,
	})
}

func CheckOrganizerBankDetailsEmptyOrNot(c *gin.Context) {
	encUserID := c.Param("user_id")
	if encUserID == "" {
		utils.HandleError(c, "User ID is required", fmt.Errorf("missing user_id"))
		return
	}

	// Decrypt the user ID
	userID, err := crypto.NDecrypt(encUserID)
	if err != nil {
		utils.HandleError(c, "Failed to decrypt user ID", err)
		return
	}

	user, err := models.GetUserByID(int(userID))
	if err != nil {
		utils.HandleError(c, "Unable to fetch user's data.", err)
		return
	}

	// Fetch bank details from DB
	bankInfo, err := models.GetBankDetailsByUserID(userID)
	if err != nil {
		utils.HandleSuccess(c, "No bank details found for this user.", gin.H{
			"bankDetailsPresent": false,
			"user":               user,
		})
		return
	}
	bankInfo.EncUserId = crypto.NEncrypt(bankInfo.UserId)
	bankInfo.InfoEncId = crypto.NEncrypt(bankInfo.InfoId)

	// Check if bank info fields are empty
	if IsBankInfoEmpty(bankInfo) {
		utils.HandleSuccess(c, "Bank details are empty", gin.H{
			"bankDetailsPresent": false,
			"user":               user,
		})
		return
	}

	// If bank details present
	utils.HandleSuccess(c, "Bank details fetched successfully", gin.H{
		"bankDetailsPresent": true,
		"user":               user,
	})
}

func IsBankInfoEmpty(info *models.PaymentInfo) bool {
	return info.UPIId == "" &&
		info.AccountName == "" &&
		info.AccountNo == "" &&
		info.AccountType == "" &&
		info.BranchName == "" &&
		info.IFSCCode == "" &&
		info.QRCode == ""
}
