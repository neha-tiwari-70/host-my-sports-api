package migrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sports-events-api/database"
	"sports-events-api/database/seeders"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/sahilm/fuzzy"
)

func MigrateNewLocationData() {
	//NOTE - use transacion for the flow so we can roll back if there is an error
	tx, _ := database.DB.Begin()

	// rename current states and cities table as old_*table_name*
	if err := RenameOldTables(tx, "states"); err != nil {
		tx.Rollback()
		panic(fmt.Errorf("error renaming states table-->%v", err))
	}
	if err := RenameOldTables(tx, "cities"); err != nil {
		tx.Rollback()
		panic(fmt.Errorf("error renaming cities table-->%v", err))
	}

	// Add State codes in the old db in states table if not exists
	if err := AddStateCodeColumn(tx); err != nil {
		tx.Rollback()
		panic(fmt.Errorf("error adding state code column table-->%v", err))
	}

	// Create new 'states' and 'cities' table and dump the data from
	// https://github.com/dr5hn/countries-states-cities-database/tree/master/csv to the states and cities table

	if err := seeders.StatesSeeder(tx); err != nil {
		panic(fmt.Errorf("error seeding states table-->%v", err))
	}
	if err := seeders.CitiesSeeder(tx); err != nil {
		panic(fmt.Errorf("error seeding cities table-->%v", err))
	}
	// Add new_city_id column in the old_cities table if not exists
	if err := AddNewCityIdColumn(tx); err != nil {
		tx.Rollback()
		panic(fmt.Errorf("error adding new_city_id column table-->%v", err))
	}

	//ensure that the state_id in cities table corrospond to the id using state_code
	if err := SetStateIds(tx); err != nil {
		tx.Rollback()
		panic(fmt.Errorf("error setting state ids-->%v", err))
	}

	// update dependencies in events table
	if err := UpdateLocationDependenciesInEvents(tx); err != nil {
		tx.Rollback()
		panic(fmt.Errorf("error setting state ids-->%v", err))
	}

	// update dependencies in user_details table
	if err := UpdateLocationDependenciesInUserDetails(tx); err != nil {
		tx.Rollback()
		panic(fmt.Errorf("error setting state ids-->%v", err))
	}

	tx.Commit()
}

func RenameOldTables(tx *sql.Tx, TableName string) error {
	//drop old location table
	dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS old_%v", TableName)
	_, err := tx.Exec(dropQuery)
	if err != nil {
		return fmt.Errorf("error dropping old_%v table-->%v", TableName, err)
	}

	query := fmt.Sprintf("ALTER TABLE %v RENAME TO old_%v;", TableName, TableName)
	_, err = tx.Exec(query)
	if err != nil {
		return err
	}
	fmt.Printf("\nold %v table renamed\n", TableName)
	return nil
}
func AddStateCodeColumn(tx *sql.Tx) error {
	query := `
		SELECT EXISTS (
		SELECT 1
		FROM information_schema.columns
		WHERE table_name = 'old_states'
		AND column_name = 'state_code');
		
	`
	var exists bool
	err := tx.QueryRow(query).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("state code column alredy exists")
		return nil
	}
	query = `
		ALTER TABLE old_states
		ADD COLUMN IF NOT EXISTS state_code VARCHAR(2)
	`
	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	err = AssignCode(tx)
	if err != nil {
		return err
	}
	fmt.Printf("\nstate code column added\n\n")
	return nil
}
func AddNewCityIdColumn(tx *sql.Tx) error {
	query := `
		ALTER TABLE old_cities
		ADD COLUMN IF NOT EXISTS new_city_id INTEGER
	`
	_, err := tx.Exec(query)
	if err != nil {
		return err
	}

	err = AssignNewCityIds(tx)
	if err != nil {
		return err
	}

	fmt.Println("\nnew_city_id column added")
	return nil
}

type cityData struct {
	City       string `json:"city"`
	Id         int64  `json:"id"`
	FuzzyScore int64  `json:"-"`
}

func AssignNewCityIds(tx *sql.Tx) error {
	var oldCityData []cityData
	query := `
		UPDATE old_cities
		SET new_city_id = nc.id
		FROM cities nc
		WHERE nc.city = old_cities.city;

	`
	_, err := tx.Exec(query)
	if err != nil {
		return err
	}
	//get mismatched oldcities
	query = `
		SELECT JSON_AGG(
			JSON_BUILD_OBJECT(
				'id',id,
				'city',city
			)
		)FROM old_cities
		WHERE new_city_id IS NULL
	`
	var jsonData sql.NullString
	err = tx.QueryRow(query).Scan(&jsonData)
	if err != nil {
		return fmt.Errorf("error fetching mismatched old cities-->%v", err)
	}

	if jsonData.Valid {
		err = json.Unmarshal([]byte(jsonData.String), &oldCityData)
		if err != nil {
			return fmt.Errorf("unmarshal error-->%v", err)
		}
	} else {
		return nil
	}
	// tempOldCities := oldCityData
	//
	var newCitiesMap map[string][]cityData
	var flatNewCityArray []string

	var jsonObjAgg sql.NullString
	var jsonAgg sql.NullString

	query = `
	SELECT JSON_OBJECT_AGG(c.city, NULL), JSON_AGG(c.city) 
		FROM cities c
		LEFT JOIN old_cities oc on oc.new_city_id= c.id
		where oc.id IS NULL
	`

	err = tx.QueryRow(query).Scan(&jsonObjAgg, &jsonAgg)
	if err != nil {
		return fmt.Errorf("error fetching new cities --> %v", err)
	}

	// Unmarshal the object into map
	if jsonObjAgg.Valid {
		err = json.Unmarshal([]byte(jsonObjAgg.String), &newCitiesMap)
		if err != nil {
			return fmt.Errorf("unmarshal map error --> %v", err)
		}
	}

	// Unmarshal the array into flatNewCityArray
	if jsonAgg.Valid {
		err = json.Unmarshal([]byte(jsonAgg.String), &flatNewCityArray)
		if err != nil {
			return fmt.Errorf("unmarshal array error --> %v", err)
		}
	}

	for _, data := range oldCityData {
		targetString := RemoveDistrictSuffix(data.City)
		//squential check
		matches := fuzzy.Find(data.City, flatNewCityArray)
		if len(matches) == 0 {
			matches = fuzzy.Find(targetString, flatNewCityArray)
		}

		var bestMatch string
		var bestScore int
		const maxDistance = 3 // You can tweak this based on your fuzziness tolerance

		//levenshtien check
		for _, newCity := range flatNewCityArray {
			newCityTargetString := RemoveDistrictSuffix(newCity)
			// dist := levenshtein.ComputeDistance(data.City, newCity)
			dist := levenshtein.ComputeDistance(targetString, newCityTargetString)
			if bestMatch == "" || dist < bestScore {
				bestMatch = newCity
				bestScore = dist
			}
		}
		data := cityData{
			Id:   data.Id,
			City: data.City,
		}
		// fmt.Print(matches[0].)
		if len(matches) > 0 {
			newCitiesMap[matches[0].Str] = append(newCitiesMap[matches[0].Str], data)
		} else if bestScore <= maxDistance {
			newCitiesMap[bestMatch] = append(newCitiesMap[bestMatch], data)
		}

	}
	filteredMap := make(map[string][]cityData)

	for key, val := range newCitiesMap {
		if len(val) > 0 {
			allNil := true
			for _, v := range val {
				if v != (cityData{}) {
					allNil = false
					break
				}
			}
			if !allNil {
				filteredMap[key] = val
			}
		}
	}

	//update new_city_id

	for key, val := range filteredMap {
		updateQuery := `
			UPDATE old_cities
			SET new_city_id = nc.id
			FROM cities nc
			WHERE nc.city = $1
			AND old_cities.id= $2; 
		`
		_, err := tx.Exec(updateQuery, key, val[0].Id)
		if err != nil {
			return err
		}
	}

	countQuery := `
		SELECT 
			(SELECT COUNT(new_city_id) FROM old_cities) AS updated,
			(SELECT COUNT(*) FROM old_cities) AS total;
	`

	var updated, totalCount int64
	err = tx.QueryRow(countQuery).Scan(&updated, &totalCount)
	if err != nil {
		return err
	}
	fmt.Printf("\n%v / %v old_cities paired with new cities\n", updated, totalCount)
	return nil
}

func RemoveDistrictSuffix(s string) string {
	if strings.HasSuffix(s, " District") {
		return strings.TrimSuffix(s, " District")
	} else if strings.HasSuffix(s, " district") {
		return strings.TrimSuffix(s, " district")
	} else if strings.HasSuffix(s, " City") {
		return strings.TrimSuffix(s, " City")
	} else if strings.HasSuffix(s, " city") {
		return strings.TrimSuffix(s, " city")
	} else if strings.HasSuffix(s, " Nagar") {
		return strings.TrimSuffix(s, " Nagar")
	} else if strings.HasSuffix(s, " nagar") {
		return strings.TrimSuffix(s, " nagar")
	}
	return s
}

func AssignCode(tx *sql.Tx) error {
	StateWiseCode := map[string]string{
		"Andaman & Nicobar": "AN",
		"Andhra Pradesh":    "AP",
		"Arunachal Pradesh": "AR",
		"Assam":             "AS",
		"Bihar":             "BR",
		"Chandigrah":        "CH",
		"Chattisgarh":       "CT",
		"Dadra & Nagar":     "DH",
		"Daman & Diu":       "DH",
		"Delhi":             "DL",
		"Goa":               "GA",
		"Gujarat":           "GJ",
		"Haryana":           "HR",
		"Himachal Pradesh":  "HP",
		"Jammu & Kashmir":   "JK",
		"Jharkhand":         "JH",
		"Karnataka":         "KA",
		"Kerala":            "KL",
		// :"LA"
		"Lakshwadeep":    "LD",
		"Madhya Pradesh": "MP",
		"Maharashtra":    "MH",
		"Manipur":        "MN",
		"Meghalaya":      "ML",
		"Mizoram":        "MZ",
		"Nagaland":       "NL",
		"Orissa":         "OR",
		"Pondicherry":    "PY",
		"Punjab":         "PB",
		"Rajasthan":      "RJ",
		"Sikkim":         "SK",
		"Tamil Nadu":     "TN",
		// :"TG"
		"Tripura":       "TR",
		"Uttar Pradesh": "UP",
		"Uttranchal":    "UK",
		"West Bengal":   "WB",
	}

	for state := range StateWiseCode {
		query := `
			UPDATE old_states SET state_code=$1
			WHERE name=$2
		`
		if _, err := tx.Exec(query, StateWiseCode[state], state); err != nil {
			return err
		}
	}
	return nil
}

func SetStateIds(tx *sql.Tx) error {
	query := `
		UPDATE cities c
		SET state_id = s.id
		FROM states s
		WHERE s.state_code = c.state_code;
	`
	_, err := tx.Exec(query)
	if err != nil {
		return err
	}
	fmt.Println("\nState ids updated in cities table")
	return nil
}

func UpdateLocationDependenciesInEvents(tx *sql.Tx) error {
	query := `
		WITH updated AS (
		UPDATE events e
		SET city_id = c.id,
			state_id = s.id,
			updated_at = NOW()
		FROM cities c
		JOIN states s ON s.state_code = c.state_code
		JOIN old_cities oc ON c.id = oc.new_city_id
		WHERE oc.id = e.city_id::INTEGER
		RETURNING e.id
		)
		SELECT 
		(SELECT COUNT(*) FROM updated) AS updated_count,
		(SELECT COUNT(*) FROM events) AS total_count;
	`

	var updated, totalCount int64
	err := tx.QueryRow(query).Scan(&updated, &totalCount)
	if err != nil {
		return err
	}
	fmt.Printf("\n%v / %v events updated\n", updated, totalCount)
	return nil
}

func UpdateLocationDependenciesInUserDetails(tx *sql.Tx) error {
	query := `
		WITH updated AS (
			UPDATE user_details ud
			SET city = c.id,
				state = s.id,
				updated_at = NOW()
			FROM cities c
			JOIN states s ON s.state_code = c.state_code
			JOIN old_cities oc ON c.id = oc.new_city_id
			WHERE oc.id = ud.city
			RETURNING ud.id
		)
		SELECT 
		(SELECT COUNT(*) FROM updated) AS updated_count,
		(SELECT COUNT(*) FROM user_details) AS total_count;
	`

	var updated, totalCount int64
	err := tx.QueryRow(query).Scan(&updated, &totalCount)
	if err != nil {
		return err
	}
	fmt.Printf("\n%v / %v user_details table entries updated\n", updated, totalCount)
	return nil
}
