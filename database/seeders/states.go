package seeders

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"

	"github.com/qustavo/dotsql"
)

func StatesSeeder(txArr ...*sql.Tx) error {
	var tx *sql.Tx
	if len(txArr) < 1 {
		tx, _ = database.DB.Begin()
	} else {
		tx = txArr[0]
	}
	dot, err := dotsql.LoadFromFile("database/seeders/state.sql")
	if err != nil {
		tx.Rollback()
		return err
	}

	if _, err := dot.Exec(tx, "create-states-table"); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := dot.Exec(tx, "insert-states"); err != nil {
		tx.Rollback()
		return err
	}

	fmt.Println("States table seeded successfully.")
	if len(txArr) < 1 {
		tx.Commit()
	}
	return nil
}
