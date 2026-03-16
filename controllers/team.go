package controllers

import (
	"fmt"
	"math/rand"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetAllTeams retrieves a list of teams based on various filters such as pagination, search, sorting, status, and associated event/game information.
// This function performs the following:
// 1. Extracts query parameters for pagination, search, sorting, and status.
// 2. Decrypts the event_id, game_id, and game_type_id to ensure privacy in the response.
// 3. Retrieves teams based on the provided filters and conditions (status, search, etc.).
// 4. Encrypts the necessary fields (e.g., team ID and team captain ID) for security.
// 5. Responds with the fetched teams and total count for pagination purposes.
//
// Params:
//   - c (*gin.Context): The Gin context to handle the HTTP request and response.
//
// Returns:
//   - error: If any error occurs during the process (e.g., database fetch failure).
func GetAllTeams(c *gin.Context) {
	// Step 1: Extract query parameters for pagination, search, sorting, and status.
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))    // Default page is 1
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10")) // Default limit is 10
	search := c.Query("search")                             // Search filter for team names or other fields
	sort := c.DefaultQuery("sort", "created_at")            // Sorting field, default is 'created_at'
	dir := c.DefaultQuery("dir", "DESC")                    // Sorting direction, default is 'DESC'
	status := c.Query("status")                             // Team status filter (e.g., active, inactive)
	offset := (page - 1) * limit                            // Calculate the offset for pagination
	event_id := c.Query("event_id")                         // Event ID filter (encrypted)
	game_id := c.Query("game_id")                           // Game ID filter (encrypted)
	game_type_id := c.Query("game_type_id")                 // Game Type ID filter (encrypted)
	age_group_id := c.Query("age_group_id")

	// Step 2: Decrypt event_id, game_id, and game_type_id for secure use
	ev_id, _ := crypto.NDecrypt(event_id)
	gm_id, _ := crypto.NDecrypt(game_id)
	gm_tp_id, _ := crypto.NDecrypt(game_type_id)
	age_grp_id, _ := crypto.NDecrypt(age_group_id)
	// fmt.Println(ev_id, gm_id, gm_tp_id, age_grp_id)
	eventHasGameTypeId, err := models.GetEventHasGameTypeId(ev_id, gm_id, gm_tp_id, age_grp_id)
	if err != nil {
		utils.HandleError(c, err.Error())
		return
	}
	// fmt.Println("Event has game type id : ", eventHasGameTypeId)
	game, err := models.GetGamesById(gm_id)
	if err != nil {
		utils.HandleError(c, "Error getting game name.")
		return
	}
	// fmt.Println("Game name : ", game.Name)
	tournamentType, err := models.GetTournamentTypeByEventAndGame(int64(ev_id), int64(gm_id))
	if err != nil {
		utils.HandleError(c, err.Error())
		return
	}

	isMatchAlreadyExist, err := models.IsMatchAlreadyScheduled(eventHasGameTypeId)
	if err != nil {
		utils.HandleError(c, err.Error())
	}
	if isMatchAlreadyExist {
		latestRound, err := models.GetLatestRound(eventHasGameTypeId)
		if err != nil {
			utils.HandleError(c, err.Error())
			return
		}
		latestMatches, err := models.FetchLatestMatchesWithTeams(eventHasGameTypeId, latestRound)
		if err != nil {
			utils.HandleError(c, "Failed to fecth the latest matches", err)
			return
		}
		if tournamentType == "Atheletics" || tournamentType == "Time Trial" || tournamentType == "Mass Start" || tournamentType == "Relay" || tournamentType == "Fun Ride" || tournamentType == "Endurance" {
			// Flag to detect if at least one match has IsDraw set
			hasAnyDrawValue := false
			for _, match := range latestMatches {
				if match.IsDraw != nil {
					hasAnyDrawValue = true
					break
				}
			}

			if hasAnyDrawValue {
				// Fetch points because at least one match has IsDraw != nil
				teamPoints, err := models.GetPointTable(ev_id, gm_id, []int64{gm_tp_id}, []int64{age_grp_id})
				if err != nil {
					utils.HandleError(c, "Error fetching point table", err)
					return
				}

				utils.HandleSuccess(c, "Matches and points fetched successfully", gin.H{
					"is_match_already_exist": isMatchAlreadyExist,
					"round_no":               latestRound,
					"matches":                latestMatches,
					"tournament_type":        tournamentType,
					"topTeams":               teamPoints,
					"is_last_round":          false,
				})
			} else {
				// No draw values set → return only matches
				utils.HandleSuccess(c, "Matches fetched successfully", gin.H{
					"is_match_already_exist": isMatchAlreadyExist,
					"round_no":               latestRound,
					"matches":                latestMatches,
					"tournament_type":        tournamentType,
					"is_last_round":          false,
				})
			}
			return
		}
		isPointsNull, err := models.CheckForNullPoints(latestMatches)
		if err != nil {
			utils.HandleError(c, err.Error())
			return
		}

		if tournamentType == "League" && !isPointsNull {
			if game.Name == "Chess" {
				// 1. Extract unique team IDs from latestMatches
				teamIDMap := make(map[int64]bool)
				for _, match := range latestMatches {
					if len(match.TeamsArray) > 0 {
						teamIDMap[match.TeamsArray[0].ID] = true
					}
					if len(match.TeamsArray) > 1 {
						teamIDMap[match.TeamsArray[1].ID] = true
					}
				}

				teamIDs := make([]int64, 0, len(teamIDMap))
				for id := range teamIDMap {
					teamIDs = append(teamIDs, id)
				}
				// fmt.Println("Team ids : ", teamIDs)
				// 2. Get total points for each team from previous rounds
				teamPointsMap := models.GetTeamPointsFromPreviousRounds(eventHasGameTypeId, teamIDs)

				// 3. Get basic team details
				teams, err := models.GetTeamsByIDs(teamIDs)
				if err != nil {
					utils.HandleError(c, "Error fetching teams for Chess system")
					return
				}
				isLastRound, err := models.GetLastRound(eventHasGameTypeId)
				if err != nil {
					fmt.Println("Error getting last round : ", err)
				}
				// fmt.Println("is last round ", isLastRound)
				// fmt.Println("Teams details came  : ", teams)
				// 4. Build topTeams array with total_points
				topTeams := make([]map[string]interface{}, 0)
				for _, team := range teams {
					encTeamID := crypto.NEncrypt(team.ID)
					points := teamPointsMap[team.ID]

					topTeams = append(topTeams, map[string]interface{}{
						"id":             encTeamID,
						"name":           team.TeamName,
						"team_logo_path": team.TeamLogo,
						"total_points":   points,
						// Add other fields if needed like "slug", "created_at", etc.
					})
				}
				// 5. Return final response
				utils.HandleSuccess(c, "Matches generated successfully", gin.H{
					"is_match_already_exist": isMatchAlreadyExist,
					"round_no":               latestRound,
					"matches":                latestMatches,
					"tournament_type":        tournamentType,
					"topTeams":               topTeams,
					"is_last_round":          isLastRound,
				})
				return
			} else {
				topTeams, err := models.CalculatePointsForLeague(latestMatches)
				if err != nil {
					utils.HandleError(c, err.Error())
					return
				}

				utils.HandleSuccess(c, "Matches generated successfully", gin.H{
					"is_match_already_exist": isMatchAlreadyExist,
					"round_no":               latestRound,
					"matches":                latestMatches,
					"tournament_type":        tournamentType,
					"topTeams":               topTeams,
					"is_last_round":          false,
				})
			}
		} else if tournamentType == "League cum knockout" {
			if latestRound == 1 && !isPointsNull {
				topTeams, err := models.CalculateLeaguePointsForGroupedTeams(latestMatches)
				if err != nil {
					utils.HandleError(c, err.Error())
					return
				}

				utils.HandleSuccess(c, "Matches generated successfully", gin.H{
					"is_match_already_exist": isMatchAlreadyExist,
					"round_no":               latestRound,
					"matches":                latestMatches,
					"tournament_type":        tournamentType,
					"topTeams":               topTeams,
					"is_last_round":          false,
				})
			} else if latestRound > 1 {
				utils.HandleSuccess(c, "Matches generated successfully", gin.H{
					"is_match_already_exist": isMatchAlreadyExist,
					"round_no":               latestRound,
					"matches":                latestMatches,
					"tournament_type":        tournamentType,
					"is_last_round":          false,
				})
			} else {
				utils.HandleSuccess(c, "Matches generated successfully", gin.H{
					"is_match_already_exist": isMatchAlreadyExist,
					"round_no":               latestRound,
					"matches":                latestMatches,
					"tournament_type":        tournamentType,
					"is_last_round":          false,
				})
			}
		} else {
			utils.HandleSuccess(c, "Matches generated successfully", gin.H{
				"is_match_already_exist": isMatchAlreadyExist,
				"round_no":               latestRound,
				"matches":                latestMatches,
				"tournament_type":        tournamentType,
				"is_last_round":          false,
			})
		}
	} else {
		totalRecords, Teams, err := models.GetTeams(search, sort, dir, status, int64(ev_id), int64(gm_id), int64(gm_tp_id), int64(age_grp_id), int64(limit), int64(offset))
		if err != nil {
			// Step 4: Return error if the teams could not be fetched
			c.JSON(http.StatusOK, gin.H{
				"status":  "error",
				"data":    err,
				"message": "Failed to fetch teams.",
			})
			return
		}

		// Step 5: Prepare the list of teams with necessary data
		encryptedTeams := make([]map[string]interface{}, 0)
		for _, team := range Teams {
			// Step 6: Set the logo path for the team, use default if unavailable
			teamLogoPath := team.TeamLogoPath
			defaultLogoPath := "public/static/staticTeamLogo.png"
			if team.TeamLogoPath == "" || !fileExists(teamLogoPath) {
				teamLogoPath = defaultLogoPath
			}

			// Step 7: Encrypt sensitive data such as team ID and captain ID
			encryptedId := crypto.NEncrypt(team.TeamId)
			team.TeamCaptainEncId = crypto.NEncrypt(team.TeamCaptainID)
			// fmt.Println("Team id : ", team.TeamId)
			// Step 8: Append the encrypted team details to the list
			encryptedTeams = append(encryptedTeams, map[string]interface{}{
				"id":                encryptedId,
				"name":              team.TeamName,
				"team_captain":      team.TeamCaptainEncId,
				"team_captain_name": team.TeamCaptain,
				"team_logo_path":    teamLogoPath,
				"slug":              team.Slug,
				"status":            team.Status,
				"group_no":          team.GroupNo,
				"created_at":        team.CreatedAt,
				"updated_at":        team.UpdatedAt,
			})

			// Step 9: Return the paginated and filtered list of teams along with the total record count

		}

		tournamentType, err := models.GetTournamentTypeByEventAndGame(int64(ev_id), int64(gm_id))
		if err != nil {
			utils.HandleError(c, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": gin.H{
				"totalRecords":           totalRecords,
				"is_match_already_exist": isMatchAlreadyExist,
				"round_no":               0,
				"teams":                  encryptedTeams,
				"is_last_round":          false,
				"tournament_type":        tournamentType,
			},
			"message": "Fetched all teams successfully.",
		})
	}

}

func SetIsLastRound(c *gin.Context) {

	decEventId := DecryptParamId(c, "event_id", true)
	if decEventId == 0 {
		return
	}
	decGameId := DecryptParamId(c, "game_id", true)
	if decGameId == 0 {
		return
	}
	decGameTypeId := DecryptParamId(c, "game_type_id", true)
	if decGameTypeId == 0 {
		return
	}
	decAgeGroupId := DecryptParamId(c, "age_group_id", true)
	if decAgeGroupId == 0 {
		return
	}

	eventHasGameTypeId, err := models.GetEventHasGameTypeId(decEventId, decGameId, decGameTypeId, decAgeGroupId)
	if err != nil {
		utils.HandleError(c, "Unable to get event has game type id ", err)
		return
	}
	// fmt.Println("Id is : ", eventHasGameTypeId)
	success, err := models.SetIsLastRound(eventHasGameTypeId)
	if err != nil {
		utils.HandleError(c, "Unable to final round", err)
		return
	}

	if success {
		utils.HandleSuccess(c, "Tournament closed successfully", success)
		return
	}

	utils.HandleError(c, "Failed to close Tournament")
}

func MakeTeamGroups(c *gin.Context) {
	var req struct {
		TeamsPerGroup int      `json:"teams_per_group" binding:"required"`
		TeamIDs       []string `json:"team_ids" binding:"required"` // Encrypted team IDs
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}

	if req.TeamsPerGroup < 2 || len(req.TeamIDs) < req.TeamsPerGroup {
		utils.HandleInvalidEntries(c, "Insufficient teams for given group size")
		return
	}

	// Decrypt team IDs using your DecryptID function
	var teamIDs []int64
	for _, encryptedID := range req.TeamIDs {
		id, err := crypto.NDecrypt(encryptedID)
		if err != nil {
			utils.HandleError(c, "Failed to decrypt id : ", err)
			return
		}
		teamIDs = append(teamIDs, id)
	}

	// Shuffle team IDs
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(teamIDs), func(i, j int) {
		teamIDs[i], teamIDs[j] = teamIDs[j], teamIDs[i]
	})

	type assignment struct {
		TeamID  int64
		GroupNo int
	}
	assignments := []assignment{}

	groupNo := 1
	i := 0
	for i < len(teamIDs) {
		remaining := len(teamIDs) - i

		if remaining == 1 {
			assignments = append(assignments, assignment{
				TeamID:  teamIDs[i],
				GroupNo: 1,
			})
			break
		}

		end := i + req.TeamsPerGroup
		if end > len(teamIDs) {
			end = len(teamIDs)
		}

		for _, teamID := range teamIDs[i:end] {
			assignments = append(assignments, assignment{
				TeamID:  teamID,
				GroupNo: groupNo,
			})
		}

		groupNo++
		i = end
	}

	for _, a := range assignments {
		_, err := database.DB.Exec(`
			UPDATE event_has_teams
			SET group_no = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, a.GroupNo, a.TeamID)

		if err != nil {
			utils.HandleError(c, "Failed to assign group", err)
			return
		}
	}

	utils.HandleSuccess(c, "Groups assigned successfully", gin.H{
		// "assignments":  assignments,
		"total_groups": groupNo - 1,
	})
}

func VerifyTeamName(c *gin.Context) {
	var payload struct {
		Name    string `json:"team_name"`
		EventId string `json:"event_id"`
		GameId  string `json:"game_id"`
		TeamId  string `json:"team_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.HandleError(c, "Invalid input", err)
		return
	}
	DecEventId, err := crypto.NDecrypt(payload.EventId)
	if err != nil {
		utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting event_has_game_type_id(value:'%v')->%v", DecEventId, err))
		return
	}
	DecGameId, err := crypto.NDecrypt(payload.GameId)
	if err != nil {
		utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting game_id(value:'%v')->%v", DecGameId, err))
		return
	}
	var DecTeamId int64
	if payload.TeamId != "" {
		DecTeamId, err = crypto.NDecrypt(payload.TeamId)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting team_id(value:'%v')->%v", DecTeamId, err))
			return
		}
	}

	isTeamNameAvalable, err := models.VerifyTeamName(payload.Name, DecEventId, DecGameId, DecTeamId)
	if err != nil {
		utils.HandleError(c, "Error verifying team name", err)
		return
	}
	utils.HandleSuccess(c, "team name verified successfull", isTeamNameAvalable)
}
