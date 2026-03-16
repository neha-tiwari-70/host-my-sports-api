package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func SendSMS(mobileNo string, otp string) error {
	apiURL := "https://www.smsgatewayhub.com/api/mt/SendSMS"

	apiKey := os.Getenv("API_KEY")
	senderid := os.Getenv("SENDER_ID")
	route := os.Getenv("ROUTE")
	entityId := os.Getenv("ENTITY_ID")
	dltTemplateId := os.Getenv("DLT_TEMPLATE_ID")

	message := fmt.Sprintf(`Dear User,

Your OTP to log in to HostMySports.com is %s. For your eyes only - don't share it.

HostMySports Team
Floreo Healthcare & Sports LLP`, otp)

	params := url.Values{}
	params.Set("APIKey", apiKey)
	params.Set("senderid", senderid)
	params.Set("channel", "2")
	params.Set("DCS", "0")
	params.Set("flashsms", "0")
	params.Set("number", mobileNo)
	params.Set("text", message)
	params.Set("route", route)
	params.Set("EntityId", entityId)
	params.Set("dlttemplateid", dltTemplateId)

	finalURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	resp, err := http.Get(finalURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send SMS. Status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
