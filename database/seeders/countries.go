package seeders

import (
	"fmt"
	"sports-events-api/database"

	"github.com/qustavo/dotsql"
)

func CountriesSeeder() {
	dot, err := dotsql.LoadFromFile("database/seeders/country.sql")
	if err != nil {
		panic(err)
	}

	if _, err := dot.Exec(database.DB, "create-countries-table"); err != nil {
		panic(err)
	}

	if _, err := dot.Exec(database.DB, "insert-countries"); err != nil {
		panic(err)
	}

	fmt.Println("Countries table seeded successfully.")
}
