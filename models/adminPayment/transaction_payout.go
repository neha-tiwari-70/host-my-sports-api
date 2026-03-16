package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"
	"time"
)

type PayoutRequest struct {
	EventId     string `json:"event_id"`
	OrganizerId string `json:"organizer_id"`
	Amount      int    `json:"amount"`
}

type Contact struct {
	EncId             string    `json:"id"`
	Id                int64     `json:"-"`
	EncOrganizerId    string    `json:"organizer_id"`
	OrganizerId       int64     `json:"-"`
	RazorpayContactId string    `json:"razorpay_contact_id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	MobileNo          string    `json:"mobile_no"`
	Type              string    `json:"type"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type RazorpayFundAccount struct {
	EncId                 string    `json:"id"`
	Id                    int64     `json:"-"`
	EncOrganizerId        string    `json:"organizer_id"`
	OrganizerId           int64     `json:"-"`
	EncContactId          string    `json:"contact_id"`
	ContactId             int64     `json:"-"`                        // Link with Contact
	RazorpayFundAccountId string    `json:"razorpay_fund_account_id"` // Razorpay Fund Account ID
	AccountType           string    `json:"account_type"`             // bank_account or vpa
	AccountNumber         string    `json:"account_number"`
	IFSCCode              string    `json:"ifsc_code"`
	BankName              string    `json:"bank_name"`
	Name                  string    `json:"beneficiary_name"`
	UPIID                 string    `json:"upi_id"`
	Active                string    `json:"active"`
	Status                string    `json:"status"`
	FailureReason         string    `json:"failure_reason"`
	ErrorCode             string    `json:"error_code"`
	ErrorDescription      string    `json:"error_description"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type RazorpayPayout struct {
	EncId            string    `json:"id"`
	Id               int64     `json:"-"`
	EncOrganizerId   string    `json:"organizer_id"`
	OrganizerId      int64     `json:"-"`
	EncEventId       string    `json:"event_id"`
	EventId          int64     `json:"-"`
	EncFundAccountId string    `json:"fund_account_id"`
	FundAccountId    int64     `json:"-"`
	RazorpayPayoutId string    `json:"razorpay_payout_id"`
	Amount           int       `json:"amount"`
	Currency         string    `json:"currency"` // INR
	Mode             string    `json:"mode"`     // NEFT, IMPS, UPI
	Purpose          string    `json:"purpose"`  // refund etc.
	Status           string    `json:"status"`   // processing, success, failed
	FailureReason    string    `json:"failure_reason"`
	ErrorCode        string    `json:"error_code"`
	ErrorDescription string    `json:"error_description"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type RazorpayPayoutSuccessResponse struct {
	ID               string `json:"id"`
	Entity           string `json:"entity"`
	FundAccountID    string `json:"fund_account_id"`
	Amount           int    `json:"amount"`
	Currency         string `json:"currency"`
	Mode             string `json:"mode"`
	Purpose          string `json:"purpose"`
	Status           string `json:"status"`
	Utr              string `json:"utr"`
	FailureReason    string `json:"failure_reason"`
	ErrorCode        string `json:"error_code"`
	ErrorDescription string `json:"error_description"`
	CreatedAt        int64  `json:"created_at"`
	// Error            Error  `json:"error"`
	// Add more fields as needed from Razorpay docs
}

// type RazorPayoutErrorResponse struct{

// }

func GetContactByOrganizerId(organizer_id int64) (*Contact, error) {
	var contact Contact
	query := `SELECT * FROM contacts WHERE organizer_id = $1 LIMIT 1`
	err := database.DB.QueryRow(query, organizer_id).Scan(
		&contact.Id,
		&contact.OrganizerId,
		&contact.RazorpayContactId,
		&contact.Name,
		&contact.Email,
		&contact.MobileNo,
		&contact.Type,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact not found")
		}
		return nil, fmt.Errorf("failed to fetch contact: %v", err)
	}

	return &contact, nil
}

func GetFundAccountByOrganizerId(organierId int64) (*RazorpayFundAccount, error) {
	query := `SELECT id, organizer_id, contact_id, razorpay_fund_account_id, account_type,
              upi_id, account_number, ifsc, bank_name, name, active, status, created_at, updated_at
              FROM fund_accounts WHERE organizer_id=$1 AND status = 'active' LIMIT 1`

	row := database.DB.QueryRow(query, organierId)

	var fundAccount RazorpayFundAccount
	err := row.Scan(&fundAccount.Id,
		&fundAccount.OrganizerId,
		&fundAccount.ContactId,
		&fundAccount.RazorpayFundAccountId,
		&fundAccount.AccountType,
		&fundAccount.UPIID,
		&fundAccount.AccountNumber,
		&fundAccount.IFSCCode,
		&fundAccount.BankName,
		&fundAccount.Name,
		&fundAccount.Active,
		&fundAccount.Status,
		&fundAccount.CreatedAt,
		&fundAccount.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("fund account not found")
		}
		return nil, fmt.Errorf("failed to fetch fund account: %v", err)
	}
	return &fundAccount, nil

}

func InsertContactDetails(contact *Contact) (int64, error) {
	query := `INSERT INTO contacts(organizer_id, razorpay_contact_id, name, email, mobile_no, type)
			  VALUES($1, $2, $3, $4, $5, $6) RETURNING id`

	err := database.DB.QueryRow(query,
		contact.OrganizerId,
		contact.RazorpayContactId,
		contact.Name,
		contact.Email,
		contact.MobileNo,
		contact.Type).Scan(&contact.Id)

	if err != nil {
		return 0, err
	}

	return contact.Id, nil
}

func InsertFundAccount(fa *RazorpayFundAccount) (int64, error) {
	query := `INSERT INTO fund_accounts(organizer_id, contact_id, razorpay_fund_account_id, account_type, upi_id, account_number ,ifsc, bank_name, name, active, status)
			  VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`

	err := database.DB.QueryRow(query,
		fa.OrganizerId,
		fa.ContactId,
		fa.RazorpayFundAccountId,
		fa.AccountType,
		fa.UPIID,
		fa.AccountNumber,
		fa.IFSCCode,
		fa.BankName,
		fa.Name,
		fa.Active,
		fa.Status,
	).Scan(&fa.Id)

	if err != nil {
		return 0, err
	}

	return fa.Id, nil
}

func InsertPayoutData(payoutData *RazorpayPayout) (int64, error) {
	query := `INSERT INTO payouts(organizer_id, event_id, fund_account_id, razorpay_payout_id, amount, currency, mode, purpose, failure_reason, error_code, error_description, status)
			  VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`

	err := database.DB.QueryRow(query,
		payoutData.OrganizerId,
		payoutData.EventId,
		payoutData.FundAccountId,
		payoutData.RazorpayPayoutId,
		payoutData.Amount,
		payoutData.Currency,
		payoutData.Mode,
		payoutData.Purpose,
		payoutData.FailureReason,
		payoutData.ErrorCode,
		payoutData.ErrorDescription,
		payoutData.Status,
	).Scan(&payoutData.Id)

	if err != nil {
		return 0, err
	}

	return payoutData.Id, nil
}

func UpdatePayoutStatusByRazorpayIDFull(payoutID, status, failureReason, errorCode, errorDesc string) error {
	query := `
        UPDATE payouts
        SET status = $1,
            failure_reason = $2,
            error_code = $3,
            error_description = $4,
            updated_at = NOW()
        WHERE razorpay_payout_id = $5
    `
	_, err := database.DB.Exec(query, status, failureReason, errorCode, errorDesc, payoutID)
	return err
}
