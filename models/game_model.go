package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sports-events-api/utils"
	"strconv"
	"time"

	"github.com/lib/pq"
)

type Game struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name,omitempty" validate:"omitempty,min=2"`
	Slug       string    `json:"slug" gorm:"unique;not null"`
	GameTypeID []int64   `json:"game_type_id" validate:"required"`
	AgeGroupID []int64   `json:"age_group_id" validate:"required"`
	Status     string    `json:"status,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

type AgeGroup struct {
	EncID     string    `json:"id"`
	ID        int64     `json:"-"`
	Category  string    `json:"category,omitempty" validate:"omitempty,min=2"`
	MinAge    int       `json:"min_age"`
	MaxAge    int       `json:"max_age"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
type GameUpdate struct {
	Name        string   `json:"game_name"`
	GameTypeIDs []string `json:"game_type_id"`
	AgeGroupIDs []string `json:"age_group_id"`
}

type GameHasType struct {
	ID         int64     `json:"id"`
	GameID     int64     `json:"game_id"`
	GameTypeID int64     `json:"game_type_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type GameHasAgeGroup struct {
	ID         int64     `json:"id"`
	GameId     int64     `json:"game_id"`
	AgeGroupId int64     `json:"age_group_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type GameInfo struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	GameTypes []GameType `json:"game_types"`
	AgeGroups []AgeGroup `json:"age_groups"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func InsertGame(game *Game) (*Game, error) {
	var gameID int64
	query := `INSERT INTO games(game_name,slug,status)
			  VALUES ($1,$2,$3) RETURNING id`

	err := database.DB.QueryRow(query, game.Name, game.Slug, "Active").Scan(&gameID)
	if err != nil {
		fmt.Printf("Error During Database Query: %v\n", err)
		return nil, fmt.Errorf("unable to create game: %v", err)
	}

	game.ID = gameID
	return game, nil
}

func GetGames(search, sort, dir string, limit, offset int64) (int, []Game, error) {
	var games []Game
	args := []interface{}{limit, offset}
	whereClause := ""

	if search != "" {
		whereClause = "WHERE game_name ILIKE $3"
		args = append(args, "%"+search+"%")
	}

	query := fmt.Sprintf(
		`SELECT id, game_name, created_at, updated_at,status, COUNT(*) OVER() AS totalrecords
		 FROM games
		 %s
		 ORDER BY %s %s
		 LIMIT $1 OFFSET $2`,
		whereClause, sort, dir,
	)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying games: %v\n", err)
		return 0, nil, err
	}
	defer rows.Close()

	totalRecords := 0
	for rows.Next() {
		var game Game
		if err := rows.Scan(
			&game.ID,
			&game.Name,
			&game.CreatedAt,
			&game.UpdatedAt,
			&game.Status,
			&totalRecords,
		); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			return 0, nil, err
		}
		games = append(games, game)
	}

	return totalRecords, games, nil
}

func GetGamesById(id int64) (Game, error) {
	query := `SELECT id, game_name, created_at, updated_at FROM games WHERE id=$1`

	var game Game
	err := database.DB.QueryRow(query, id).Scan(
		&game.ID,
		&game.Name,
		&game.CreatedAt,
		&game.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return game, fmt.Errorf("game type with ID %d is not found", id)
	} else if err != nil {
		return game, fmt.Errorf("error fetching game type: %v", err)
	}
	game.GameTypeID, _ = GetTypeByGameId(id)
	game.AgeGroupID, _ = GetAgeGroupIDByGameId(id)

	game.AgeGroupID, err = GetAgeGroupIDByGameId(id)
	if err != nil {
		return game, err
	}

	return game, nil
}

func DeleteGameById(id int64) error {
	query := `DELETE FROM games WHERE id=$1`

	result, err := database.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error deleting game type: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("game with ID %d is not found", id)
	}

	return nil
}

func UpdateGameById(id int64, gameUpdate GameUpdate) error {
	slug := utils.GenerateSlug(gameUpdate.Name)

	// Update the game name and slug
	query := `UPDATE games SET game_name=$1, slug=$2 WHERE id=$3`
	result, err := database.DB.Exec(query, gameUpdate.Name, slug, id)
	if err != nil {
		return fmt.Errorf("error updating game: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("game with ID %d is not found", id)
	}

	var gameTypeIDs []int64
	for _, encryptedID := range gameUpdate.GameTypeIDs {
		decryptedTypeID, err := crypto.NDecrypt(encryptedID)
		if err != nil {
			return fmt.Errorf("failed to decrypt game type ID: %v", err)
		}
		gameTypeIDs = append(gameTypeIDs, decryptedTypeID)
	}

	err = UpdateGameTypeAssociations(id, gameTypeIDs)
	if err != nil {
		return err
	}

	var ageGroupIds []int64
	for _, encryptedID := range gameUpdate.AgeGroupIDs {
		decryptedAgeID, err := crypto.NDecrypt(encryptedID)
		if err != nil {
			return fmt.Errorf("failed to decrypt game type ID: %v", err)
		}
		ageGroupIds = append(ageGroupIds, decryptedAgeID)
	}

	err = UpdateAgeGroup(id, ageGroupIds)
	if err != nil {
		return err
	}

	return nil
}

func GetTypeByGameId(gameId int64) ([]int64, error) {
	query := `SELECT game_type_id FROM game_has_types WHERE game_id=$1`
	var type_ids []int64
	rows, err := database.DB.Query(query, gameId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var x int64
		err := rows.Scan(&x)
		if err != nil {
			return nil, err
		}
		type_ids = append(type_ids, x)
	}

	return type_ids, nil
}

func GetAgeGroupIDByGameId(gameId int64) ([]int64, error) {
	query := `SELECT age_group_id FROM game_has_age_group WHERE game_id=$1`
	var age_group_ids []int64
	rows, err := database.DB.Query(query, gameId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var x int64
		err := rows.Scan(&x)
		if err != nil {
			return nil, err
		}
		age_group_ids = append(age_group_ids, x)
	}

	return age_group_ids, nil
}

func GetAgeGroupsByGameId(gameId int64, tx *sql.Tx) ([]AgeGroup, []ShortAgeGroup, error) {
	query := `
	SELECT JSON_AGG(res)
		FROM (SELECT ag.id::TEXT, ag.category, ag.minage as min_age, ag.maxage as max_age, ag.status,
		to_char(ag.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') as created_at,
		to_char(ag.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') as updated_at
		FROM age_group ag
			JOIN game_has_age_group gm_ag ON gm_ag.age_group_id = ag.id
			WHERE game_id=$1) res`
	var AgeGroups []AgeGroup
	var ShortAgeGroups []ShortAgeGroup
	var resJson sql.NullString
	err := tx.QueryRow(query, gameId).Scan(&resJson)
	if err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	if resJson.Valid {
		// Only unmarshal if the string is not empty
		if len(resJson.String) > 0 {
			err = json.Unmarshal([]byte(resJson.String), &AgeGroups)
			if err != nil {
				tx.Rollback()
				return nil, nil, err
			}

			for i := range AgeGroups {
				AgeGroups[i].ID, _ = strconv.ParseInt(AgeGroups[i].EncID, 10, 64)
				AgeGroups[i].EncID = ""
			}

			err = json.Unmarshal([]byte(resJson.String), &ShortAgeGroups)
			if err != nil {
				tx.Rollback()
				return nil, nil, err
			}

			for i := range ShortAgeGroups {
				ShortAgeGroups[i].ID, _ = strconv.ParseInt(ShortAgeGroups[i].EncID, 10, 64)
				ShortAgeGroups[i].EncID = ""
			}
		} else {
			// Empty JSON string is considered valid but unmarshal would fail
			AgeGroups = []AgeGroup{}
			ShortAgeGroups = []ShortAgeGroup{}
		}
	} else {
		// Null JSON from DB
		AgeGroups = []AgeGroup{}
		ShortAgeGroups = []ShortAgeGroup{}
	}

	return AgeGroups, ShortAgeGroups, nil
}

func UpdateGameTypeAssociations(gameID int64, gameTypeIDs []int64) error {
	clearQuery := `DELETE FROM game_has_types WHERE game_id=$1`
	_, err := database.DB.Exec(clearQuery, gameID)
	if err != nil {
		return fmt.Errorf("error clearing game type associations: %v", err)
	}

	for _, gameTypeID := range gameTypeIDs {
		insertQuery := `INSERT INTO game_has_types (game_id, game_type_id) VALUES ($1, $2)`
		_, err := database.DB.Exec(insertQuery, gameID, gameTypeID)
		if err != nil {
			return fmt.Errorf("error inserting game type association: %v", err)
		}
	}

	return nil
}

func UpdateAgeGroup(gameID int64, ageGroupIds []int64) error {
	clearQuery := `DELETE FROM game_has_age_group WHERE game_id=$1`
	_, err := database.DB.Exec(clearQuery, gameID)
	if err != nil {
		return fmt.Errorf("error deleting game has age group: %v", err)
	}

	for _, ageGroupId := range ageGroupIds {
		insertQuery := `INSERT INTO game_has_age_group (game_id, age_group_id) VALUES ($1, $2)`
		_, err := database.DB.Exec(insertQuery, gameID, ageGroupId)
		if err != nil {
			return fmt.Errorf("error inserting game has age group: %v", err)
		}
	}
	return nil
}

func UpdateGameStatus(gameID int64, status string) error {
	query := `UPDATE games SET status = $1 WHERE id = $2`

	_, err := database.DB.Exec(query, status, gameID)
	if err != nil {
		log.Printf("Error updating status for game ID %v: %v\n", gameID, err)
		return fmt.Errorf("failed to update status")
	}

	return nil
}

func GetGameStatusByID(gameID int64) (string, error) {
	var status string
	query := `SELECT status FROM games WHERE id = $1`
	err := database.DB.QueryRow(query, gameID).Scan(&status)
	if err != nil {
		log.Printf("Error fetching status for game ID %d: %v\n", gameID, err)
		return "", fmt.Errorf("failed to fetch status")
	}
	return status, nil
}

type GameType struct {
	EncID string `json:"id"`
	ID    int64  `json:"-"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
}

type ShortAgeGroup struct {
	EncID    string `json:"id"`
	ID       int64  `json:"-"`
	Category string `json:"category"`
	Slug     string `json:"slug"`
}

func GetGameTypeById(id int64) (*GameType, error) {
	query := `SELECT id, name FROM games_types WHERE id=$1`

	var gameType GameType
	err := database.DB.QueryRow(query, id).Scan(&gameType.ID, &gameType.Name)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("game type with ID %d is not found", id)
	} else if err != nil {
		return nil, fmt.Errorf("error fetching game type: %v", err)
	}

	return &gameType, nil
}

func InsertGameHasType(gameHasType *GameHasType) (*GameHasType, error) {
	gameHasType.CreatedAt = time.Now()
	gameHasType.UpdatedAt = time.Now()

	query := `INSERT INTO game_has_types (game_id, game_type_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := database.DB.QueryRow(query, gameHasType.GameID, gameHasType.GameTypeID, gameHasType.CreatedAt, gameHasType.UpdatedAt).Scan(&gameHasType.ID)
	if err != nil {
		fmt.Printf("Error inserting into game_has_types: %v\n", err)
		return nil, fmt.Errorf("unable to create game_has_type: %v", err)
	}

	return gameHasType, nil
}

func InsertGameHasAgeGroup(gameHasAgeGroup *GameHasAgeGroup) (*GameHasAgeGroup, error) {
	gameHasAgeGroup.CreatedAt = time.Now()
	gameHasAgeGroup.UpdatedAt = time.Now()

	query := `INSERT INTO game_has_age_group (game_id, age_group_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := database.DB.QueryRow(query, gameHasAgeGroup.GameId, gameHasAgeGroup.AgeGroupId, gameHasAgeGroup.CreatedAt, gameHasAgeGroup.UpdatedAt).Scan(&gameHasAgeGroup.ID)
	if err != nil {
		fmt.Printf("Error inserting into game_has_types: %v\n", err)
		return nil, fmt.Errorf("unable to create game_has_type: %v", err)
	}

	return gameHasAgeGroup, nil
}

func GetAllGameTypes() ([]GameType, error) {
	var gameTypes []GameType

	query := `SELECT id, name FROM games_types WHERE status = 'Active'`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch game types: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var gameType GameType
		if err := rows.Scan(&gameType.ID, &gameType.Name); err != nil {
			return nil, fmt.Errorf("error scanning game type: %v", err)
		}
		gameTypes = append(gameTypes, gameType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over game types: %v", err)
	}

	return gameTypes, nil
}

func GetAllAgeGroup() ([]ShortAgeGroup, error) {
	var ageGroups []ShortAgeGroup

	query := `SELECT id, category FROM age_group
	ORDER BY maxage asc, minage ASC;`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch age group: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ageGroup ShortAgeGroup
		if err := rows.Scan(&ageGroup.ID, &ageGroup.Category); err != nil {
			return nil, fmt.Errorf("error scanning age group: %v", err)
		}
		ageGroups = append(ageGroups, ageGroup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over age groups: %v", err)
	}

	return ageGroups, nil
}

func GetGameIdByEventGameId(EventGameId int64, EventId int64) (int64, bool, error) {
	query := `SELECT game_id, CASE WHEN event_id = $1 THEN TRUE ELSE FALSE END AS event_matches
			FROM event_has_games WHERE id=$2`
	var GameId int64
	var eventMatches bool
	err := database.DB.QueryRow(query, EventId, EventGameId).Scan(&GameId, &eventMatches)
	if err != nil {
		return 0, false, fmt.Errorf("could not fetch game_id -> %v", err)
	}
	return int64(GameId), eventMatches, nil
}

func GetGamesInfoByGameIds(ids []int64) ([]*GameInfo, error) {
	query := `
	SELECT 
		g.id AS game_id,
		g.game_name,
		g.slug,
		g.status,
		g.created_at,
		g.updated_at,

		gt.id AS type_id,
		gt.name AS type_name,
		gt.slug AS type_slug,

		ag.id AS age_group_id,
		ag.category AS age_group_name,
		ag.slug AS age_group_slug

	FROM games g
	LEFT JOIN game_has_types ght ON g.id = ght.game_id
	LEFT JOIN games_types gt ON ght.game_type_id = gt.id
	LEFT JOIN game_has_age_group ghag ON g.id = ghag.game_id
	LEFT JOIN age_group ag ON ghag.age_group_id = ag.id
	WHERE g.id = ANY($1)
	`

	rows, err := database.DB.Query(query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gameMap := make(map[int64]*GameInfo)

	for rows.Next() {
		var (
			gameID, typeID, ageGroupID sql.NullInt64
			gameName, gameSlug, status string
			typeName, typeSlug         sql.NullString
			ageGroupName, ageGroupSlug sql.NullString
			createdAt, updatedAt       time.Time
		)

		if err := rows.Scan(&gameID, &gameName, &gameSlug, &status, &createdAt, &updatedAt,
			&typeID, &typeName, &typeSlug,
			&ageGroupID, &ageGroupName, &ageGroupSlug); err != nil {
			return nil, err
		}

		game, exists := gameMap[gameID.Int64]
		if !exists {
			game = &GameInfo{
				ID:        gameID.Int64,
				Name:      gameName,
				Slug:      gameSlug,
				Status:    status,
				GameTypes: []GameType{},
				AgeGroups: []AgeGroup{},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}
			gameMap[gameID.Int64] = game
		}

		if typeID.Valid {
			exists := false
			for _, gt := range game.GameTypes {
				if gt.ID == typeID.Int64 {
					exists = true
					break
				}
			}
			if !exists {
				game.GameTypes = append(game.GameTypes, GameType{
					ID:   typeID.Int64,
					Name: typeName.String,
					Slug: typeSlug.String,
				})
			}
		}

		if ageGroupID.Valid {
			exists := false
			for _, ag := range game.AgeGroups {
				if ag.ID == ageGroupID.Int64 {
					exists = true
					break
				}
			}
			if !exists {
				game.AgeGroups = append(game.AgeGroups, AgeGroup{
					ID:       ageGroupID.Int64,
					Category: ageGroupName.String,
				})
			}
		}
	}

	var result []*GameInfo
	for _, game := range gameMap {
		result = append(result, game)
	}
	return result, nil
}

func containsInt64(slice []int64, val int64) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func GetGameNameById(id int64) (string, error) {
	var name string
	query := `SELECT game_name FROM games WHERE id = $1`
	err := database.DB.QueryRow(query, id).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}
