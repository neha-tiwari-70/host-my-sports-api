package payout

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func ExtractUPIFromQR(qrImagePath string) (string, error) {
	// Call zbarimg command-line tool to scan QR code
	cmd := exec.Command("zbarimg", "--quiet", "--raw", qrImagePath)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to scan QR code using zbarimg: %w", err)
	}

	qrData := strings.TrimSpace(out.String())
	fmt.Println("Decoded QR Data:", qrData)

	// Extract UPI ID from QR data
	if upiID := parseUPIIDFromPayload(qrData); upiID != "" {
		return upiID, nil
	}

	return "", fmt.Errorf("UPI ID not found in QR code")
}

func parseUPIIDFromPayload(data string) string {
	const key = "pa="
	start := strings.Index(data, key)
	if start == -1 {
		return ""
	}
	start += len(key)
	end := strings.Index(data[start:], "&")
	if end == -1 {
		return data[start:]
	}
	return data[start : start+end]
}
