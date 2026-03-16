package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/crypto"
	"sports-events-api/database"
)

type Country struct {
	EncID string `json:"id"`
	ID    int64  `json:"-"`
	Name  string `json:"name"`
	Code  string `json:"code"`
}
type State struct {
	EncID     string `json:"id"`
	ID        int64  `json:"-"`
	Name      string `json:"name"`
	CountryId string `json:"country_id"`
}
type City struct {
	EncID   string `json:"id"`
	ID      int64  `json:"-"`
	Name    string `json:"name"`
	StateId string `json:"state_id"`
}

// func GetCityById(id int64) (*City, error) {
// 	query := `SELECT city FROM cities WHERE id=$1`
// 	var city City
// 	err := database.DB.QueryRow(query, id).Scan(
// 		&city.Name,
// 	)
// 	if err == sql.ErrNoRows {
// 		return nil, fmt.Errorf("city is not found")
// 	} else if err != nil {
// 		return nil, fmt.Errorf("error fetching city : %v", err)
// 	}
// 	return &city, nil
// }

//	func GetStateById(id int64) (*State, error) {
//		query := `SELECT name FROM states WHERE id=$1`
//		var state State
//		err := database.DB.QueryRow(query, id).Scan(
//			&state.Name,
//		)
//		if err == sql.ErrNoRows {
//			return nil, fmt.Errorf("state is not found")
//		} else if err != nil {
//			return nil, fmt.Errorf("error fetching state : %v", err)
//		}
//		return &state, nil
//	}
func GetCityById(id int64) (*City, error) {
	query := `SELECT id, city FROM cities WHERE id=$1`
	var city City
	err := database.DB.QueryRow(query, id).Scan(&city.ID, &city.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("city not found")
		}
		return nil, fmt.Errorf("error fetching city: %v", err)
	}
	city.EncID = crypto.NEncrypt(city.ID)
	return &city, nil
}

func GetStateById(id int64) (*State, error) {
	query := `SELECT id, name FROM states WHERE id=$1`
	var state State
	err := database.DB.QueryRow(query, id).Scan(&state.ID, &state.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("state not found")
		}
		return nil, fmt.Errorf("error fetching state: %v", err)
	}
	state.EncID = crypto.NEncrypt(state.ID)
	return &state, nil
}

//	func GetAllCountries() (*sql.Rows, error) {
//		query := "SELECT id, name, countrycode FROM countries"
//		Rows, err := database.DB.Query(query)
//		if err != nil {
//			return nil, err
//		}
//		return Rows, nil
//	}
func GetAllCountries() ([]Country, error) {
	query := "SELECT id, name, countrycode FROM countries"
	Rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer Rows.Close()

	var countries []Country
	for Rows.Next() {
		var item Country
		err := Rows.Scan(&item.ID, &item.Name, &item.Code)
		if err != nil {
			return nil, err
		}
		item.EncID = crypto.NEncrypt(item.ID)
		countries = append(countries, item)
	}
	return countries, nil
}

// func GetStateByCountry(Country_id int64) (*sql.Rows, error) {
// 	query := "SELECT id, name FROM states where country_id=$1"
// 	Rows, err := database.DB.Query(query, Country_id)
// 	if err != nil {
// 		return Rows, err
// 	}
// 	return Rows, nil
// }

func GetStateByCountry(Country_id int64) ([]State, error) {
	query := "SELECT id, name FROM states WHERE country_id=$1"
	Rows, err := database.DB.Query(query, Country_id)
	if err != nil {
		return nil, err
	}
	defer Rows.Close()

	var states []State
	for Rows.Next() {
		var item State
		err := Rows.Scan(&item.ID, &item.Name)
		if err != nil {
			return nil, err
		}
		item.EncID = crypto.NEncrypt(item.ID)
		states = append(states, item)
	}
	return states, nil
}

//	func GetCityByState(State_id int64) (*sql.Rows, error) {
//		query := "SELECT id, city FROM Cities where state_id=$1"
//		Rows, err := database.DB.Query(query, State_id)
//		if err != nil {
//			return Rows, err
//		}
//		return Rows, nil
//	}
func GetCityByState(State_id int64) ([]City, error) {
	query := "SELECT id, city FROM cities WHERE state_id=$1"
	Rows, err := database.DB.Query(query, State_id)
	if err != nil {
		return nil, err
	}
	defer Rows.Close()

	var cities []City
	for Rows.Next() {
		var item City
		err := Rows.Scan(&item.ID, &item.Name)
		if err != nil {
			return nil, err
		}
		item.EncID = crypto.NEncrypt(item.ID)
		cities = append(cities, item)
	}
	return cities, nil
}
