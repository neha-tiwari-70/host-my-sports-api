package models

import (
	"database/sql"
	"fmt"
	"log"
	"sports-events-api/database"
)

type EventTransaction struct {
	EncId      string `json:"id"`
	Id         int64  `json:"-"`
	EncUserId  string `json:"user_id"`
	UserId     int64  `json:"-"`
	EncEventId string `json:"event_id"`
	EventId    int64  `json:"-"`
	// RegistrationStatus string  `json:"registration_status"`
	PaymentStatus string  `json:"payment_status"`
	OrderId       string  `json:"order_id"`
	PaymentId     *string `json:"payment_id"`
	Signature     *string `json:"signature"`
	Fees          int     `json:"fees"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// this function will create an  entry for the event transaction
func CreateEventTransaction(transactionData *EventTransaction) (*EventTransaction, error) {
	query := `
		INSERT INTO event_transactions (
			user_id, event_id,
			razor_order_id, fees
		) VALUES ($1, $2, $3, $4)
		RETURNING id, payment_status, created_at, updated_at
	`

	err := database.DB.QueryRow(query, transactionData.UserId, transactionData.EventId, transactionData.OrderId, transactionData.Fees).Scan(
		&transactionData.Id, &transactionData.PaymentStatus, &transactionData.CreatedAt, &transactionData.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error inserting transaction", err)
	}

	return transactionData, nil
}

func GetTransactionByEventAndUserId(eventId int64, userId int64) (*EventTransaction, error) {
	query := `SELECT 
				id,
				user_id, 
				event_id,
				payment_status,
				razor_order_id,
				razor_payment_id,
				signature,
				fees,
				created_at,
				updated_at
			  FROM event_transactions
			  WHERE event_id = $1
			    AND user_id = $2
				AND payment_status = 'Success'
				AND razor_payment_id IS NOT NULL
				AND signature IS NOT NULL
			  LIMIT 1;
	`
	row := database.DB.QueryRow(query, eventId, userId)

	var et EventTransaction
	err := row.Scan(
		&et.Id,
		&et.UserId,
		&et.EventId,
		&et.PaymentStatus,
		&et.OrderId,
		&et.PaymentId,
		&et.Signature,
		&et.Fees,
		&et.CreatedAt,
		&et.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &et, nil
}

// this will get the transaction by order id to update the payment status
func GetEventTransactionByOrderId(orderId string) (*EventTransaction, error) {
	query := `
		SELECT id, user_id, event_id, payment_status,
		       razor_order_id, razor_payment_id, signature, fees, created_at, updated_at
		FROM event_transactions
		WHERE razor_order_id = $1
	`

	var tx EventTransaction
	err := database.DB.QueryRow(query, orderId).Scan(
		&tx.Id,
		&tx.UserId,
		&tx.EventId,
		&tx.PaymentStatus,
		&tx.OrderId,
		&tx.PaymentId,
		&tx.Signature,
		&tx.Fees,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error fetching transaction by order_id: %w", err)
	}

	return &tx, nil
}

// this function will update the event transaction status
func UpdateEventTransaction(tx *EventTransaction) error {
	query := `
		UPDATE event_transactions
		SET razor_payment_id = $1,
		    signature = $2,
		    payment_status = $3,
		    updated_at = CURRENT_TIMESTAMP
		WHERE razor_order_id = $4
	`

	_, err := database.DB.Exec(query,
		tx.PaymentId,
		tx.Signature,
		tx.PaymentStatus,
		tx.OrderId,
	)

	if err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}

	return nil
}

func ExpireOldPendingTransactions() {
	// fmt.Println("cron job executed")
	query := `
		UPDATE event_transactions
		SET payment_status = 'Expired', updated_at = NOW()
		WHERE payment_status = 'Pending'
		AND created_at <= NOW() - INTERVAL '15 minutes'
	`

	result, err := database.DB.Exec(query)
	if err != nil {
		log.Println("Failed to expire old pending transactions:", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Expired %d old pending transactions.\n", rowsAffected)
}
