package payout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	adminPaymentModel "sports-events-api/models/adminPayment"

	"os"
)

func MakeRazorpayPayout(fundAccountID string, accountType string, amount int) (*adminPaymentModel.RazorpayPayoutSuccessResponse, error) {
	url := "https://api.razorpay.com/v1/payouts"
	razKeyId := os.Getenv("KEY_ID")
	razSecretKey := os.Getenv("KEY_SECRET")

	var mode string
	switch accountType {
	case "upi":
		mode = "UPI"
	case "bank_account":
		mode = "IMPS" // or NEFT/RTGS depending on urgency
	default:
		return nil, fmt.Errorf("unsupported account type: %s", accountType)
	}

	payload := map[string]interface{}{
		"account_number":  os.Getenv("RAZORPAYX_ACCOUNT_NUMBER"), // your virtual account no
		"fund_account_id": fundAccountID,
		"amount":          amount * 100,
		"currency":        "INR",
		"mode":            mode,
		"purpose":         "payout",
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	req.SetBasicAuth(razKeyId, razSecretKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Payout API Response:", string(body))
	var payoutResp adminPaymentModel.RazorpayPayoutSuccessResponse
	json.Unmarshal(body, &payoutResp) // attempt parse even if it's an error

	// Include Razorpay error info if available
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		// Custom logic to extract failure reason
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errObj, ok := errorResp["error"].(map[string]interface{}); ok {
				payoutResp.FailureReason = fmt.Sprintf("%v", errObj["reason"])
				payoutResp.ErrorCode = fmt.Sprintf("%v", errObj["code"])
				payoutResp.ErrorDescription = fmt.Sprintf("%v", errObj["description"])
				payoutResp.Status = "failed"
			}
		}
		return &payoutResp, fmt.Errorf("payout failed: %s", payoutResp.ErrorDescription)
	}

	// var payoutResp models.RazorpayPayoutSuccessResponse
	// if err := json.Unmarshal(body, &payoutResp); err != nil {
	// 	return nil, err
	// }

	return &payoutResp, nil
}
