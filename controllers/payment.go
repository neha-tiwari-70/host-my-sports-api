package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	razorpay "github.com/razorpay/razorpay-go"
)

// this will help us to make and entry for the order id df
func InitializeEventPayment(c *gin.Context) {
	var EventTransaction models.EventTransaction
	//extaract and decrypt the encrypted ID
	EventTransaction.UserId = DecryptParamId(c, "userId", true)
	if EventTransaction.UserId == 0 {
		return
	}
	EventTransaction.EventId = DecryptParamId(c, "eventId", true)
	if EventTransaction.EventId == 0 {
		return
	}

	valid, err := IsEventActive(EventTransaction.EventId)
	if err != nil {
		utils.HandleError(c, "Oops somthing went wrong", err)
	}
	if !valid {
		utils.HandleInvalidEntries(c, "Event not found", fmt.Errorf("no active event found"))
		return
	}

	paymentDone, err := models.GetTransactionByEventAndUserId(EventTransaction.EventId, EventTransaction.UserId)
	if err != nil {
		utils.HandleError(c, "Error checking existing transaction", err)
		return
	}

	if paymentDone != nil {
		paymentDone.EncEventId = crypto.NEncrypt(paymentDone.EventId)
		paymentDone.EncUserId = crypto.NEncrypt(paymentDone.UserId)
		paymentDone.EncId = crypto.NEncrypt(paymentDone.Id)

		utils.HandleSuccess(c, "Payment already done.", paymentDone)
		return
	}

	Event, err := models.GetEventByID(int(EventTransaction.EventId))
	if err != nil {
		utils.HandleError(c, "Could not find any event", err)
		return
	}

	err = godotenv.Load()
	if err != nil {
		fmt.Println("Error loading env", err)
		return
	}
	EventTransaction.Fees, _ = strconv.Atoi(Event.Fees)
	uid := uuid.New().String()                                 // e.g., "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	receipt := "rcpt_" + strings.ReplaceAll(uid, "-", "")[:30] // total 40 chars

	keyId := os.Getenv("KEY_ID")
	keySecret := os.Getenv("KEY_SECRET")
	client := razorpay.NewClient(keyId, keySecret)

	orderData := map[string]interface{}{
		"amount":   EventTransaction.Fees * 100,
		"currency": "INR",
		"receipt":  receipt,
	}

	body, err := client.Order.Create(orderData, map[string]string{})
	if err != nil {
		utils.HandleError(c, "Unable to initialize payment.", err)
		fmt.Println("error : ", err)
		return
	}

	EventTransaction.OrderId = body["id"].(string)

	createdTransaction, err := models.CreateEventTransaction(&EventTransaction)
	if err != nil {
		utils.HandleError(c, "Failed to save transaction", err)
		return
	}

	createdTransaction.EncEventId = crypto.NEncrypt(createdTransaction.EventId)
	createdTransaction.EncUserId = crypto.NEncrypt(createdTransaction.UserId)
	createdTransaction.EncId = crypto.NEncrypt(createdTransaction.Id)

	utils.HandleSuccess(c, "Order Initialized Successfully", createdTransaction)
}

func HandleRazorPayResponse(c *gin.Context) {
	var payload struct {
		OrderId   string `json:"order_id"`
		PaymentId string `json:"payment_id"`
		Signature string `json:"signature"`
		Status    string `json:"status"` // "success" or "failed"
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.HandleError(c, "Invalid payload", err)
		return
	}

	transaction, err := models.GetEventTransactionByOrderId(payload.OrderId)
	if err != nil || transaction == nil {
		utils.HandleError(c, "Transaction not found for given order ID", err)
		return
	}

	// Load Razorpay secret key from env
	keySecret := os.Getenv("KEY_SECRET")
	if keySecret == "" {
		utils.HandleError(c, "Payment secret not configured", nil)
		return
	}

	if payload.Status == "success" {
		// ✅ Step: Verify signature
		isValid := VerifyRazorpaySignature(payload.OrderId, payload.PaymentId, payload.Signature, keySecret)
		if !isValid {
			utils.HandleError(c, "Invalid Razorpay signature", nil)
			return
		}

		transaction.PaymentId = &payload.PaymentId
		transaction.Signature = &payload.Signature
		transaction.PaymentStatus = "Success"
	} else {
		transaction.PaymentStatus = "Failed"
	}

	if err := models.UpdateEventTransaction(transaction); err != nil {
		utils.HandleError(c, "Failed to update transaction", err)
		return
	}

	utils.HandleSuccess(c, "Transaction status updated", nil)
}

// This is the function which handles our razorpay webhook and updates our table automatically with razorpay response when razorpay call it.
func RazorpayWebhookHandler(c *gin.Context) {
	var webhookPayload struct {
		Payload struct {
			Payment struct {
				Entity struct {
					ID            string  `json:"id"`
					OrderID       string  `json:"order_id"`
					CardID        *string `json:"card_id"`
					Method        string  `json:"method"`
					Status        string  `json:"status"`
					International bool    `json:"international"`
					Bank          *string `json:"bank"`
					Wallet        *string `json:"wallet"`
					Email         string  `json:"email"`
					Contact       string  `json:"contact"`
					Card          *struct {
						Last4   string `json:"last4"`
						Name    string `json:"name"`
						Type    string `json:"type"`
						Network string `json:"network"`
						Issuer  string `json:"issuer"`
						Emi     bool   `json:"emi"`
					} `json:"card"`
					Amount           int                    `json:"amount"`
					AcquirerData     map[string]interface{} `json:"acquirer_data"`
					ErrorDescription *string                `json:"error_description"`
					ErrorSource      *string                `json:"error_source"`
					ErrorStep        *string                `json:"error_step"`
					ErrorReason      *string                `json:"error_reason"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}

	if err := c.ShouldBindJSON(&webhookPayload); err != nil {
		utils.HandleError(c, "Invalid webhook payload", err)
		return
	}

	entity := webhookPayload.Payload.Payment.Entity

	query := `INSERT INTO verified_transaction
    (razorpay_payment_id, payment_method, card_id, bank, wallet, upi, email, contact,
    card_details, card_holder_name, card_type, card_network, issuer, emi,
    bank_transaction_id, rrn, upi_transaction_id, auth_code, amount,
    error_description, error_source, error_step, error_reason, order_id, status, created_at, updated_at)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25, NOW(), NOW())
    ON CONFLICT (razorpay_payment_id) DO UPDATE SET
        status = EXCLUDED.status,
        updated_at = NOW();`

	// Prepare values from entity (handle nils carefully)

	emi := false
	if entity.Card != nil {
		emi = entity.Card.Emi
	}

	// Extract acquirer data safely as strings
	var bankTransactionID, rrn, upiTransactionID, authCode *string

	if entity.Method == "netbanking" {
		if val, ok := entity.AcquirerData["bank_transaction_id"].(string); ok {
			bankTransactionID = &val
		}
	}
	if entity.Method == "upi" {
		if val, ok := entity.AcquirerData["rrn"].(string); ok {
			rrn = &val
		}
		if entity.Status == "captured" {
			if val, ok := entity.AcquirerData["upi_transaction_id"].(string); ok {
				upiTransactionID = &val
			}
		}
	}
	if entity.Method == "card" {
		if val, ok := entity.AcquirerData["auth_code"].(string); ok {
			authCode = &val
		}
	}

	// Then execute the insert query with all fields (convert nil *string to nil interface{} or "" as needed)

	// Example:
	_, err := database.DB.Exec(query,
		entity.ID,
		entity.Method,
		entity.CardID,
		entity.Bank,
		entity.Wallet,
		nil, // upi field? You can add if available
		entity.Email,
		entity.Contact,
		func() interface{} {
			if entity.Card != nil {
				return entity.Card.Last4
			}
			return nil
		}(),
		func() interface{} {
			if entity.Card != nil {
				return entity.Card.Name
			}
			return nil
		}(),
		func() interface{} {
			if entity.Card != nil {
				return entity.Card.Type
			}
			return nil
		}(),
		func() interface{} {
			if entity.Card != nil {
				return entity.Card.Network
			}
			return nil
		}(),
		func() interface{} {
			if entity.Card != nil {
				return entity.Card.Issuer
			}
			return nil
		}(),
		emi,
		bankTransactionID,
		rrn,
		upiTransactionID,
		authCode,
		entity.Amount/100,
		entity.ErrorDescription,
		entity.ErrorSource,
		entity.ErrorStep,
		entity.ErrorReason,
		entity.OrderID,
		entity.Status,
	)
	if err != nil {
		utils.HandleError(c, "Failed to save verified transaction", err)
		return
	}

	//  Map Razorpay status to internal status
	finalStatus := "Pending"
	switch entity.Status {
	case "authorized", "captured":
		finalStatus = "Success"
	case "failed":
		finalStatus = "Failed"
	}

	//  Update event_transactions table
	updateQuery := `
		UPDATE event_transactions
		SET payment_status = $1,
		    razor_payment_id = $2,
		    updated_at = NOW()
		WHERE razor_order_id = $3;
	`
	_, err = database.DB.Exec(updateQuery, finalStatus, entity.ID, entity.OrderID)
	if err != nil {
		utils.HandleError(c, "Failed to update event transaction status", err)
		return
	}

	utils.HandleSuccess(c, "Webhook processed successfully", nil)
}

// this function verifies the signature with help or orderId, paymentId, secret for security reason
func VerifyRazorpaySignature(orderId, paymentId, razorpaySignature, secret string) bool {
	data := orderId + "|" + paymentId

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	computedSignature := hex.EncodeToString(h.Sum(nil))

	return computedSignature == razorpaySignature
}

func GetPaymentStatus(c *gin.Context) {
	//extaract and decrypt the encrypted ID
	decUserId := DecryptParamId(c, "user_id", true)
	if decUserId == 0 {
		return
	}
	decEventId := DecryptParamId(c, "event_id", true)
	if decEventId == 0 {
		return
	}

	var paymentObj *models.EventTransaction
	valid, err := IsEventActive(decEventId)
	if err != nil {
		utils.HandleError(c, "Oops somthing went wrong", err)
	}
	if !valid {
		utils.HandleInvalidEntries(c, "Event not found", fmt.Errorf("no active event found"))
		return
	}

	event, err := models.GetEventByID(int(decEventId))
	if err != nil {
		utils.HandleError(c, "Unable to fetch event data", err)
		return
	}

	fees := event.Fees
	paymentObj, err = models.GetTransactionByEventAndUserId(decEventId, decUserId)
	if err != nil {
		utils.HandleError(c, "Unable to fetch payment status", err)
		return
	}

	if paymentObj != nil || fees == "0" {
		utils.HandleSuccess(c, "Payment already done", gin.H{
			"payment_done": true,
		})
		return
	} else {
		utils.HandleSuccess(c, "Payment pending", gin.H{
			"payment_done": false,
		})
		return
	}
}
