package controllers

import (
	"database/sql"
	"fmt"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

func GetParticipantsData(c *gin.Context) {

	// Step 1: Decrypt all the encrypted route parameters

	eventId := DecryptParamId(c, "event_id", true)
	if eventId == 0 {
		return
	}
	gameId := DecryptParamId(c, "game_id", true)
	if gameId == 0 {
		return
	}
	gameTypeId := DecryptParamId(c, "game_type_id", true)
	if gameTypeId == 0 {
		return
	}
	ageGroupId := DecryptParamId(c, "age_group_id", true)
	if ageGroupId == 0 {
		return
	}

	// Step 2: Fetch event, game, game type, age group names
	var eventName, gameName, gameTypeName, ageGroupName string

	err := database.DB.QueryRow("SELECT name FROM events WHERE id = $1", eventId).Scan(&eventName)
	if err != nil {
		utils.HandleError(c, "Failed to fetch the event name", err)
		return
	}

	err = database.DB.QueryRow("SELECT game_name FROM games WHERE id = $1", gameId).Scan(&gameName)
	if err != nil {
		utils.HandleError(c, "Failed to fetch the game name", err)
		return
	}

	err = database.DB.QueryRow("SELECT name FROM games_types WHERE id = $1", gameTypeId).Scan(&gameTypeName)
	if err != nil {
		utils.HandleError(c, "Failed to fetch the game type  name", err)
		return
	}

	err = database.DB.QueryRow("SELECT category FROM age_group WHERE id = $1", ageGroupId).Scan(&ageGroupName)
	if err != nil {
		utils.HandleError(c, "Failed to fetch the age group name", err)
		return
	}

	// Step 3: Fetch all teams for this event/game/type/age group
	teamsQuery := `
		SELECT id, team_name,team_captain 
		FROM event_has_teams 
		WHERE event_id = $1 AND game_id = $2 AND game_type_id = $3 AND age_group_id = $4
	`
	rows, err := database.DB.Query(teamsQuery, eventId, gameId, gameTypeId, ageGroupId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch the teams", err)
		return
	}
	defer rows.Close()

	type User struct {
		EncId    string  `json:"user_id"`
		ID       int64   `json:"-"`
		Name     string  `json:"name"`
		Email    string  `json:"email"`
		MobileNo string  `json:"mobile_no"`
		UserCode string  `json:"user_code"`
		Height   *string `json:"height"`
		Weight   *string `json:"weight"`
		DOB      *string `json:"dob"`
	}

	type Team struct {
		EncId   string `json:"team_id"`
		ID      int64  `json:"-"`
		Name    string `json:"team_name"`
		Captain string `json:"captain_name"`
		Users   []User `json:"users"`
	}

	var teams []Team

	for rows.Next() {
		var teamId int64
		var teamName string
		var teamCaptainId int64

		err := rows.Scan(&teamId, &teamName, &teamCaptainId)
		if err != nil {
			continue // Skip faulty row
		}

		// Step 4: Fetch users for this team
		usersQuery := `
			SELECT 
				u.id, u.name, u.email, u.mobile_no, u.user_code,
				ud.height, ud.weight, ud.dob
			FROM event_has_users ehu 
			JOIN users u ON ehu.user_id = u.id 
			LEFT JOIN user_details ud ON ud.user_id = u.id
			WHERE ehu.event_has_team_id = $1

		`
		userRows, err := database.DB.Query(usersQuery, teamId)
		if err != nil {
			continue
		}

		var users []User
		for userRows.Next() {
			var u User
			var dob sql.NullTime
			var height, weight sql.NullString

			err := userRows.Scan(
				&u.ID, &u.Name, &u.Email, &u.MobileNo, &u.UserCode,
				&height, &weight, &dob,
			)
			if err != nil {
				userRows.Close()
				utils.HandleError(c, "Failed to parse user row for team ID "+fmt.Sprint(teamId), err)
				return
			}

			if height.Valid {
				u.Height = &height.String
			}
			if weight.Valid {
				u.Weight = &weight.String
			}
			if dob.Valid {
				dobStr := dob.Time.Format("2006-01-02") // or your preferred format
				u.DOB = &dobStr
			}

			u.EncId = crypto.NEncrypt(u.ID)
			users = append(users, u)
		}

		userRows.Close()

		// Step 5: Get captain name
		var captainName string
		err = database.DB.QueryRow("SELECT name FROM users WHERE id = $1", teamCaptainId).Scan(&captainName)
		if err != nil {
			utils.HandleError(c, "Failed to fetch the captain name for team ID "+fmt.Sprint(teamId), err)
			return
		}

		teams = append(teams, Team{
			EncId:   crypto.NEncrypt(teamId),
			Name:    teamName,
			Captain: captainName,
			Users:   users,
		})
	}

	utils.HandleSuccess(c, "Participants fetched successfully", gin.H{
		"event_name":     eventName,
		"game_name":      gameName,
		"game_type_name": gameTypeName,
		"age_group_name": ageGroupName,
		"teams":          teams,
	})

}
