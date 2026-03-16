package adminpayment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sports-events-api/controllers"
	adminPaymentModel "sports-events-api/models/adminPayment"
	"strings"

	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/services/payout"
	"sports-events-api/utils"
	"sync"

	"github.com/gin-gonic/gin"
)

func PayoutHandling(c *gin.Context) {

	// --------------Step 1 : RECEIVE PAYLOAD AND CHECK THE ORGANIZER'S BANK DETAILS, CONTACT AND FUND ACCOUNT DATA----------------

	// Process 1 - Receive payload
	var payload adminPaymentModel.PayoutRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Request", "details": err.Error()})
		return
	}

	// --Decrypt ids
	organizer_id, err := crypto.NDecrypt(payload.OrganizerId)
	if err != nil {
		utils.HandleError(c, "Unable to decrypt the organizer's id.", err)
		return
	}

	event_id, err := crypto.NDecrypt(payload.EventId)
	if err != nil {
		utils.HandleError(c, "Unable to decrypt the event's id.", err)
		return
	}
	// making go routines to fetch the bank details and the user details of the user.
	var organizerDetails *models.User
	var bankDetails *models.PaymentInfo
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// -- Fetch organizer's bank details
		bankDetails, err = models.GetBankDetailsByUserID(organizer_id)
		if err != nil {
			utils.HandleError(c, "Failed to fetch the organizer's bank details.", err)
			return
		}

		// -- Make sure that bank details are not empty
		isBankDetailsEmpty := controllers.IsBankInfoEmpty(bankDetails)
		if isBankDetailsEmpty {
			utils.HandleError(c, "Organizer's bank details are empty.")
			return
		}
	}()

	go func() {
		defer wg.Done()
		organizerDetails, err = models.GetUserByID(int(organizer_id))
		if err != nil {
			utils.HandleError(c, "Failed to fetch the organizer's details", err)
			return
		}
	}()
	wg.Wait()

	contactDetails, err := payout.HandleContacts(organizerDetails)
	if err != nil {
		utils.HandleError(c, "Failed to identify the organizer", err)
		return
	}

	fundDetails := payout.HandleFundAccounts(c, bankDetails, contactDetails.Id, contactDetails.RazorpayContactId)
	if fundDetails == nil {
		return
	}

	// fmt.Println(fundDetails)
	// utils.HandleSuccess(c, "Contacts created successfully", gin.H{
	// 	"contact":     contactDetails,
	// 	"fundAccount": fundDetails,
	// })

	payoutResp, err := payout.MakeRazorpayPayout(fundDetails.RazorpayFundAccountId, fundDetails.AccountType, payload.Amount)

	// Create RazorpayPayout object regardless of error
	payoutData := &adminPaymentModel.RazorpayPayout{
		OrganizerId:      fundDetails.OrganizerId,
		EventId:          event_id,
		FundAccountId:    fundDetails.Id,
		RazorpayPayoutId: payoutResp.ID,
		Amount:           payload.Amount,
		Currency:         payoutResp.Currency,
		Mode:             payoutResp.Mode,
		Purpose:          payoutResp.Purpose,
		FailureReason:    payoutResp.FailureReason,
		ErrorCode:        payoutResp.ErrorCode,
		ErrorDescription: payoutResp.ErrorDescription,
		Status:           payoutResp.Status,
	}

	createdPayoutId, dbErr := adminPaymentModel.InsertPayoutData(payoutData)
	if dbErr != nil {
		utils.HandleError(c, "Unable to store the payout data", dbErr)
		return
	}

	// Encrypt IDs
	payoutData.EncId = crypto.NEncrypt(createdPayoutId)
	payoutData.EncOrganizerId = crypto.NEncrypt(payoutData.OrganizerId)
	payoutData.EncFundAccountId = crypto.NEncrypt(payoutData.FundAccountId)
	payoutData.EncEventId = crypto.NEncrypt(payoutData.EventId)

	// Notify user with appropriate message
	if err != nil && payoutResp.FailureReason == "insufficient_funds" {
		utils.HandleError(c, "Payment could not be completed due to insufficient balance.", err)
		return
	}

	if err != nil {
		utils.HandleError(c, "Payment failed due to server problem.", err)
		return
	}
	utils.HandleSuccess(c, "Payment completed successfully", payoutData)
}

func RazorpayWebhook(c *gin.Context) {
	// 1️⃣ Read webhook body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.HandleError(c, "Unable to read webhook payload", err)
		return
	}

	// Optional: log full payload for debugging/audit
	fmt.Println("Webhook payload received:", string(body))
	// You can store in DB if needed:
	// adminPaymentModel.InsertWebhookLog("<payout_id_placeholder>", string(body))

	// 2️⃣ Parse JSON safely
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		utils.HandleError(c, "Invalid webhook payload", err)
		return
	}

	// 3️⃣ Extract event type
	event, ok := payload["event"].(string)
	if !ok {
		utils.HandleError(c, "Invalid webhook payload: missing event field", nil)
		return
	}

	// Only process payout events
	if !strings.HasPrefix(event, "payout.") {
		utils.HandleSuccess(c, "Ignored non-payout event", nil)
		return
	}

	// 4️⃣ Extract payout entity safely
	payoutEntity, ok := payload["payload"].(map[string]interface{})["payout"].(map[string]interface{})["entity"].(map[string]interface{})
	if !ok {
		utils.HandleError(c, "Invalid webhook payload: missing payout entity", nil)
		return
	}

	payoutID, _ := payoutEntity["id"].(string)
	status, _ := payoutEntity["status"].(string)
	failureReason := ""
	if fr, exists := payoutEntity["failure_reason"]; exists && fr != nil {
		failureReason, _ = fr.(string)
	}

	// Optional: error object
	errorCode := ""
	errorDescription := ""
	if errObj, exists := payoutEntity["error"].(map[string]interface{}); exists && errObj != nil {
		if code, ok := errObj["code"].(string); ok {
			errorCode = code
		}
		if desc, ok := errObj["description"].(string); ok {
			errorDescription = desc
		}
	}

	// 5️⃣ Update payouts table in DB
	err = adminPaymentModel.UpdatePayoutStatusByRazorpayIDFull(payoutID, status, failureReason, errorCode, errorDescription)
	if err != nil {
		utils.HandleError(c, "Unable to update payout status", err)
		return
	}

	utils.HandleSuccess(c, "Payout status updated successfully", gin.H{
		"payout_id":      payoutID,
		"status":         status,
		"failure_reason": failureReason,
		"error_code":     errorCode,
		"error_desc":     errorDescription,
	})
}

// func VerifyRazorpayXSignature(body []byte, signature string, secret string) bool {
// 	h := hmac.New(sha256.New, []byte(secret))
// 	h.Write(body)
// 	expectedSignature := hex.EncodeToString(h.Sum(nil))
// 	return expectedSignature == signature
// }
