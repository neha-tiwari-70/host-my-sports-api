package payout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	adminPaymentModel "sports-events-api/models/adminPayment"

	"sports-events-api/models"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

func HandleFundAccounts(c *gin.Context, organizerBankDetails *models.PaymentInfo, contactId int64, razorpayContactId string) *adminPaymentModel.RazorpayFundAccount {
	// Step 1: Check existing fund account in DB
	existingFA, err := adminPaymentModel.GetFundAccountByOrganizerId(organizerBankDetails.UserId)
	if err == nil && existingFA != nil {
		return existingFA
	}
	// Step 2: Decide payment method (priority based)
	var paymentMethod string
	var upiId string
	if organizerBankDetails.AccountNo != "" && organizerBankDetails.IFSCCode != "" {
		// Priority 1: Bank Account
		paymentMethod = "bank_account"
	} else if organizerBankDetails.UPIId != "" {
		// Priority 2: UPI ID
		paymentMethod = "upi"
		upiId = organizerBankDetails.UPIId
	} else if organizerBankDetails.QRCode != "" {
		// Priority 3: QR Code → Extract UPI ID from QR
		upiId, err = ExtractUPIFromQR(organizerBankDetails.QRCode)
		if err != nil {
			utils.HandleError(c, "Wrong QR code provided", err)
			return nil
		} else if upiId != "" {
			paymentMethod = "upi"
			organizerBankDetails.UPIId = upiId
		}
	}

	if paymentMethod == "" {
		utils.HandleError(c, "Invalid payment method")
		return nil
	}

	// Step 3: Create Razorpay fund account via API
	FARazorpayResponse, err := MakeRazorpayFundAccount(
		razorpayContactId,
		organizerBankDetails,
		paymentMethod,
	)

	if err != nil && FARazorpayResponse.FailureReason == "input_validation_failed" {
		utils.HandleError(c, "Wrong bank details", err)
		return nil
	}

	if err != nil {
		utils.HandleError(c, "Unable to complete the payment", err)
		return nil
	}

	// Step 4: Save fund account to DB
	fundAccount := &adminPaymentModel.RazorpayFundAccount{
		OrganizerId:           organizerBankDetails.UserId,
		ContactId:             contactId, // assuming you store contact ID too
		RazorpayFundAccountId: FARazorpayResponse.RazorpayFundAccountId,
		AccountType:           paymentMethod,
		UPIID:                 upiId,
		AccountNumber:         organizerBankDetails.AccountNo,
		IFSCCode:              organizerBankDetails.IFSCCode,
		BankName:              organizerBankDetails.BranchName,
		Name:                  organizerBankDetails.AccountName,
		Active:                "true",
		Status:                "active",
	}

	fundAccount.Id, err = adminPaymentModel.InsertFundAccount(fundAccount)
	if err != nil {
		utils.HandleError(c, "Unable to insert fund account", err)
		return nil
	}
	// utils.HandleSuccess(c, "Fund account created successfull")
	return fundAccount
}

func MakeRazorpayFundAccount(contactId string, bank *models.PaymentInfo, paymentMethod string) (*adminPaymentModel.RazorpayFundAccount, error) {
	razKeyId := os.Getenv("KEY_ID")
	razSecretKey := os.Getenv("KEY_SECRET")

	fundAccountUrl := "https://api.razorpay.com/v1/fund_accounts"

	var payload map[string]interface{}

	if paymentMethod == "bank_account" {
		payload = map[string]interface{}{
			"contact_id":   contactId,
			"account_type": "bank_account",
			"bank_account": map[string]interface{}{
				"name":           bank.AccountName,
				"ifsc":           bank.IFSCCode,
				"account_number": bank.AccountNo,
			},
		}
	} else if paymentMethod == "upi" {
		payload = map[string]interface{}{
			"contact_id":   contactId,
			"account_type": "vpa",
			"vpa": map[string]interface{}{
				"address": bank.UPIId,
			},
		}
	} else {
		return nil, fmt.Errorf("unsupported payment method: %s", paymentMethod)
	}

	payloadJson, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", fundAccountUrl, bytes.NewBuffer(payloadJson))
	req.SetBasicAuth(razKeyId, razSecretKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// getting body
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Fund Account API Response:", string(body))

	var fundAccount adminPaymentModel.RazorpayFundAccount
	// checking for the error response.
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errObj, ok := errorResp["error"].(map[string]interface{}); ok {
				fundAccount.FailureReason = fmt.Sprintf("%v", errObj["reason"])
				fundAccount.ErrorCode = fmt.Sprintf("%v", errObj["code"])
				fundAccount.ErrorDescription = fmt.Sprintf("%v", errObj["description"])
				fundAccount.Status = "failed"
			}
		}
		return &fundAccount, fmt.Errorf("fund account creation failed : %s", fundAccount.ErrorDescription)
	}

	var razorResponse map[string]interface{}
	if err := json.Unmarshal(body, &razorResponse); err != nil {
		fundAccount.ErrorDescription = err.Error()
		return &fundAccount, fmt.Errorf("invalid Razorpay JSON response: %w", err)
	}
	var ok bool
	if fundAccount.RazorpayFundAccountId, ok = razorResponse["id"].(string); ok {
		return &fundAccount, nil
	}

	return nil, fmt.Errorf("missing fund_account id in Razorpay response")
}
