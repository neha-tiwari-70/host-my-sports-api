package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"sports-events-api/database"
	"strconv"

	// "sports-events-api/models"
	"sports-events-api/utils"
	"strings"
	"time"
)

// Event represents a sports event with metadata like location, organizer details,
// duration, registration period, and associated games.
//
// Fields:
//   - ID: Internal DB ID.
//   - EncID: Encrypted ID for external reference.
//   - CreatedBy: Organizer information (encrypted ID, name, role, email).
//   - Name, Dates, Venue: Basic event info.
//   - Location: Encrypted and internal IDs for state and city.
//   - Fees: Event registration fee.
//   - Games: List of games in this event (each with types, duration, etc.).
//   - Social Media Links: Facebook, Instagram, YouTube.
//   - Media: Logo and sponsor-related fields.
//   - About: Description of the event.
//   - Status: Soft delete or publish state.
//   - CreatedAt, UpdatedAt: Timestamps.
type Event struct {
	ID                      int64                 `form:"id" json:"id"`
	EncID                   string                `form:"enc_id,omitempty"`
	Slug                    string                `form:"slug" json:"slug"`
	CreatedByEncId          string                `form:"created_by_id" json:"created_by_id" binding:"required"`
	CreatedById             int64                 `form:"-" json:"-"`
	CreatedByRole           string                `form:"created_by_role" json:"created_by_role" binding:"required"`
	CreatedByName           string                `form:"created_by_name" json:"created_by_name"`
	CreatedByEmail          string                `form:"created_by_email" json:"created_by_email"`
	Name                    string                `form:"name" json:"name" validate:"required,min=2"`
	FromDate                string                `form:"from_date" json:"from_date" validate:"required"`
	ToDate                  string                `form:"to_date" json:"to_date" validate:"required"`
	LastRegistrationDate    string                `json:"last_registration_date"`
	StartRegistrationDate   string                `json:"start_registration_date"`
	GoogleMapLink           string                `json:"google_map_link"`
	StateEncId              string                `form:"state_id" json:"state_id" binding:"required"`
	StateId                 int64                 `form:"-" json:"-"`
	CityEncId               string                `form:"city_id" json:"city_id" binding:"required"`
	CityId                  int64                 `form:"-" json:"-"`
	Venue                   string                `form:"venue" json:"venue" validate:"required"`
	Fees                    string                `form:"fees" json:"fees" validate:"required"`
	Games                   []EventGame           `json:"games"`
	FacebookLink            string                `form:"facebook_link" json:"facebook_link"`
	InstagramLink           string                `form:"instagram_link" json:"instagram_link"`
	LinkedinLink            *string               `form:"linkedin_link" json:"linkedin_link"`
	Logo                    string                `form:"logo" json:"logo"`
	LogoPath                string                `form:"-" json:"-"`
	SponsorLogo             *multipart.FileHeader `form:"sponsor_logo" json:"sponsor_logo"`
	SponsorLogoPath         string                `form:"-" json:"-"`
	SponsorTitle            string                `form:"sponsor_title" json:"sponsor_title"`
	About                   string                `form:"about" json:"about"`
	LevelOfCompetition      int64                 `form:"level_of_competition_id"`
	LevelOfCompetitionTitle string                `form:"title"`
	TeamCount               int                   `json:"team_count"`
	Status                  string                `form:"status" json:"status,omitempty"`
	CreatedAt               time.Time             `form:"created_at" json:"created_at,omitempty"`
	UpdatedAt               time.Time             `form:"updated_at" json:"updated_at,omitempty"`
	// LinkedinLink            string                `form:"linkedin_link json:"linkedin_link"`
}

// EventGame defines a game under an event with additional settings like type of tournament,
// team configuration, categories, and fee structure.
//
// Fields:
//   - ID, EncID: Internal and encrypted game mapping IDs.
//   - GameID, GameEncID: Links to master game data.
//   - TypeOfTournament: e.g. Knockout, League.
//   - Type: Game type list like singles/doubles.
//   - Category/AgeGroup: Restrictions or divisions.
//   - MaxRegistration: Registration cap per game.
//   - Sets: Number of sets to play.
//   - MaxRegistrationsReached: Boolean flag to prevent overflow.
type EventGame struct {
	EncID                   string          `json:"id" validate:"required"`
	ID                      int64           `json:"-"`
	EventID                 int             `json:"event_id" validate:"required"`
	GameEncID               string          `json:"game_id" validate:"required"`
	GameID                  int64           `json:"-"`
	TypeOfTournament        string          `json:"type_of_tournament,omitempty"`
	TeamSize                int64           `json:"team_size,omitempty"`
	Duration                string          `json:"duration,omitempty"`
	Type                    []EventGameType `json:"type,omitempty"`
	Category                string          `json:"category,omitempty"`
	AgeGroup                string          `json:"age_group,omitempty"`
	MaxRegistration         int             `json:"max_registration,omitempty"`
	Sets                    int             `json:"sets,omitempty"`
	NumberOfPlayers         int             `json:"number_of_players"`
	Fees                    float64         `json:"fees,omitempty"`
	Weight                  float64         `json:"weight,omitempty"`
	MaxSetPoint             *string         `json:"maximum_set_points"`
	CreatedAt               time.Time       `json:"created_at,omitempty"`
	UpdatedAt               time.Time       `json:"updated_at,omitempty"`
	TeamCount               int             `json:"team_count"`
	NumberOfOvers           int             `json:"number_of_overs"`
	BallType                string          `json:"ball_type"`
	MaxRegistrationsReached bool            `form:"max_registrations_reached" json:"max_registrations_reached,omitempty"`
	IsTshirtSizeRequired    bool            `form:"is_tshirt_size_required" json:"is_tshirt_size_required,omitempty"`
	Participated            bool            `json:"participated"`
	CycleType               string          `json:"cycle_type"`
	DistanceCategory        int             `json:"distance_category"`
	MinPlayer               int             `json:"min_player"`
	MaxPlayer               int             `json:"max_player"`
	GameName                string          `json:"game_name"`
}

// EventGameType represents a specific game type (e.g., singles, doubles)
// mapped to an event game instance.
//
// Fields:
//   - ID, TypeID: Internal database IDs.
//   - EncId, TypeEncId: Encrypted external IDs.
//   - Name, Slug: Display and URL-friendly identifier.
type EventGameType struct {
	EncId                   string     `json:"id"`
	ID                      int64      `json:"-"`
	TypeEncId               string     `json:"type_id"`
	TypeID                  int64      `json:"-"`
	Name                    string     `json:"name"`
	Slug                    string     `json:"slug"`
	AgeGroups               []Category `json:"age_groups"`
	ActiveTeamCount         int        `json:"active_team_count"`
	MaxRegistrationsReached bool       `form:"max_registrations_reached" json:"max_registrations_reached,omitempty"`
	Participated            bool       `json:"participated"`
	MinPlayer               int        `json:"min_player"`
	MaxPlayer               int        `json:"max_player"`
	GameName                string     `json:"game_name"`
}

// GameTypeById is a simplified structure used for retrieving game type data by ID.
type GameTypeById struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Category struct {
	EncId                   string `json:"ag_id"`
	Id                      int64  `json:"-"`
	EventHasGameTypeEncId   string `json:"event_has_game_type_id"`
	EventHasGameTypeId      int64  `json:"-"`
	Category                string `json:"category"`
	MinAge                  int    `json:"minage"`
	MaxAge                  int    `json:"maxage"`
	Slug                    string `json:"slug"`
	ActiveTeamCount         int    `json:"active_team_count"`
	MaxRegistrationsReached bool   `form:"max_registrations_reached" json:"max_registrations_reached,omitempty"`
	Participated            bool   `json:"participated"`
	MinPlayer               int    `json:"min_player"`
	MaxPlayer               int    `json:"max_player"`
}

// ListGame provides a compact format for listing games,
// useful in dropdowns or simple views.
type ListGame struct {
	EncId    string `json:"id"`
	ID       int64  `json:"-"`
	GameName string `json:"game_name"`
	Slug     string `json:"slug"`
	Category string `json:"category"`
}

// GameConfig holds a game and all its supported types,
// used during game selection or event configuration.
type GameConfig struct {
	EncId     string          `json:"id"`
	ID        int64           `json:"-"`
	GameName  string          `json:"game_name"`
	Types     []GameType      `json:"types"`
	AgeGroups []ShortAgeGroup `json:"age_groups"`
}

// EventData encapsulates event metadata along with the creator info.
type EventData struct {
	Event     Event `json:"event"`
	CreatedBy struct {
		ID    int64  `json:"-"`
		EncId string `json:"id"`
		Role  string `json:"role"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"created_by"`
}

// EventFolderPath defines the root path for event-related uploads.
const EventFolderPath = "public/event"

// CreateEvent creates a new event in the database and returns the created event object.
// It processes the event details such as the name, dates, location, fees, and registration period.
// It also handles logo path generation, slug creation, and ensures proper date formats.
// Additionally, it creates associated event games.
//
// Params:
//   - event (*Event): The event object containing details for creation.
//
// Returns:
//   - (*Event): The created event object with populated ID and other attributes.
//   - error: If there is an error during the event creation process, an error is returned.
func CreateEvent(event *Event, tx *sql.Tx) (*Event, error) {
	// Set default values for CreatedAt and UpdatedAt if not provided
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now() // Set to current time if not set
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = time.Now() // Set to current time if not set
	}

	// Generate a slug from the event name (used for URL-friendly naming)
	slug := utils.GenerateSlug(event.Name)
	// Set the full path for the event logo image
	event.LogoPath = filepath.Join(EventFolderPath, event.LogoPath)

	// Parse 'FromDate' and 'ToDate' for the event (expecting the format "YYYY-MM-DD")
	fromDate, err1 := time.Parse("2006-01-02", event.FromDate)
	if err1 != nil {
		// Error parsing 'FromDate'
		fmt.Println("Error parsing time:", err1)
		return nil, err1
	}
	toDate, err1 := time.Parse("2006-01-02", event.ToDate)
	if err1 != nil {
		// Error parsing 'ToDate'
		// fmt.Println("Error parsing time:", err1)
		return nil, err1
	}

	// Parse 'LastRegistrationDate' if provided (optional)
	var lastRegDate *time.Time
	if event.LastRegistrationDate != "" {
		parsedDate, err := time.Parse("2006-01-02", event.LastRegistrationDate)
		if err != nil {
			// Invalid 'LastRegistrationDate' format
			return nil, fmt.Errorf("invalid last_registration_date format: %v", err)
		}
		lastRegDate = &parsedDate
	}

	// Parse 'StartRegistrationDate' if provided (optional)
	var startRegDate *time.Time
	if event.StartRegistrationDate != "" {
		parsedDate, err := time.Parse("2006-01-02", event.StartRegistrationDate)
		if err != nil {
			// Invalid 'StartRegistrationDate' format
			return nil, fmt.Errorf("invalid start_registration_date format: %v", err)
		}
		startRegDate = &parsedDate
	}

	// SQL query to insert the new event into the 'events' table
	query := `INSERT INTO events (
			name, created_by_id, created_by_role, from_date, to_date, last_registration_date, start_registration_date,
			state_id, city_id, venue, fees, facebook_link, instagram_link,
			linkedin_link, google_map_link, about, slug, logo, level_of_competition_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19) RETURNING id, logo`

	var eventID int64
	var logoPath sql.NullString

	// Execute the SQL query and retrieve the event ID and logo path
	err := tx.QueryRow(query,
		event.Name,
		event.CreatedById,
		event.CreatedByRole,
		fromDate,
		toDate,
		lastRegDate,
		startRegDate,
		event.StateId,
		event.CityId,
		event.Venue,
		event.Fees,
		event.FacebookLink,
		event.InstagramLink,
		event.LinkedinLink,
		event.GoogleMapLink,
		event.About,
		slug,
		event.Logo,
		event.LevelOfCompetition,
	).Scan(&eventID, &logoPath)

	if err != nil {
		// Error during event creation query execution
		fmt.Printf("Error during database query: %v\n", err)
		return nil, fmt.Errorf("unable to create event: %v", err)
	}

	// After creating the event, create associated games for the event
	for i := range event.Games {
		// Set the EventID for each game
		event.Games[i].EventID = int(eventID)
		// Create the game associated with the event
		event.Games[i], err = CreateEventGame(event.Games[i], tx)
		if err != nil {
			// If an error occurs during game creation, return it
			return nil, err
		}
	}

	// Set the final Event ID after creation
	event.ID = eventID
	return event, nil
}

// CreateEventGame creates a new event game in the database and returns the created event game object.
// It handles the creation of the event game, including its details such as team size, tournament type, duration,
// and other attributes. Additionally, it creates the game types associated with the event game.
//
// Params:
//   - EventGame (EventGame): The event game object containing details for creation.
//
// Returns:
//   - EventGame (EventGame): The created event game object with populated ID and other attributes.
//   - error: If there is an error during the event game creation process, an error is returned.
func CreateEventGame(EventGame EventGame, tx *sql.Tx) (EventGame, error) {
	var gameName string
	err := tx.QueryRow("SELECT game_name FROM games WHERE id = $1", EventGame.GameID).Scan(&gameName)
	if err != nil {
		return EventGame, fmt.Errorf("failed to fetch game name: %v", err)
	}

	teamGames := map[string]bool{
		"football":   true,
		"handball":   true,
		"basketball": true,
		"hockey":     true,
		"volleyball": true,
		"tug of war": true,
		"kho-kho":    true,
		"kabaddi":    true,
		"cricket":    true,
	}

	if teamGames[strings.ToLower(gameName)] {
		if EventGame.MinPlayer < 1 {
			return EventGame, fmt.Errorf("min_player must be at least 1 for %s", gameName)
		}
		if EventGame.MaxPlayer < EventGame.MinPlayer {
			return EventGame, fmt.Errorf("max_player must be greater than or equal to min_player for %s", gameName)
		}
	}

	if EventGame.CreatedAt.IsZero() {
		EventGame.CreatedAt = time.Now()
	}
	if EventGame.UpdatedAt.IsZero() {
		EventGame.UpdatedAt = time.Now()
	}

	var ballType *string
	if EventGame.BallType != "" {
		temp := strings.ToLower(EventGame.BallType)
		ballType = &temp
	}
	var cycleType *string
	if EventGame.CycleType != "" {
		temp := strings.ToLower(EventGame.CycleType)
		cycleType = &temp
	}

	var EventGameID int
	query := `INSERT INTO event_has_games
        (event_id, game_id, team_size, type_of_tournament, duration,
         max_registration, sets, fees, weight, maximum_set_points,
         is_tshirt_size_required, number_of_overs, ball_type,
         cycle_type, distance_category,
         created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
        RETURNING id`

	err = tx.QueryRow(query,
		EventGame.EventID, EventGame.GameID, EventGame.TeamSize,
		EventGame.TypeOfTournament, EventGame.Duration,
		EventGame.MaxRegistration, EventGame.Sets,
		EventGame.Fees, EventGame.Weight, EventGame.MaxSetPoint,
		EventGame.IsTshirtSizeRequired, EventGame.NumberOfOvers,
		ballType, cycleType, EventGame.DistanceCategory,
		EventGame.CreatedAt, EventGame.UpdatedAt,
	).Scan(&EventGameID)

	if err != nil {
		return EventGame, fmt.Errorf("unable to create EventGame: %v", err)
	}

	for i := range EventGame.Type {
		for j := range EventGame.Type[i].AgeGroups {
			EventGame.Type[i].AgeGroups[j].Id, err = CreateEventHasGameType(EventGameID, int(EventGame.Type[i].TypeID), int(EventGame.Type[i].AgeGroups[j].Id),
				EventGame.Type[i].Name, gameName, EventGame.MinPlayer, EventGame.MaxPlayer, tx)
			if err != nil {
				return EventGame, err
			}
		}
	}

	EventGame.ID = int64(EventGameID)
	return EventGame, nil
}

// GetEventLogoById retrieves the logo path for an event by its ID.
// If the event is found, it returns the logo path. If the event is not found, it returns an empty string.
//
// Params:
//   - eventID (int): The ID of the event for which the logo path is to be retrieved.
//
// Returns:
//   - string: The logo path of the event if found, otherwise an empty string.
//   - error: If an error occurs while querying the database, an error is returned.
func GetEventLogoById(eventID int) (string, error) {
	// Variable to store the retrieved logo path
	var logoPath string

	// SQL query to select the logo path from the events table based on event ID
	query := `SELECT logo FROM events WHERE id = $1`

	// Execute the query and scan the result into logoPath
	err := database.DB.QueryRow(query, eventID).Scan(&logoPath)

	if err != nil {
		// If no rows are found (i.e., event does not exist), return an empty string
		if err == sql.ErrNoRows {
			return "", nil
		}
		// For other errors, return a detailed error message
		return "", fmt.Errorf("failed to fetch event logo: %v", err)
	}

	// Return the logo path if found
	return logoPath, nil
}

// CreateEventHasGameType inserts a new entry into the event_has_game_types table linking an event game to a game type.
// It returns the ID of the newly created event_has_game_type entry.
//
// Params:
//   - eventhasgame_id (int): The ID of the event_has_game entry (which represents an event's game).
//   - type_id (int): The ID of the game type to be linked with the event game.
//
// Returns:
//   - int64: The ID of the newly created event_has_game_type entry.
//   - error: If an error occurs while inserting into the database, an error is returned.
func CreateEventHasGameType(eventhasgame_id int, type_id int, age_group_id int, typeName string, gameName string, minPlayer int, maxPlayer int, tx *sql.Tx) (int64, error) {
	var eventHadGameTypeId int64

	lowerGameName := strings.ToLower(strings.TrimSpace(gameName))
	lowerTypeName := strings.ToLower(strings.TrimSpace(typeName))

	switch {
	case strings.Contains(lowerTypeName, "singles"):
		minPlayer, maxPlayer = 1, 1
	case strings.Contains(lowerTypeName, "doubles"):
		minPlayer, maxPlayer = 2, 2
	case strings.HasPrefix(lowerGameName, "4x"):
		minPlayer, maxPlayer = 4, 4
	}

	query := `INSERT INTO event_has_game_types
		(event_has_game_id, game_type_id, age_group_id, min_player, max_player)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err := tx.QueryRow(query, eventhasgame_id, type_id, age_group_id, minPlayer, maxPlayer).
		Scan(&eventHadGameTypeId)
	if err != nil {
		return 0, fmt.Errorf("unable to create eventGameType: %v", err)
	}

	return eventHadGameTypeId, nil
}

// GetGameTypesByEvent fetches the game types linked to a specific event using the event's ID.
// It returns a list of GameType objects associated with the event.
//
// Params:
//   - eventID (int): The ID of the event whose linked game types are to be fetched.
//
// Returns:
//   - []GameType: A slice of GameType objects representing the game types associated with the event.
//   - error: If an error occurs during the database query or scanning process, an error is returned.
func GetGameTypesByEvent(eventID int) ([]GameType, error) {
	// SQL query to fetch the game types linked to the provided event ID
	query := `
	SELECT gt.id, gt.name
	FROM games_types gt
	JOIN event_has_game_types egt ON gt.id = egt.event_has_game_id
	WHERE egt.game_type_id = $1
	`
	// Execute the query to fetch the rows from the database
	rows, err := database.DB.Query(query, eventID)
	if err != nil {
		// If an error occurs during the query execution, return an error
		return nil, fmt.Errorf("unable to fetch game types: %v", err)
	}
	// Ensure that rows are closed after the function completes
	defer rows.Close()

	// Initialize an empty slice to store the fetched game types
	var gameTypes []GameType

	// Iterate through the result rows
	for rows.Next() {
		var gameType GameType
		// Scan the row values into the gameType struct
		if err := rows.Scan(&gameType.ID, &gameType.Name); err != nil {
			// If an error occurs during scanning, return an error
			return nil, fmt.Errorf("error scanning game type: %v", err)
		}
		// Append the gameType to the gameTypes slice
		gameTypes = append(gameTypes, gameType)
	}

	// Return the slice of game types after all rows have been processed
	return gameTypes, nil
}

func UpdateEvent(CurrentEvent *Event) error {
	var originalCreatorID int
	err := database.DB.QueryRow(`SELECT created_by_id FROM events WHERE id = $1`, CurrentEvent.ID).Scan(&originalCreatorID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("event with ID %d does not exist", CurrentEvent.ID)
	}
	if err != nil {
		return err
	}

	CurrentEvent.UpdatedAt = time.Now()
	CurrentEvent.LogoPath = CurrentEvent.Logo

	updateQuery := `UPDATE events SET
		name=$1, slug=$2, from_date=$3, to_date=$4, state_id=$5,
		city_id=$6, venue=$7, fees=$8, facebook_link=$9, instagram_link=$10,
		linkedin_link=$11, logo=$12, about=$13, updated_at=$14, google_map_link=$15,
		start_registration_date=$16, last_registration_date=$17, level_of_competition_id=$18
		WHERE id=$19`

	_, err = database.DB.Exec(updateQuery,
		CurrentEvent.Name, CurrentEvent.Slug,
		CurrentEvent.FromDate, CurrentEvent.ToDate,
		CurrentEvent.StateId, CurrentEvent.CityId,
		CurrentEvent.Venue, CurrentEvent.Fees,
		CurrentEvent.FacebookLink, CurrentEvent.InstagramLink, CurrentEvent.LinkedinLink,
		CurrentEvent.LogoPath, CurrentEvent.About,
		CurrentEvent.UpdatedAt, CurrentEvent.GoogleMapLink,
		CurrentEvent.StartRegistrationDate, CurrentEvent.LastRegistrationDate,
		CurrentEvent.LevelOfCompetition,
		CurrentEvent.ID,
	)
	if err != nil {
		return fmt.Errorf("unable to update event: %v", err)
	}
	return nil
}

// DeleteEventHasGames deletes a game association from the event and its related game types.
// This function performs the following:
// 1. Retrieves all the associated game types linked to the given gameId.
// 2. Deletes each game type association.
// 3. Deletes the game association itself from the event_has_games table.
//
// Params:
//   - gameId (int): The ID of the game to delete associations for.
//
// Returns:
//   - error: If any error occurs during the deletion process (e.g., failed database query or deletion).
func DeleteEventHasGames(gameId int) error {
	// Query to fetch all game types associated with the gameId from event_has_game_types
	query := `SELECT id FROM event_has_game_types where event_has_game_id=$1`
	rows, err := database.DB.Query(query, gameId)
	if err != nil {
		// If there's an error querying the game types, return it
		return err
	}

	// Iterate through each row to fetch game type IDs and delete their associations
	for rows.Next() {
		var typeId int
		err = rows.Scan(&typeId)
		if err != nil {
			// If scanning the game type ID fails, return the error
			return err
		}
		// Call DeleteEventHasGameTypes to delete each game type association
		err = DeleteEventHasGameTypes(typeId)
		if err != nil {
			// If deleting the game type association fails, return the error
			return err
		}
	}

	// After deleting all game type associations, delete the game itself from event_has_games
	query = `DELETE FROM event_has_games WHERE id=$1`
	_, err = database.DB.Exec(query, gameId)
	if err != nil {
		// If there's an error deleting the game, return it
		return err
	}

	// Return nil if all operations (deletions) are successful
	return nil
}

// DeleteEventHasGameTypes deletes a game type association from the event_has_game_types table.
// This function removes the specified game type entry based on the provided typeId.
//
// Params:
//   - typeId (int): The ID of the game type to delete from the event_has_game_types table.
//
// Returns:
//   - error: If an error occurs during the deletion process (e.g., failed database query).
func DeleteEventHasGameTypes(typeId int) error {
	// Query to delete the game type association by its ID from event_has_game_types
	query := `DELETE FROM event_has_game_types WHERE id=$1`

	// Execute the delete query
	_, err := database.DB.Exec(query, typeId)
	if err != nil {
		// If an error occurs while executing the query, return the error
		return err
	}

	// Return nil if the deletion is successful (no error)
	return nil
}

// GetEventStatusByID retrieves the status of an event by its event ID.
// This function queries the database to fetch the event status for the given event_id.
//
// Params:
//   - event_id (int64): The ID of the event whose status is to be retrieved.
//
// Returns:
//   - string: The status of the event.
//   - error: An error if the status cannot be fetched (e.g., database error or event not found).
func GetEventStatusByID(event_id int64) (string, error) {
	var status string

	// SQL query to select the status from the 'events' table based on event_id
	query := `SELECT status FROM events WHERE id = $1`

	// Execute the query and scan the result into the 'status' variable
	err := database.DB.QueryRow(query, event_id).Scan(&status)
	if err != nil {
		// Log the error if there is an issue fetching the status from the database
		log.Printf("Error fetching status : %v\n", err)

		// Return an error message indicating that fetching the status failed
		return "", fmt.Errorf("failed to fetch status")
	}

	// Return the fetched status and nil if no error occurred
	return status, nil
}

// UpdateEventStatusByID updates the status of an event in the database using the event ID.
// This function executes a query to update the event's status based on the provided event_id and status.
//
// Params:
//   - event_id (int64): The ID of the event to update.
//   - status (string): The new status to be assigned to the event.
//
// Returns:
//   - error: An error if the status update fails, otherwise nil.
func UpdateEventStatusByID(event_id int64, status string) error {
	// SQL query to update the event status where event_id matches
	query := `UPDATE events SET status=$1 WHERE id=$2`

	// Execute the query, passing the status and event_id as arguments
	_, err := database.DB.Exec(query, status, event_id)
	if err != nil {
		// Log error if status update fails
		log.Printf("Error updating status. err : %v", err)
		return fmt.Errorf("failed to update status")
	}
	return nil
}

// GetEvents retrieves a list of events based on various filters, including search terms, date range, and status.
// The function uses dynamic query generation to handle filtering and sorting based on the provided parameters.
//
// Params:
//   - search (string): A search term to filter events by name, venue, city, or game name.
//   - sort (string): The column by which to sort the events (e.g., 'name', 'created_at').
//   - dir (string): The direction of sorting ('ASC' or 'DESC').
//   - status (string): A comma-separated string of event statuses to filter by (e.g., 'Active', 'Upcoming').
//   - from_date (string): The starting date to filter events (in 'YYYY-MM-DD' format).
//   - to_date (string): The ending date to filter events (in 'YYYY-MM-DD' format).
//   - limit (int64): The maximum number of events to return.
//   - offset (int64): The number of events to skip (for pagination).
//   - user_id (*int64): The ID of the user (used for filtering events the user has created or participated in).
//   - isOrganized (bool): If true, filter for events organized by the user.
//   - isParticipated (bool): If true, filter for events the user has participated in.
//
// Returns:
//   - totalRecords (int): The total number of records matching the filters (for pagination).
//   - events ([]Event): A list of events matching the filters.
//   - error: An error if the query fails or there's an issue retrieving the events.
func GetEvents(search, sort, dir, status, from_date, to_date string, limit, offset int64, user_id *int64, isOrganized, isParticipated bool) (int, []Event, error) {
	var events []Event
	args := []interface{}{limit, offset}
	query := `
			SELECT
			e.id,
			e.name,
			e.created_by_id,
			e.created_by_role,
			e.from_date,
			e.to_date,
			e.start_registration_date,
			e.last_registration_date,
			e.state_id,
			e.city_id,
			c.city AS city_name,
			e.venue,
			e.fees,
			e.about,
			e.facebook_link,
			e.instagram_link,
			COALESCE(e.linkedin_link, '') AS linkedin_link,
			e.logo,
			e.status,
			e.created_at,
			e.updated_at,
			COALESCE(STRING_AGG(g.game_name, ', '), '') AS game_names,
			COUNT(e.id) OVER() AS totalrecords
			FROM events e
			LEFT JOIN event_has_games eg ON e.id = eg.event_id
			LEFT JOIN games g ON eg.game_id = g.id
			LEFT JOIN cities c ON e.city_id::TEXT = c.id::TEXT  -- Casting both columns to TEXT to avoid type mismatch
			`

	// Apply filter for events user has participated in
	if isParticipated {
		query += `
				INNER JOIN event_has_users ehu ON e.id = ehu.event_id
				`
	}

	// Apply filter for events user has organized, joining with the organization_has_score_moderator table
	if isOrganized {
		query += `LEFT JOIN organization_has_score_moderator osm ON osm.organization_id = e.created_by_id`
	}

	// Filter conditions start
	query += " WHERE 1=1"

	//NOTE - do this
	// Apply Organized Filter: Filter by events the user has organized or is the moderator for
	// if isOrganized && user_id != nil {
	// 	query += fmt.Sprintf(" AND e.created_by_id = $%d OR osm.moderator_id=$%d ", len(args)+1, len(args)+1)
	// 	args = append(args, *user_id)
	// }
	// if isOrganized && user_id != nil {
	// 	query += fmt.Sprintf(" AND e.created_by_id = $%d OR (osm.moderator_id=$%d AND osm.event_id= e.id AND osm.status='Active')", len(args)+1, len(args)+1)
	// 	args = append(args, *user_id)
	// }
	if isOrganized && user_id != nil {
		query += fmt.Sprintf(" AND (e.created_by_id = $%d OR (osm.moderator_id=$%d AND osm.event_id= e.id AND osm.status='Active'))", len(args)+1, len(args)+1)
		args = append(args, *user_id)
	}

	// Apply Participated Filter: Filter by events the user has participated in
	if isParticipated && user_id != nil {
		query += fmt.Sprintf(" AND ehu.user_id = $%d", len(args)+1)
		args = append(args, *user_id)
	}

	// Filter by event status (e.g., Active, Inactive)
	// Default to 'Active' if no specific status filter is provided
	if isOrganized && user_id != nil {
		query += " AND e.status IN ('Active', 'Inactive')"
	} else {
		query += " AND e.status IN ('Active')"
	}

	// Apply status filter based on provided status argument
	currentDate := time.Now()
	if status != "" {
		switch status {
		case "Live":
			// Filter events that are currently live (active within the date range)
			query += " AND from_date <= $3 AND to_date >= $3"
			args = append(args, currentDate)
		case "Upcoming":
			// Filter events that are upcoming (start date in the future)
			query += " AND from_date > $3"
			args = append(args, currentDate)
		case "Past":
			// Filter events that are in the past (end date before the current date)
			query += " AND to_date < $3"
			args = append(args, currentDate)
		default:
			// If status contains multiple values, split them and filter accordingly
			statusValues := strings.Split(status, ",")
			statusPlaceholders := []string{}
			for _, s := range statusValues {
				statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, strings.TrimSpace(s))
			}
			query += fmt.Sprintf(" AND status IN (%s)", strings.Join(statusPlaceholders, ", "))
		}
	}

	// Filter by date range (from_date and to_date)
	// If both dates are provided, filter events that fall within the range
	if from_date != "" && to_date != "" {
		query += fmt.Sprintf(" AND from_date <= $%d AND to_date >= $%d", len(args)+1, len(args)+2)
		args = append(args, to_date, from_date)
	} else if from_date != "" {
		// If only 'from_date' is provided, filter events that start after 'from_date'
		query += fmt.Sprintf(" AND from_date >= $%d", len(args)+1)
		args = append(args, from_date)
	} else if to_date != "" {
		// If only 'to_date' is provided, filter events that end before 'to_date'
		query += fmt.Sprintf(" AND to_date <= $%d", len(args)+1)
		args = append(args, to_date)
	}

	// Apply search filters: Searching by event name, venue, city, and game name
	if search != "" {
		// Decode '%2C' and '%20' from search query and prepare keywords for searching
		search = strings.Replace(search, "%2C", "", -1)
		keyWords := strings.Split(search, "%20")
		query += " AND ("
		for i, word := range keyWords {
			if i != 0 {
				query += " AND "
			}
			// Search by event name, venue, city, or game name using 'ILIKE' for case-insensitive search
			query += fmt.Sprintf("(e.name ILIKE $%d OR e.venue ILIKE $%d OR c.city ILIKE $%d OR g.game_name ILIKE $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4)
			args = append(args, "%"+word+"%", "%"+word+"%", "%"+word+"%", "%"+word+"%")
		}
		query += ")"
	}

	// Group and order the query results by event details and aggregate games
	query += ` GROUP BY
		e.id, e.name, e.created_by_id, e.created_by_role, e.from_date, e.to_date,
		e.state_id, e.city_id, c.city, e.venue, e.fees, e.about,
		e.facebook_link, e.instagram_link, e.linkedin_link,
		e.logo, e.status, e.created_at, e.updated_at`
	// Sort by the specified column and direction, then apply pagination with LIMIT and OFFSET
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $1 OFFSET $2", sort, dir)

	// Execute the query and fetch event data from the database
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		// Log the error if the query fails
		fmt.Printf("Error querying events : %v\n", err)
		return 0, nil, err
	}
	defer rows.Close()

	// Initialize variables for the city name and game names
	var city_name string
	var gameNames string
	totalRecords := 0
	// Iterate through the query results and scan the data into the 'Event' struct
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.CreatedById,
			&event.CreatedByRole,
			&event.FromDate,
			&event.ToDate,
			&event.StartRegistrationDate,
			&event.LastRegistrationDate,
			&event.StateId,
			&event.CityId,
			&city_name,
			&event.Venue,
			&event.Fees,
			&event.About,
			&event.FacebookLink,
			&event.InstagramLink,
			&event.LinkedinLink,
			&event.LogoPath,
			&event.Status,
			&event.CreatedAt,
			&event.UpdatedAt,
			&gameNames,
			&totalRecords,
		); err != nil {
			// Log the error if there is an issue scanning the row
			fmt.Printf("Error scanning row : %v\n", err)
			return 0, nil, err
		}
		var games []EventGame

		// Fetch the associated games for the event
		if isParticipated {
			games, err = GetEventGameByEventID(event.ID, *user_id)
		} else {
			games, err = GetEventGameByEventID(event.ID)
		}
		if err != nil {
			// Log the error if fetching event games fails
			fmt.Printf("Error fetching event games: %v\n", err)
			return 0, nil, err
		}
		// Retrieve the creator information based on their role (either User or Admin)
		if event.CreatedByRole == "User" {
			// Fetch user details if the creator is a user
			user, err := GetUserByID(int(event.CreatedById))
			if err != nil {
				return 0, nil, err
			}
			event.CreatedByName = user.Name
			event.CreatedByEmail = user.Email
		} else {
			// Fetch admin details if the creator is an admin
			admin, err := GetAdminByID(int(event.CreatedById))
			if err != nil {
				return 0, nil, err
			}
			event.CreatedByName = admin.Name
			event.CreatedByEmail = admin.Email
		}
		event.Games = games
		events = append(events, event)
	}

	// Return the total number of records and the list of events
	return totalRecords, events, nil
}

// GetGamesList retrieves a list of active games from the database.
// It selects game ID, name, and slug, filtering by games that are marked as 'Active'.
func GetGamesList() ([]ListGame, error) {
	var List []ListGame
	query := `SELECT id, game_name, slug
              FROM games
              WHERE status='Active'
              ORDER BY game_name ASC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return List, err
	}
	defer rows.Close()

	for rows.Next() {
		var item ListGame
		err := rows.Scan(&item.ID, &item.GameName, &item.Slug)
		if err != nil {
			return List, err
		}
		item.Category = assignCategory(item.GameName)
		List = append(List, item)
	}
	return List, nil
}

// to assign the games category...example: running, throwing, jumping and other
func assignCategory(gameName string) string {
	gameNameLower := strings.ToLower(gameName)
	switch {
	case strings.Contains(gameNameLower, "sprint") || strings.Contains(gameNameLower, "race") ||
		strings.Contains(gameNameLower, "relay") || strings.Contains(gameNameLower, "hurdles") ||
		strings.Contains(gameNameLower, "marathon") || strings.Contains(gameNameLower, "steeplechase"):
		return "Running"
	case strings.Contains(gameNameLower, "jump"):
		return "Jumping"
	case strings.Contains(gameNameLower, "throw") || strings.Contains(gameNameLower, "put"):
		return "Throwing"
	case strings.Contains(gameNameLower, "freestyle") || strings.Contains(gameNameLower, "backstroke") ||
		strings.Contains(gameNameLower, "butterfly") || strings.Contains(gameNameLower, "medley") ||
		strings.Contains(gameNameLower, "breaststroke"):
		return "Swimming"
	default:
		return "Other"
	}
}

// GetEventsById retrieves the details of an event by its ID, along with related games and creator information.
// It first fetches event details, then fetches games associated with the event, and finally retrieves the creator's information (either user or admin).
func GetEventsById(id int64, userId ...int64) (*Event, error) {
	// Query to fetch event details by ID
	query := `SELECT id, created_by_id, created_by_role, name, from_date, to_date, state_id, city_id, venue, fees, about, facebook_link, instagram_link, linkedin_link, logo, status, created_at, updated_at, last_registration_date ,start_registration_date, google_map_link, level_of_competition_id FROM events WHERE id=$1`
	var levelOfCompetitionId sql.NullInt64
	var event Event
	// Execute query and scan the result into the event struct
	err := database.DB.QueryRow(query, id).Scan(
		&event.ID,
		&event.CreatedById,
		&event.CreatedByRole,
		&event.Name,
		&event.FromDate,
		&event.ToDate,
		&event.StateId,
		&event.CityId,
		&event.Venue,
		&event.Fees,
		&event.About,
		&event.FacebookLink,
		&event.InstagramLink,
		&event.LinkedinLink,
		&event.LogoPath,
		&event.Status,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.LastRegistrationDate,
		&event.StartRegistrationDate,
		&event.GoogleMapLink,
		&levelOfCompetitionId,
	)
	if err != nil {
		// Return error if event doesn't exist
		return nil, fmt.Errorf("Event doesn't exist->%v", err)
	}

	// Check if the event status is "Delete", in which case return an error
	if event.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	var games []EventGame
	// Fetch the games associated with the event
	if len(userId) > 0 {
		games, err = GetEventGameByEventID(event.ID, userId[0])
	} else {
		games, err = GetEventGameByEventID(event.ID)
	}
	if err != nil {
		// Return error if fetching event games fails
		return nil, fmt.Errorf("error fetching event games : %v", err)
	}

	// Assign the fetched games to the event
	event.Games = games

	if levelOfCompetitionId.Valid {
		level_of_competition, err := GetLevelofCompetitionById(levelOfCompetitionId.Int64)
		if err == nil {
			event.LevelOfCompetitionTitle = level_of_competition.Title
		} else {
			event.LevelOfCompetitionTitle = "Not Available"
		}
	} else {
		event.LevelOfCompetitionTitle = "Not Available"
	}

	// Retrieve the creator information based on their role (either User or Admin)
	if event.CreatedByRole == "User" {
		// Fetch user details if the creator is a user
		user, err := GetUserByID(int(event.CreatedById))
		if err != nil {
			return nil, err
		}
		event.CreatedByName = user.Name
		event.CreatedByEmail = user.Email
	} else {
		// Fetch admin details if the creator is an admin
		admin, err := GetAdminByID(int(event.CreatedById))
		if err != nil {
			return nil, err
		}
		event.CreatedByName = admin.Name
		event.CreatedByEmail = admin.Email
	}

	// Return the populated event structure
	return &event, nil
}

// GetEventGameByEventID retrieves all games associated with a specific event ID.
// It performs the following:
// 1. Executes a query that joins event_has_games, event_has_game_types, and games_types.
// 2. Aggregates associated game types using JSON.
// 3. Computes 'active_team_count' (number of active teams registered for each game type).
// 4. Evaluates whether the max registration limit has been reached for each game type.
// 5. Returns a list of EventGame objects populated with detailed type information.
//
// Params:
//   - eventID (int64): The ID of the event for which to fetch all associated games.
//
// Returns:
//   - []EventGame: A list of all games with associated type information for the specified event.
//   - error: If any error occurs during the query, scanning, or JSON unmarshalling.
func GetEventGameByEventID(eventID int64, userId ...int64) ([]EventGame, error) {
	// SQL query to fetch all game details for a given event.
	// Joins with event_has_game_types and games_types to retrieve type data,
	// aggregates game types into a JSON array, and calculates active team counts per type.
	// Also checks if max registration count has been reached for each game type.
	query := `
	SELECT
	g.id,
	g.game_id,
	g.type_of_tournament,
	g.team_size,
	g.duration,
	g.max_registration,
	g.maximum_set_points,
	g.sets,
	g.fees,
	g.is_tshirt_size_required,
	g.number_of_overs,
	g.ball_type,
	g.cycle_type,
	g.distance_category,
	g.created_at,
	g.updated_at,
	-- 👇 team count at the outermost level
    COALESCE((
        SELECT COUNT(*)
        FROM event_has_teams t
        WHERE t.status != 'Delete'
        AND t.event_id = g.event_id
        AND t.game_id = g.game_id
    ), 0) AS team_count,
	json_agg(
		DISTINCT jsonb_build_object(
			'id', gt.id::TEXT,
			'type_id', gt.id::TEXT,
			'name', gt.name,
			'slug', gt.slug,
			'active_team_count', COALESCE((
				SELECT COUNT(*)
				FROM event_has_teams t
				WHERE t.status = 'Active'
				AND t.event_id = g.event_id
				AND t.game_id = g.game_id
				AND t.game_type_id = gt.id
			), 0),
			'age_groups', (
				SELECT json_agg(
					json_build_object(
						'ag_id', ag.id::TEXT,
						'event_has_game_type_id', sub_ehgt.id::TEXT,
						'category', ag.category,
						'min_age', ag.minage,
						'maxage', ag.maxage,
						'slug', ag.slug,
						'min_player', sub_ehgt.min_player,
						'max_player', sub_ehgt.max_player,
						'active_team_count', COALESCE((
							SELECT COUNT(*)
							FROM event_has_teams t
							WHERE t.status = 'Active'
							AND t.event_id = g.event_id
							AND t.game_id = g.game_id
							AND t.game_type_id = gt.id
							AND t.age_group_id= ag.id
						), 0),
						'max_registrations_reached', (
							COALESCE((
								SELECT COUNT(*)
								FROM event_has_teams t
								WHERE t.status = 'Active'
								AND t.event_id = g.event_id
								AND t.game_id = g.game_id
								AND t.game_type_id = gt.id
								AND t.age_group_id= ag.id
							), 0) >= COALESCE(NULLIF(g.max_registration, '')::INT, 0)
						)
					)
				)
				FROM event_has_game_types sub_ehgt
				JOIN age_group ag ON ag.id = sub_ehgt.age_group_id
				WHERE sub_ehgt.event_has_game_id = g.id
				AND sub_ehgt.game_type_id = gt.id
			)
		)
	) AS game_types
	FROM event_has_games g
	JOIN event_has_game_types ehgt ON ehgt.event_has_game_id = g.id
	JOIN games_types gt ON ehgt.game_type_id = gt.id
	WHERE g.event_id = $1
	GROUP BY g.id;
	`

	// Execute the SQL query with the provided eventID
	rows, err := database.DB.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("error querying event games : %v", err)
	}
	defer rows.Close() // Ensure rows are closed once processing is complete

	var games []EventGame // Container for all fetched event games
	for rows.Next() {
		var game EventGame
		var typeJson string
		var nullBallType sql.NullString
		var nullCycleType sql.NullString
		game.EventID = int(eventID) // Assign the event ID to the game

		// Populate the game struct fields and the aggregated game types JSON string
		err := rows.Scan(
			&game.ID,
			&game.GameID,
			&game.TypeOfTournament,
			// &game.NumberOfPlayers,
			&game.TeamSize,
			&game.Duration,
			&game.MaxRegistration,
			&game.MaxSetPoint,
			&game.Sets,
			&game.Fees,
			&game.IsTshirtSizeRequired,
			&game.NumberOfOvers,
			&nullBallType,
			&nullCycleType,
			// &game.MinPlayer,
			// &game.MaxPlayer,
			&game.DistanceCategory,
			&game.CreatedAt,
			&game.UpdatedAt,
			&game.TeamCount,
			&typeJson,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning event games : %v", err)
		}

		if nullBallType.Valid {
			game.BallType = nullBallType.String
		}

		if nullCycleType.Valid {
			game.CycleType = nullCycleType.String
		}

		// Parse the JSON array of game types into the Type field
		err = json.Unmarshal([]byte(typeJson), &game.Type)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling typeJson : %v", err)
		}
		// Initialize MaxRegistrationsReached as true; will be overwritten if any type hasn't reached the cap
		game.MaxRegistrationsReached = true
		for i := range game.Type {
			game.Type[i].MaxRegistrationsReached = true
			// Convert string-based IDs back to int64
			game.Type[i].ID, err = strconv.ParseInt(game.Type[i].EncId, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing int : %v", err)
			}
			game.Type[i].TypeID, err = strconv.ParseInt(game.Type[i].TypeEncId, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing int : %v", err)
			}
			for j := range game.Type[i].AgeGroups {
				game.Type[i].AgeGroups[j].Id, err = strconv.ParseInt(game.Type[i].AgeGroups[j].EncId, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing int : %v", err)
				}
				game.Type[i].AgeGroups[j].EncId = ""

				game.Type[i].AgeGroups[j].EventHasGameTypeId, err = strconv.ParseInt(game.Type[i].AgeGroups[j].EventHasGameTypeEncId, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing int : %v", err)
				}
				game.Type[i].AgeGroups[j].EventHasGameTypeEncId = ""

				game.Type[i].MaxRegistrationsReached = (game.Type[i].MaxRegistrationsReached && game.Type[i].AgeGroups[j].MaxRegistrationsReached)
			}
			// Clear encoded fields post-conversion to maintain clean output
			game.Type[i].EncId = ""
			game.Type[i].TypeEncId = ""

			// Set game's max registration flag based on each type's flag
			game.MaxRegistrationsReached = (game.MaxRegistrationsReached && game.Type[i].MaxRegistrationsReached)
		}

		// Append the game with enriched data to the final list
		games = append(games, game)
	}

	if len(userId) > 0 {
		query = `
		SELECT eht.game_id, eht.game_type_id, eht.age_group_id
			FROM event_has_users ehu
			JOIN event_has_teams eht on ehu.event_has_team_id= eht.id
			WHERE ehu.user_id= $1 AND eht.event_id=$2
		`
		// Execute the SQL query with the provided eventID
		rows2, err := database.DB.Query(query, userId[0], eventID)
		if err != nil {
			return nil, fmt.Errorf("error querying participated games : %v", err)
		}
		defer rows2.Close() // Ensure rows are closed once processing is complete
		for rows2.Next() {
			var gameId, typeId, ageGroupId int64
			err := rows2.Scan(&gameId, &typeId, &ageGroupId)

			if err != nil {
				return nil, fmt.Errorf("error scanning participated games : %v", err)
			}
			for i, game := range games {
				if gameId != game.GameID {
					continue
				}
				game.Participated = true
				for j, gameType := range game.Type {
					if typeId != gameType.TypeID {
						continue
					}
					gameType.Participated = true
					for k := range gameType.AgeGroups {
						if ageGroupId != gameType.AgeGroups[k].Id {
							continue
						}
						gameType.AgeGroups[k].Participated = true
					}
					game.Type[j] = gameType
				}
				games[i] = game
			}
		}
	}

	// Return the complete list of games with types for the event
	return games, nil
}

// GetEventGameByEventAndGame retrieves a specific event game by eventID and gameID.
// It performs the following steps:
// 1. Queries the database to fetch the details of the event game, including tournament type, number of players, team size, etc.
// 2. Retrieves the associated game types for the event game.
// 3. Returns the event game along with its associated types.
//
// Params:
//   - EventID (int64): The ID of the event to fetch the game for.
//   - GameID (int64): The ID of the game to fetch for the event.
//
// Returns:
//   - EventGame: The game and its associated details for the specified event and game.
//   - error: If any error occurs during the fetching process (e.g., failed database query or scan).
func GetEventGameByEventAndGame(EventID int64, GameID int64) (EventGame, error) {
	var Game EventGame
	var nullBallType sql.NullString
	var nullCycleType sql.NullString
	// SQL query to fetch event game details for a specific event and game by their IDs
	query := `SELECT
	id,
	game_id,
	type_of_tournament,
	number_of_players,
	team_size,
	duration,
	max_registration,
	sets,
	is_tshirt_size_required,
	number_of_overs,
	ball_type,
	cycle_type,
	distance_category,
	min_player,
	max_player,
	created_at,
	updated_at
	FROM event_has_games WHERE game_id=$1 AND event_id=$2`

	// Execute the query and scan the result into the Game struct
	err := database.DB.QueryRow(query, GameID, EventID).Scan(
		&Game.ID,
		&Game.GameID,
		&Game.TypeOfTournament,
		&Game.NumberOfPlayers,
		&Game.TeamSize,
		&Game.Duration,
		&Game.MaxRegistration,
		&Game.Sets,
		&Game.IsTshirtSizeRequired,
		&Game.NumberOfOvers,
		&nullBallType,
		&nullCycleType,
		&Game.DistanceCategory,
		&Game.CreatedAt,
		&Game.UpdatedAt,
		&Game.MinPlayer,
		&Game.MaxPlayer,
	)
	if err != nil {
		return Game, fmt.Errorf("error querying event Game : %v", err)
	}

	if nullBallType.Valid {
		Game.BallType = nullBallType.String
	}
	// Fetch the associated game types for the specific event game
	var gameTypes []EventGameType
	gameTypes, err = GetEventGameTypesByGameID(Game.ID)
	// fmt.Println("Game:", Game.ID, "\ntypes:", gameTypes) // Debugging line (currently commented out)

	if err != nil {
		return Game, fmt.Errorf("error fetching event Game has types : %v", err)
	}

	// Assign the fetched game types to the Game struct
	Game.Type = gameTypes
	return Game, nil
}

// GetEventGameTypesByGameID retrieves the game types associated with a specific event game.
// It performs the following:
// 1. Queries the database to fetch the types of games linked to the given event game ID.
// 2. Returns the list of game types for the event game.
//
// Params:
//   - eventGameID (int64): The ID of the event game to fetch game types for.
//
// Returns:
//   - []EventGameType: The list of game types associated with the event game.
//   - error: If any error occurs during the fetching process (e.g., failed database query or scan).
func GetEventGameTypesByGameID(eventGameID int64) ([]EventGameType, error) {
	// SQL query to fetch the game types for a specific event game
	query := `SELECT eht.id, gt.id, gt.name, gt.slug, ag.id as age_group_id, ag.category
		FROM event_has_game_types eht
		JOIN games_types gt ON eht.game_type_id = gt.id
		LEFT JOIN age_group ag ON eht.age_group_id = ag.id
		WHERE eht.event_has_game_id = $1 AND gt.status IN ('Active', 'Inactive');`

	// Execute the query to retrieve game types for the event game
	rows, err := database.DB.Query(query, eventGameID)
	if err != nil {
		return nil, fmt.Errorf("error querying event game types: %v", err)
	}
	defer rows.Close() // Ensure rows are closed once processing is complete

	var gameTypes []EventGameType
	// Iterate over the rows and scan the results into the gameTypes slice
	for rows.Next() {
		var gameType EventGameType
		err := rows.Scan(&gameType.ID, &gameType.TypeID, &gameType.Name, &gameType.Slug)
		if err != nil {
			return nil, fmt.Errorf("error scanning event game types: %v", err)
		}
		gameTypes = append(gameTypes, gameType)
	}
	return gameTypes, nil
}

// DeleteEventByID deletes an event by its ID.
// The function performs the following steps:
// 1. Checks if the event exists and its status.
// 2. Marks the event as "Delete" in the database (soft delete).
// 3. Returns the updated event information.
//
// Params:
//   - id (int64): The ID of the event to delete.
//
// Returns:
//   - Event: The updated event with the status set to "Delete".
//   - error: If any error occurs during the deletion process (e.g., event not found or failed database query).
func DeleteEventByID(id int64) (*Event, error) {
	// Check if the event record exists in the database
	checkQuery := `SELECT id, status FROM events WHERE id = $1`
	var event Event

	// Execute the check query to see if the event exists
	err := database.DB.QueryRow(checkQuery, id).Scan(&event.ID, &event.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching event: %v", err)
	}

	// If the event is already deleted, return an error
	if event.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Update the event's status to "Delete" (soft delete)
	deleteQuery := `UPDATE events SET status = 'Delete', updated_at = $1 WHERE id = $2`
	_, err = database.DB.Exec(deleteQuery, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("error deleting events: %v", err)
	}

	// Update the event status and return the updated event
	event.Status = "Delete"
	event.UpdatedAt = time.Now()

	return &event, nil
}

func GetAllModeratorsForEvent(EventId int64) ([]int64, error) {
	query := `
	SELECT json_agg(u.id)
	FROM users u
	JOIN organization_has_score_moderator osm ON osm.moderator_id=u.id
	JOIN events e ON e.id= osm.event_id
	WHERE e.id=$1
`
	var jsonResult []byte
	err := database.DB.QueryRow(query, EventId).Scan(&jsonResult)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var ids []int64
	if jsonResult == nil {
		ids = []int64{}
	} else {
		err = json.Unmarshal(jsonResult, &ids)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal team ids: %w", err)
		}
	}

	return ids, nil
}
