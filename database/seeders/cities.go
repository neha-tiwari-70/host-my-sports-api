package seeders

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"sports-events-api/database"
	"strconv"
)

func CitiesSeeder(txArr ...*sql.Tx) error {
	var tx *sql.Tx
	if len(txArr) < 1 {
		tx, _ = database.DB.Begin()
	} else {
		tx = txArr[0]
	}
	query := `
		CREATE TABLE IF NOT EXISTS cities (
			id SERIAL PRIMARY KEY,
			city VARCHAR(255) NOT NULL,
			state_id INT NOT NULL ,
			state_code VARCHAR(10) NOT NULL
		);
	`
	if _, err := tx.Exec(query); err != nil {
		tx.Rollback()
		return err
	}

	if err := insertCities(tx); err != nil {
		tx.Rollback()
		return err
	}
	fmt.Println("Cities table seeded successfully.")
	if len(txArr) < 1 {
		tx.Commit()
	}
	return nil
}

func insertCities(tx *sql.Tx) error {
	// Open CSV file
	file, err := os.Open("database/seeders/cities_data.csv")
	if err != nil {
		return fmt.Errorf("error opening csv:%v", err)
	}
	defer file.Close()

	// Read CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading csv:%v", err)
	}
	query := `
		INSERT INTO cities (city, state_id, state_code) VALUES`

	// Skip header and insert rows
	for i, row := range records {
		if i == 0 {
			continue // skip header
		}

		city := row[0]
		state_id, _ := strconv.Atoi(row[1])
		state_code := row[2]

		query += fmt.Sprintf("\n('%v', %v, '%v')", city, state_id, state_code)
		if i == len(records)-1 {
			query += ";"
		} else {
			query += ","
		}
	}

	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	fmt.Println("CSV data inserted successfully.")
	return nil
}
