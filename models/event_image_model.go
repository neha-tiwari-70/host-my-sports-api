package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/database"
)

// EventImage represents the structure that links an image to a specific event.
// Fields:
//   - EventID (int): The ID of the associated event.
//   - ImagePath (string): The path or URL to the event's image.
type EventImage struct {
	EventID   int    `json:"event_id"`
	ImagePath string `json:"image_path"`
}

// GetEventByID retrieves the name of an event by its unique ID.
//
// This function performs the following:
// 1. Executes a SELECT query to fetch the event_name from the event_details table.
// 2. Returns the event name if found.
// 3. Returns an error if no event is found or a database query fails.
//
// Params:
//   - eventID (int): The ID of the event to fetch.
//
// Returns:
//   - *Event: A pointer to the Event struct containing the event name.
//   - error: If any error occurs (e.g., event not found or database failure).
func GetEventByID(eventID int) (*Event, error) {
	var event Event
	// query := `SELECT event_name FROM public.event_details WHERE event_id = $1`
	/*query := `SELECT * FROM public.events WHERE id = $1`
	err := database.DB.QueryRow(query, eventID).Scan(&event.Name)*/
	query := `SELECT id, name, from_date, to_date, state_id, city_id, venue, fees, facebook_link, instagram_link, linkedin_link, logo, about, updated_at, google_map_link, last_registration_date, start_registration_date FROM public.events WHERE id = $1`
	err := database.DB.QueryRow(query, eventID).Scan(
		&event.ID,
		&event.Name,
		&event.FromDate,
		&event.ToDate,
		&event.StateId,
		&event.CityId,
		&event.Venue,
		&event.Fees,
		&event.FacebookLink,
		&event.InstagramLink,
		&event.LinkedinLink,
		&event.Logo,
		&event.About,
		&event.UpdatedAt,
		&event.GoogleMapLink,
		&event.LastRegistrationDate,
		&event.StartRegistrationDate,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found")
		}
		return nil, err
	}
	return &event, nil
}

// GetAllEventImages retrieves all event image mappings from the database.
//
// This function performs the following:
// 1. Executes a SELECT query to fetch all event_id and image entries from the event_has_image table.
// 2. Iterates through the result rows and scans each into an EventImage struct.
// 3. Accumulates the results in a slice and returns them.
// 4. Returns an error if the query fails or scanning/iteration encounters an issue.
//
// Returns:
//   - []EventImage: A slice of EventImage structs containing the event ID and image path.
//   - error: If any error occurs during the database interaction or row scanning.
func GetAllEventImages() ([]EventImage, error) {
	var eventImages []EventImage

	query := `SELECT event_id, image FROM event_has_image`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch event images: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventImage EventImage
		if err := rows.Scan(&eventImage.EventID, &eventImage.ImagePath); err != nil {
			return nil, fmt.Errorf("error scanning event image: %v", err)
		}
		eventImages = append(eventImages, eventImage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over event images: %v", err)
	}

	return eventImages, nil
}
