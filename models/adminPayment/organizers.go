package models

import (
	"encoding/json"
	"fmt"
	"sports-events-api/database"
)

type Organizer struct {
	OrganizerID   int64       `json:"organizer_id"`
	OrganizerName string      `json:"organizer_name"`
	Email         string      `json:"email"`
	MobileNo      string      `json:"mobile_no"`
	BankDetails   interface{} `json:"bank_details"`
	Events        interface{} `json:"events"`
}

func GetOrganizers(search, sort, dir string, limit, offset int64) (int64, []Organizer, error) {
	var organizers []Organizer

	// ------------------------------
	// 1️⃣ Total records count
	// ------------------------------
	countArgs := []interface{}{}
	countQuery := `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		INNER JOIN events e ON e.created_by_id = u.id AND e.status = 'Active'`

	if search != "" {
		countQuery += " AND (u.name ILIKE $1 OR u.email ILIKE $2 OR u.mobile_no ILIKE $3)"
		countArgs = append(countArgs, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var totalRecords int64
	if err := database.DB.QueryRow(countQuery, countArgs...).Scan(&totalRecords); err != nil {
		return 0, nil, err
	}

	// ------------------------------
	// 2️⃣ Main query
	// ------------------------------
	queryArgs := []interface{}{limit, offset}
	argIdx := 3 // $1=limit, $2=offset, search starts at $3

	query := `
	SELECT
		u.id AS organizer_id,
		u.name AS organizer_name,
		u.email,
		u.mobile_no,
		json_build_object(
			'upi_id', b.upi_id,
			'qr_code', b.qr_code,
			'account_name', b.account_name,
			'account_no', b.account_no,
			'account_type', b.account_type,
			'branch_name', b.branch_name,
			'ifsc_code', b.ifsc_code
		) AS bank_details,
		json_agg(
			json_build_object(
				'event_id', e.id,
				'event_name', e.name,
				'from_date', e.from_date,
				'to_date', e.to_date,
				'event_fees', e.fees,
				'total_fees_collected', COALESCE(et.total_collected, 0)
			)
		) AS events
	FROM users u
	LEFT JOIN bank_details b ON b.user_id = u.id
	INNER JOIN events e ON e.created_by_id = u.id AND e.status = 'Active'
	LEFT JOIN (
		SELECT event_id, SUM(fees) AS total_collected
		FROM event_transactions
		WHERE payment_status = 'Success'
		GROUP BY event_id
	) et ON et.event_id = e.id`

	if search != "" {
		query += fmt.Sprintf(" AND (u.name ILIKE $%d OR u.email ILIKE $%d OR u.mobile_no ILIKE $%d)", argIdx, argIdx+1, argIdx+2)
		queryArgs = append(queryArgs, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	query += fmt.Sprintf(`
	GROUP BY u.id, u.name, u.email, u.mobile_no, b.upi_id, b.qr_code, b.account_name, b.account_no, b.account_type, b.branch_name, b.ifsc_code
	ORDER BY %s %s LIMIT $1 OFFSET $2`, sort, dir)

	// Execute query
	rows, err := database.DB.Query(query, queryArgs...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var org Organizer
		var bankJSON []byte
		var eventsJSON []byte

		if err := rows.Scan(
			&org.OrganizerID,
			&org.OrganizerName,
			&org.Email,
			&org.MobileNo,
			&bankJSON,
			&eventsJSON,
		); err != nil {
			return 0, nil, err
		}

		if err := json.Unmarshal(bankJSON, &org.BankDetails); err != nil {
			org.BankDetails = nil
		}
		if err := json.Unmarshal(eventsJSON, &org.Events); err != nil {
			org.Events = []interface{}{}
		}

		organizers = append(organizers, org)
	}

	return totalRecords, organizers, nil
}
