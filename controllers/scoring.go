package controllers

import (
	"fmt"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// ProcessMatchTeamScores handles inserting, updating, and deleting match scores based on action.
//
// This function:
//  1. Validates and binds the incoming JSON payload to a MatchTeamScorePayload struct.
//  2. Decrypts the encrypted match ID and team ID from the payload.
//  3. Determines the game's super category to apply appropriate scoring rules.
//  4. Validates each score object according to the game's category rules (e.g., distance, player ID, set number, etc.).
//  5. Decrypts encrypted IDs for players and scores where applicable.
//  6. Passes validated data to the models layer to be processed in the database.
//  7. Re-encrypts IDs for the response and sends back a success message.
//
// Any invalid data or decryption errors result in an immediate error response.
func ProcessMatchTeamScores(c *gin.Context) {
	var payload models.MatchTeamScorePayload

	err := c.ShouldBindJSON(&payload)
	if err != nil {
		utils.HandleError(c, "Invalid request body", err)
		return
	}

	payload.MatchID, err = crypto.NDecrypt(payload.MatchEncID)
	if err != nil {
		utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting MatchEncId(value:'%v')->%v", payload.MatchEncID, err))
		return
	}

	payload.TeamID, err = crypto.NDecrypt(payload.TeamEncID)
	if err != nil {
		utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting TeamEncId(value:'%v')->%v", payload.TeamEncID, err))
		return
	}

	superCategory, err := GetSuperGameCategory(payload.GameSuperCategory, payload.GameSubCategory, payload.TypeOfTournament)
	if err != nil {
		utils.HandleError(c, "Unknown game category", err)
		return
	}

	for i := range payload.Scores {
		score := &payload.Scores[i]

		setNoCheck := score.SetNo > 0
		PlayerIdCheck := score.PlayerEncID != "" && score.PlayerEncID != "NzaT8AIHknhZ_9VoglF8yVl-qk3Muw"
		ScoredAtCheck := score.ScoredAt != ""
		PenaltyCheck := score.IsPenalty
		// PointsCheck := score.PointsScored > 1
		PointsCheck := score.PointsScored >= 1
		if superCategory.Name == "Aethletics" {
			PointsCheck = superCategory.HasPointsScored
		}

		metricString := strings.Trim(strings.ToLower(score.Metric), " ")
		score.Metric = metricString

		DistanceCheck := score.Distance > 0 && (metricString == "meters" ||
			metricString == "centimeters" ||
			metricString == "feet" ||
			metricString == "inches")

		if setNoCheck != superCategory.HasSetNo ||
			PlayerIdCheck != superCategory.HasPlayerId ||
			DistanceCheck != superCategory.HasDistance ||
			(ScoredAtCheck != superCategory.HasScoredAt && PenaltyCheck != superCategory.HasIsPenalty && PointsCheck != superCategory.HasPointsScored) {

			err := fmt.Errorf(`set number check: %v,
player id check: %v,
distance check: %v,
scores check: %v,
points check: %v,`,
				setNoCheck == superCategory.HasSetNo,
				PlayerIdCheck == superCategory.HasPlayerId,
				DistanceCheck == superCategory.HasDistance,
				ScoredAtCheck == superCategory.HasScoredAt,
				PointsCheck == superCategory.HasPointsScored)
			utils.HandleError(c, "Bad Payload", err)
			return
		}

		if PlayerIdCheck {
			score.PlayerID, err = crypto.NDecrypt(score.PlayerEncID)
			if err != nil {
				utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting PlayerEncId(value:'%v')->%v", score.PlayerEncID, err))
				return
			}
		}

		if score.Action != "insert" {
			score.ID, err = crypto.NDecrypt(score.EncID)
			if err != nil {
				utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting ScoreEncId(value:'%v')->%v", score.EncID, err))
				return
			}
		}
	}

	for i := range payload.Cards {
		card := &payload.Cards[i]

		card.PlayerID, err = crypto.NDecrypt(card.PlayerEncID)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting Card PlayerEncID(value:'%v')->%v", card.PlayerEncID, err))
			return
		}

		if card.Action != "insert" {
			if card.EncID == "" {
				card.Action = "insert"
			} else {
				cardID, err := crypto.NDecrypt(card.EncID)
				if err != nil {
					utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting CardEncID(value:'%v')->%v", card.EncID, err))
					return
				}
				card.ID = cardID
			}
		}
	}

	payload, err = models.ProcessMatchTeamScores(payload)
	if err != nil {
		utils.HandleError(c, "Failed to process scores", err)
		return
	}

	payload, err = models.ProcessMatchPlayerCardsFromScores(payload, payload.GameSubCategory)
	if err != nil {
		utils.HandleError(c, "Failed to process cards", err)
		return
	}

	for i := range payload.Scores {
		payload.Scores[i].EncID = crypto.NEncrypt(payload.Scores[i].ID)
		payload.Scores[i].Action = ""
	}

	for i := range payload.Cards {
		payload.Cards[i].EncID = crypto.NEncrypt(payload.Cards[i].ID)
		payload.Cards[i].Action = ""
	}

	// utils.HandleSuccess(c, "Scores & Cards saved successfully", payload)
	msg := ""
	hasScores := len(payload.Scores) > 0
	hasCards := len(payload.Cards) > 0

	switch {
	case hasScores && hasCards:
		msg = "Scores & Cards saved successfully"
	case hasScores:
		msg = "Scores saved successfully"
	case hasCards:
		msg = "Cards saved successfully"
	default:
		msg = "Saved successfully"
	}

	utils.HandleSuccess(c, msg, payload)
}

func AllocateWinner(c *gin.Context) {
	var x bool
	var WinTeamId int64
	payload := struct {
		WinTeamID         string `json:"win_team_id"`
		MatchID           string `json:"match_id" validate:"required"`
		IsDraw            *bool  `json:"is_draw"`
		TypeOfTournament  string `json:"type_of_tournament"`
		GameSuperCategory string `json:"game_super_category"`
		GameSubCategory   string `json:"game_sub_category"`
	}{
		WinTeamID: "",
		MatchID:   "",
		IsDraw:    &x,
	}

	err := c.ShouldBindJSON(&payload)
	if err != nil {
		utils.HandleError(c, "Invalid request body", err)
		return
	}

	MatchID, err := crypto.NDecrypt(payload.MatchID)
	if err != nil {
		utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting MatchEncId(value:'%v')->%v", payload.MatchID, err))
		return
	}

	tx, _ := database.DB.Begin()

	superCategory, err := GetSuperGameCategory(payload.GameSuperCategory, payload.GameSubCategory, payload.TypeOfTournament)
	if err != nil {
		utils.HandleError(c, "Unknown game category", err)
		return
	}

	// Decrypt organization ID
	if !*payload.IsDraw && !superCategory.CalculateScore {
		WinTeamId, err = crypto.NDecrypt(payload.WinTeamID)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting WinTeamEncId(value:'%v')->%v", payload.WinTeamID, err))
			return
		}
	} else if superCategory.CalculateScore {
		WinTeamId, *payload.IsDraw, err = models.DetermineWinner(MatchID, superCategory.Name, payload.GameSubCategory, tx)
		if err != nil {
			fmt.Println("SuperCategory:", payload.GameSuperCategory, ", SubCategory:", payload.GameSubCategory)
			tx.Rollback()
			utils.HandleError(c, "Error Determining winner", err)
			return
		}
	}

	if *payload.IsDraw && payload.TypeOfTournament == "Knockout" {
		utils.HandleInvalidEntries(c, "You cant declare a draw in a knockout tournament")
		return
	}
	if payload.GameSubCategory == "RankBased" {
		err := models.DeleteAllMatchScores(MatchID, tx)
		if err != nil {
			tx.Rollback()
			utils.HandleError(c, "error deleting prvious scores", err)
			return
		}
	}

	err = models.AllocateWinner(MatchID, WinTeamId, *payload.IsDraw, tx)
	if err != nil {
		utils.HandleError(c, "error allocating winners", err)
		tx.Rollback()
		return
	}
	tx.Commit()
	utils.HandleSuccess(c, "Winner assigned successfully", gin.H{
		"win_team_id": crypto.NEncrypt(WinTeamId),
		"is_draw":     *payload.IsDraw,
	})
}

// GetMatchTeamScores fetches all scores for a specific match_has_teams_id
func GetMatchTeamScores(c *gin.Context) {
	type ScoreResponse struct {
		TeamId string         `json:"team_id"`
		Scores []models.Score `json:"scores"`
		Cards  []models.Card  `json:"cards"`
	}

	data := struct {
		MatchId string   `json:"match_id"`
		TeamIds []string `json:"team_ids"`
	}{
		MatchId: "",
		TeamIds: []string{},
	}
	scoreData := []ScoreResponse{}

	if err := c.ShouldBindBodyWithJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid payload",
			"data":    err.Error(),
		})
		return
	}

	// Decrypt Match ID
	MatchId, err := crypto.NDecrypt(data.MatchId)
	if err != nil {
		utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting MatchEncId(value:'%v')->%v", data.MatchId, err))
		return
	}

	for _, TeamEncId := range data.TeamIds {
		TeamId, err := crypto.NDecrypt(TeamEncId)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting TeamEncId(value:'%v')->%v", TeamEncId, err))
			return
		}

		//Fetch scores
		scores, err := models.GetScoresByMatchTeamID(MatchId, TeamId)
		if err != nil {
			utils.HandleError(c, "Failed to retrieve scores", err)
			return
		}
		for i := range scores {
			scores[i].EncID = crypto.NEncrypt(scores[i].ID)
			scores[i].PlayerEncID = crypto.NEncrypt(scores[i].PlayerID)
		}

		//Fetch cards
		cards, err := models.GetCardsByMatchTeamID(MatchId, TeamId)
		if err != nil {
			utils.HandleError(c, "Failed to retrieve cards", err)
			return
		}
		for i := range cards {
			cards[i].EncID = crypto.NEncrypt(cards[i].ID)
			cards[i].PlayerEncID = crypto.NEncrypt(cards[i].PlayerID)
		}

		//Append both scores + cards
		scoreData = append(scoreData, ScoreResponse{
			TeamId: TeamEncId,
			Scores: scores,
			Cards:  cards,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   scoreData,
	})
}
