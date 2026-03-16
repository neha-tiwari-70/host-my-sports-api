package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ResponseObject is used to return essential team info (encrypted) back to frontend
type ResponseObject struct {
	TeamId       string            `json:"team_id"`
	TypeId       string            `json:"type_id"`
	AgeGroupId   string            `json:"age_group_id"`
	TeamLogoPath string            `json:"team_logo_path"`
	TshirtSize   map[string]string `json:"tshirt_size"`
}

const TeamFolderPath = "public/event/team_logos"

// GetParticipantByUserCode handles user lookup by userCode and checks their eligibility to participate
// in a specific game within an event. It performs the following:
// 1. Extracts encrypted IDs and decrypts them.
// 2. Validates game-event association.
// 3. Fetches the user and calculates age (if required).
// 4. Checks if the user is already registered.
// 5. Validates age eligibility against the game's category requirements.
//
// Route Parameters:
//   - userCode (string): Unique code to identify the user.
//   - eventId (string): Encrypted Event ID.
//   - gameId (string): Encrypted Game ID associated with the event.
//   - teamId (string): Optional Encrypted Team ID for checking if already part of a team.
func GetParticipantByUserCode(c *gin.Context) {
	// Extract and decrypt route parameters
	userCode, exists := c.Params.Get("userCode")
	if !exists {
		utils.HandleError(c, "Invalid request-->missing userCode param")
		return
	}
	EventId := DecryptParamId(c, "eventId", true)
	if EventId == 0 {
		return
	}
	EventGameId := DecryptParamId(c, "gameId", true)
	if EventId == 0 {
		return
	}
	EHGameTypeId := DecryptParamId(c, "ehgtypeId", true)
	if EventId == 0 {
		return
	}
	var TeamId struct {
		Id    int64  `json:"-"`
		EncId string `json:"id"`
	}

	err := c.ShouldBindJSON(&TeamId)
	if err != nil {
		utils.HandleError(c, "Invalid input--> missing teamId", err)
		return
	}

	if TeamId.EncId != "" {
		TeamId.Id, err = crypto.NDecrypt(TeamId.EncId)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting TeamEncID(value:'%v')->%v", TeamId.EncId, err))
			return
		}
	} else {
		TeamId.Id = 0
	}

	// Ensure the game belongs to the event
	GameId, eventMatches, err := models.GetGameIdByEventGameId(EventGameId, EventId)
	if err != nil {
		utils.HandleError(c, "Oops something went wrong", fmt.Errorf("could not fetch game_id -> %v", err))
		return
	}
	if !eventMatches {
		utils.HandleError(c, "Oops something went wrong", fmt.Errorf("this game does not belong in this event (event-id mismatch)"))
		return
	}

	// Fetch user by userCode (with age and details)
	user, age, err := models.GetUserByUserCode(userCode, true, true)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.HandleInvalidEntries(c, "Could not find any user with the user code you entered", err)
			return
		}
		utils.HandleError(c, "Could not find any user with the user code you entered", err)
		return
	}
	if user.RoleSlug == "organization" {
		utils.HandleInvalidEntries(c, "Organizations are not eligible to register as participant", err)
		return
	}
	EncId := crypto.NEncrypt(int64(user.ID))

	// Check if user is already registered in this game or event
	tx, _ := database.DB.Begin()
	playerAlreadyRegistered, err := models.CheckParticipationInGameType(int64(user.ID), EventId, GameId, TeamId.Id, tx)
	if err != nil {
		utils.HandleError(c, "Oops something went wrong", err)
		return
	}
	tx.Commit()
	if playerAlreadyRegistered {
		utils.HandleInvalidEntries(c, "The player you are trying to choose is already registered for this game of the event")
		return
	}

	// // Fetch game details to validate age eligibility
	// game, err := models.GetEventGameByEventAndGame(EventId, GameId)
	// if err != nil {
	// 	utils.HandleError(c, "Oops something went wrong", err)
	// 	return
	// }

	// Perform age validation based on game category
	if age == 0 {
		// Missing age in user profile
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "User has not mentioned age in their profile, please update user profile before participation",
			"data":    gin.H{"id": EncId, "name": user.Name, "age": age},
		})
		return
	}

	isAgeValid, isGenderValid, err := models.ValidateAgeAndGender(EHGameTypeId, age, *user.Details.Gender)
	if err != nil {
		utils.HandleError(c, "Error Validating Age", err)
	}

	if !isGenderValid {
		genderError := fmt.Sprintf("%v players are not allowed in this team", *user.Details.Gender)
		if *user.Details.Gender == "" {
			genderError = "this user's gender is missing"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": genderError,
			"data":    gin.H{"id": EncId, "name": user.Name, "age": age},
		})
	} else if !isAgeValid {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "User does not satisfy age requirement for this category",
			"data":    gin.H{"id": EncId, "name": user.Name, "age": age},
		})
	} else {
		utils.HandleSuccess(c, "user found!", gin.H{"id": EncId, "name": user.Name, "age": age})
	}
}

// GetParticipatedUser fetches details of a user already registered in a specific team.
//
// Route Parameters:
//   - userId (string): Encrypted ID of the user.
//   - teamId (string): Encrypted ID of the team to check for association.
func GetParticipatedUser(c *gin.Context) {
	// Extract and validate parameters
	UserId := DecryptParamId(c, "userId", true)
	if UserId == 0 {
		return
	}
	TeamId := DecryptParamId(c, "teamId", true)
	if TeamId == 0 {
		return
	}

	// Check if the user is part of the team
	UserCode, teamHasPlayer, err := models.GetTeamPlayer(TeamId, UserId)
	if err != nil {
		utils.HandleError(c, "Oops something went wrong", err)
		return
	}
	if !teamHasPlayer {
		utils.HandleError(c, "Oops something went wrong", fmt.Errorf("one of the saved players is not in the team, corrupt entry"))
		return
	}

	// Fetch user info
	user, age, err := models.GetUserByUserCode(UserCode, true, true)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.HandleInvalidEntries(c, "Could not find any user with the user code you entered", err)
			return
		}
		utils.HandleError(c, "Could not find any user with the user code you entered", err)
		return
	}
	EncId := crypto.NEncrypt(int64(user.ID))

	//fetch tshirt size by user id
	TshirtSize, err := models.GetTshirtSizeByUserId(user.ID, TeamId)
	if err != nil {
		utils.HandleError(c, "Error fetching t-shirt size", err)
	}

	// Respond with user details
	utils.HandleSuccess(c, "user found!", gin.H{"id": EncId, "user_code": UserCode, "name": user.Name, "age": age, "tshirt_size": TshirtSize})
}

// Save teams, players, and logos for games under an event
func SaveGames(c *gin.Context) {
	c.Request.ParseMultipartForm(10 << 20) // Max size: 10MB

	var EventForm models.EventEntry
	err := c.ShouldBind(&EventForm)
	if err != nil {
		// fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse nested JSON string for games
	gameJSON := c.Request.FormValue("games")
	err = json.Unmarshal([]byte(gameJSON), &EventForm.Games)
	if err != nil {
		// fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// var prettyJSON bytes.Buffer
	// err = json.Indent(&prettyJSON, []byte(gameJSON), "", "   ")
	// if err != nil {
	// 	fmt.Println("Failed to format JSON:", err)
	// }
	// fmt.Println("incoming json:", prettyJSON.String())

	// Decrypt event ID
	EventForm.EventId, err = crypto.NDecrypt(EventForm.EventEncId)
	if err != nil {
		utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting EventId (value:'%v')->%v", EventForm.EventEncId, err))
		return
	}

	valid, err := IsEventActive(EventForm.EventId)
	if err != nil {
		utils.HandleError(c, "Oops somthing went wrong", err)
	}
	if !valid {
		utils.HandleInvalidEntries(c, "Event not found", fmt.Errorf("no active event found"))
		return
	}

	// Validate registration window
	Event, err := models.GetEventsById(EventForm.EventId)
	if err != nil {
		utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error fetching event--> %v", err))
		return
	}
	reg_srt, _ := time.Parse("02-01-2006", Event.StartRegistrationDate)
	reg_end, _ := time.Parse("02-01-2006", Event.LastRegistrationDate)
	from_dt, _ := time.Parse("02-01-2006", Event.FromDate)

	if time.Now().After(reg_srt) && time.Now().Before(reg_end) && time.Now().Before(from_dt) {
		utils.HandleError(c, "Participation for this event has closed", fmt.Errorf("participation for this event has closed:\nFromDate:%v\nLastRegistrationDate:%v", Event.FromDate, Event.LastRegistrationDate))
		return
	}

	// Decrypt user ID
	EventForm.CreatedById, err = crypto.NDecrypt(EventForm.CreatedByEncId)
	if err != nil {
		utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting CreatedById(value:'%v')->%v", EventForm.CreatedByEncId, err))
		return
	}

	// Decrypt nested team/player IDs
	for i, Game := range EventForm.Games {
		Game.EventHasGameId, err = crypto.NDecrypt(Game.EventHasGameEncId)
		if err != nil {
			utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting EventHasGameId(value:'%v')->%v", Game.EventHasGameEncId, err))
			return
		}
		if len(Game.Types) == 0 {
			utils.HandleError(c, "Please send valid data", fmt.Errorf("no teams in game"))
			return
		}
		for j, Type := range Game.Types {
			Type.TypeId, err = crypto.NDecrypt(Type.TypeEncId)
			if err != nil {
				utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting TypeId(value:'%v')->%v", Type.TypeEncId, err))
				return
			}
			for k, Team := range Type.Teams {
				if Team.TeamEncId != "" {
					Team.TeamId, err = crypto.NDecrypt(Team.TeamEncId)
					if err != nil {
						utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting TeamId(value:'%v')->%v", Team.TeamEncId, err))
						return
					}
				}
				Team.EventHasGameTypeId, err = crypto.NDecrypt(Team.EventHasGameTypeEncId)
				if err != nil {
					utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting EventHasGameTypeId(value:'%v')->%v", Team.EventHasGameTypeEncId, err))
					return
				}

				Team.AgeGroupId, err = crypto.NDecrypt(Team.AgeGroupEncId)
				if err != nil {
					utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting AgeGroupId(value:'%v')->%v", Team.AgeGroupEncId, err))
					return
				}
				Team.TeamCaptainID, err = crypto.NDecrypt(Team.TeamCaptainEncId)
				if err != nil {
					utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting TeamCaptainID(value:'%v')->%v", Team.TeamCaptain, Team.TeamCaptainEncId+" "+err.Error()))
					return
				}
				for _, EncMember := range Team.TeamMemberEncId {
					member, err := crypto.NDecrypt(EncMember)
					if err != nil {
						utils.HandleError(c, "Oops something went wrong", fmt.Errorf("error decrypting User id for a member(value:'%v')->%v", EncMember, err))
						return
					}
					Team.TeamMemberIDs = append(Team.TeamMemberIDs, member)
				}
				Type.Teams[k] = Team
			}
			Game.Types[j] = Type
		}
		EventForm.Games[i] = Game
	}

	// Save game entries to DB
	ReturnedEvent, tx, err := models.SaveGames(EventForm)
	if err != nil {
		utils.HandleError(c, "Error registering for this event", err)
		return
	}

	// Handle logo uploads and updates
	var TypeResponse []struct {
		TypeId        string           `json:"type_id"`
		AgeGroupArray []ResponseObject `json:"age_group_array"`
	}
	for i, Game := range ReturnedEvent.Games {
		for j, Type := range Game.Types {
			var responseObject []ResponseObject
			for k, Team := range Type.Teams {
				Team.TeamEncId = crypto.NEncrypt(Team.TeamId)
				convertedSizes := make(map[string]string)
				for k, v := range Team.TshirtSize {
					convertedSizes[k] = v
				}

				ResponseObjectLocal := ResponseObject{
					TypeId:       Team.EventHasGameTypeEncId,
					AgeGroupId:   Team.AgeGroupEncId,
					TeamId:       Team.TeamEncId,
					TeamLogoPath: Team.TeamLogoPath,
					TshirtSize:   Team.TshirtSize,
				}
				// ResponseObjectLocal := ResponseObject{
				// 	TypeId:     Team.EventHasGameTypeEncId,
				// 	AgeGroupId: Team.AgeGroupEncId,
				// 	TeamId:     Team.TeamEncId,
				// }
				Logokey := strings.ReplaceAll(fmt.Sprintf("%v_%v_%v_logo", Game.GameName, Team.TeamName, Team.AgeGroupCategory), " ", "")
				TeamLogo, err := c.FormFile(Logokey)
				if err != nil && err.Error() == "http: no such file" {
					ResponseObjectLocal.TeamLogoPath = Team.TeamLogoPath
					goto afterUploadBlock
				} else if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					tx.Rollback()
					return
				}

				//name & path generation for logo
				if Team.TeamLogoPath == "" {
					timestamp := time.Now().Format("20060102150405")
					LogoName := strings.ReplaceAll(fmt.Sprintf("%v_%v_%v_%v_Logo_%v.png", Game.GameName, Type.TypeName, Team.AgeGroupCategory, Team.TeamName, timestamp), " ", "")
					Team.TeamLogoPath = filepath.Join(TeamFolderPath, LogoName)

					Team.TeamLogoPath, err = models.LogoUpdate(Team.TeamLogoPath, Team.TeamId, tx)
					if err != nil {
						//if there is an error in the function it'll be rolled back in the function itself
						utils.HandleError(c, fmt.Sprintf("Error updating logo for %v in db", Team.TeamName), fmt.Errorf("error setting teamlogo for %v team ->%v", Team.TeamName, err))
						return
					}
					ResponseObjectLocal.TeamLogoPath = Team.TeamLogoPath
				}
				if !(strings.HasPrefix(Team.TeamLogoPath, "http://localhost:8080/public\\event\\team_logos") || strings.HasPrefix(Team.TeamLogoPath, "http://localhost:8080/public/event/team_logos")) {
					err = c.SaveUploadedFile(TeamLogo, Team.TeamLogoPath)
					if err != nil {
						utils.HandleError(c, fmt.Sprintf("Error uploading logo for %v", Team.TeamName), fmt.Errorf("error setting teamlogo for %v team ->%v", Team.TeamName, err))
						tx.Rollback()
						return
					}
				}
			afterUploadBlock:
				Type.Teams[k] = Team
				responseObject = append(responseObject, ResponseObjectLocal)
			}
			var TypeResponseLocal struct {
				TypeId        string           `json:"type_id"`
				AgeGroupArray []ResponseObject `json:"age_group_array"`
			}
			TypeResponseLocal.TypeId = Type.TypeEncId
			TypeResponseLocal.AgeGroupArray = responseObject

			TypeResponse = append(TypeResponse, TypeResponseLocal)

			Game.Types[j] = Type
		}
		ReturnedEvent.Games[i] = Game
	}
	//final commit for the transaction
	tx.Commit()
	utils.HandleSuccess(c, "Saved your teams for "+ReturnedEvent.Games[0].GameName+" successfully", TypeResponse)
}

// Final submission of all registered teams for an event
func FinalizeParticipation(c *gin.Context) {
	var obj struct {
		CreatedByEncId string   `json:"created_by_id" validate:"required"`
		CreatedById    int64    `json:"-"`
		EventEncId     string   `json:"event_id" validate:"required"`
		EventId        int64    `json:"-"`
		GameEncIdArr   []string `json:"game_id_arr" validate:"required"`
		GameIdArr      []int64  `json:"-"`
	}
	err := c.ShouldBindJSON(&obj)
	if err != nil {
		utils.HandleError(c, "oops something went wrong", fmt.Errorf("invalid input--> %v", err))
		return
	}

	//decryption
	obj.CreatedById, err = crypto.NDecrypt(obj.CreatedByEncId)
	if err != nil {
		utils.HandleError(c, "oops somthing went wrong", fmt.Errorf("decrytion error--> %v", err))
		return
	}
	obj.EventId, err = crypto.NDecrypt(obj.EventEncId)
	if err != nil {
		utils.HandleError(c, "oops somthing went wrong", fmt.Errorf("decrytion error--> %v", err))
		return
	}

	valid, err := IsEventActive(obj.EventId)
	if err != nil {
		utils.HandleError(c, "Oops somthing went wrong", err)
	}
	if !valid {
		utils.HandleInvalidEntries(c, "Event not found", fmt.Errorf("no active event found"))
		return
	}

	for _, GameEncId := range obj.GameEncIdArr {
		GameId, err := crypto.NDecrypt(GameEncId)
		if err != nil {
			utils.HandleError(c, "oops somthing went wrong", fmt.Errorf("decrytion error--> %v", err))
			return
		}
		obj.GameIdArr = append(obj.GameIdArr, GameId)
	}
	err = models.FinalizeParticipation(obj.CreatedById, obj.EventId, obj.GameIdArr)
	if err != nil {
		utils.HandleError(c, "oops somthing went wrong", fmt.Errorf("could not finalize participation--> %v", err))
		return
	}

	utils.HandleSuccess(c, "Submition successfull")
}
