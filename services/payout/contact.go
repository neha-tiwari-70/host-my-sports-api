package payout

import (
	"bytes"
	"encoding/json"

	"fmt"
	"io"
	"net/http"
	adminPaymentModel "sports-events-api/models/adminPayment"

	"os"
	"sports-events-api/models"
)

func HandleContacts(organizerDetails *models.User) (*adminPaymentModel.Contact, error) {
	//  Step 1 : Check if contact exists
	contact, err := adminPaymentModel.GetContactByOrganizerId(organizerDetails.ID)
	if err == nil {
		return contact, nil
	}

	// Create new contact
	razorpayContactID, err := MakeRazorpayContact(organizerDetails)
	if err != nil {
		return nil, err
	}

	contact = &adminPaymentModel.Contact{
		RazorpayContactId: razorpayContactID,
		OrganizerId:       organizerDetails.ID,
		Name:              organizerDetails.Name,
		Email:             organizerDetails.Email,
		MobileNo:          organizerDetails.MobileNo,
		Type:              "customer",
	}

	contact.Id, err = adminPaymentModel.InsertContactDetails(contact)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func MakeRazorpayContact(organizerDetails *models.User) (string, error) {
	// get razorpay key_id and secret-key from env
	razKeyId := os.Getenv("KEY_ID")
	razSecretKey := os.Getenv("KEY_SECRET")

	//  url to make api call for creating a contact
	contactAPIUrl := "https://api.razorpay.com/v1/contacts"

	// defining paylaod map to send in razorpay's contact api
	payload := map[string]interface{}{
		"name":    organizerDetails.Name,
		"email":   organizerDetails.Email,
		"contact": organizerDetails.MobileNo,
		"type":    "customer",
	}

	//  razorpay accepts json body request hence converting our payload to json body
	payloadJsonBytes, _ := json.Marshal(payload)

	//  making an http request with the request method, url and payload
	req, _ := http.NewRequest("POST", contactAPIUrl, bytes.NewBuffer(payloadJsonBytes))

	// setting basic auth for razorpay api call
	req.SetBasicAuth(razKeyId, razSecretKey)

	// setting content type for this api request
	req.Header.Set("Content-Type", "application/json")

	// make a client which will send the http request to the server
	client := &http.Client{}
	resp, err := client.Do(req) // client is do the req to the server

	// if connection or req sending err then return it
	if err != nil {
		return "", err
	}

	// close the http connection to avoid resource leakage
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Razorpay response body: %w", err)
	}

	fmt.Printf("Raw Response Body: %s\n", string(bodyBytes)) // Debug print

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Razorpay API error : %s", string(body))
	}

	razorPayRes := make(map[string]interface{})
	if err := json.Unmarshal(bodyBytes, &razorPayRes); err != nil {
		return "", fmt.Errorf("failed to decode JSON response: %w", err)
	}

	if razorpayContactId, ok := razorPayRes["id"].(string); ok {
		return razorpayContactId, nil
	}

	return "", fmt.Errorf("failed to create contact: missing ID")

}
