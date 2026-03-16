package models

import (
	"database/sql"
	"fmt"
	"sports-events-api/crypto"
)

func UpdateMatchTeams(tx *sql.Tx, matchID int64, teamIDs []string) error {
	// Remove old teams
	_, err := tx.Exec(`DELETE FROM matches_has_teams WHERE match_id = $1`, matchID)
	if err != nil {
		return fmt.Errorf("failed to clear old teams: %w", err)
	}

	// Insert new ones
	for _, teamEncId := range teamIDs {
		teamID, err := crypto.NDecrypt(teamEncId)
		if err != nil {
			return fmt.Errorf("invalid team id: %w", err)
		}

		_, err = tx.Exec(`
            INSERT INTO matches_has_teams (match_id, team_id)
            VALUES ($1, $2)
        `, matchID, teamID)
		if err != nil {
			return fmt.Errorf("failed to insert new team: %w", err)
		}
	}

	return nil
}
