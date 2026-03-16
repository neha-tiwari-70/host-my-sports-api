package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"
)

type PaymentInfo struct {
	InfoEncId         string `form:"info_id"`
	InfoId            int64  `form:"-"`
	EncUserId         string `form:"user_id"`
	UserId            int64  `form:"-"`
	UPIId             string `form:"upi_id"`
	AccountName       string `form:"account_name"`
	AccountNo         string `form:"account_no"`
	AccountType       string `form:"account_type"`
	BranchName        string `form:"branch_name"`
	IFSCCode          string `form:"ifsc_code"`
	QRCode            string `form:"qr_code"`
	RazorpayContactId string `json:"razorpayContactId,omitempty"`
}

func EditBankDetails(bankData *PaymentInfo) (*PaymentInfo, error) {
	// Check if a record already exists for the user
	var existingID int64
	err := database.DB.QueryRow("SELECT id FROM bank_details WHERE user_id = $1", bankData.UserId).Scan(&existingID)

	if err != nil {
		if err == sql.ErrNoRows {
			// No record found, do an insert
			query := `
				INSERT INTO bank_details (
					user_id, upi_id, qr_code, account_name, account_no, account_type, branch_name, ifsc_code
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				RETURNING id`
			err = database.DB.QueryRow(
				query,
				bankData.UserId,
				bankData.UPIId,
				bankData.QRCode,
				bankData.AccountName,
				bankData.AccountNo,
				bankData.AccountType,
				bankData.BranchName,
				bankData.IFSCCode,
			).Scan(&bankData.InfoId)
			if err != nil {
				return nil, fmt.Errorf("error inserting bank details: %v", err)
			}
		} else {
			return nil, fmt.Errorf("error checking existing bank details: %v", err)
		}
	} else {
		// Record exists, do an update
		query := `
			UPDATE bank_details
			SET upi_id = $1,
			    qr_code = $2,
				account_name = $3,
				account_no = $4,
				account_type = $5,
				branch_name = $6,
				ifsc_code = $7,
				updated_at = NOW()
			WHERE user_id = $8
			RETURNING id`
		err = database.DB.QueryRow(
			query,
			bankData.UPIId,
			bankData.QRCode,
			bankData.AccountName,
			bankData.AccountNo,
			bankData.AccountType,
			bankData.BranchName,
			bankData.IFSCCode,
			bankData.UserId,
		).Scan(&bankData.InfoId)
		if err != nil {
			return nil, fmt.Errorf("error updating bank details: %v", err)
		}
	}

	return bankData, nil
}

func GetBankDetailsByUserID(userID int64) (*PaymentInfo, error) {
	query := `
		SELECT
			id,
			user_id,
			upi_id,
			account_name,
			account_no,
			account_type,
			branch_name,
			ifsc_code,
			qr_code
		FROM bank_details
		WHERE user_id = $1
		LIMIT 1
	`

	row := database.DB.QueryRow(query, userID)

	var info PaymentInfo
	err := row.Scan(
		&info.InfoId,
		&info.UserId,
		&info.UPIId,
		&info.AccountName,
		&info.AccountNo,
		&info.AccountType,
		&info.BranchName,
		&info.IFSCCode,
		&info.QRCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no bank details found for this user")
		}
		return nil, fmt.Errorf("error fetching bank details: %v", err)
	}

	return &info, nil
}

func DeactivateFundAccountByOrganizerID(organizerID int64) error {
	query := `
		UPDATE fund_accounts 
		SET status = 'inactive', updated_at = CURRENT_TIMESTAMP 
		WHERE organizer_id = $1 AND status = 'active'
	`
	_, err := database.DB.Exec(query, organizerID)
	return err
}
