package models

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"strings"
	"time"

	"github.com/lib/pq"
)

type TeamArray struct {
	Teams []string `json:"teams" validate:"required,min=2,dive,required"`
}

type MatchScheduleInfo struct {
	TeamID             int64  `json:"team_id"`
	TeamName           string `json:"team_name"`
	EventID            int64  `json:"event_id"`
	GameID             int64  `json:"game_id"`
	GameName           string `json:"game_name"`
	GameTypeID         int64  `json:"game_type_id"`
	AgeGroupID         int64  `json:"age_group_id"`
	CreatedBy          int64  `json:"created_by"`
	EventHasGameID     int64  `json:"event_has_game_id"`
	EventHasGameTypeID int64  `json:"event_has_game_type_id"`
	TypeOfTournament   string `json:"type_of_tournament"`
}

type MatchData struct {
	EncEventId    string         `json:"event_id"`
	EncGameId     string         `json:"game_id"`
	EncGameTypeId string         `json:"game_type_id"`
	EncAgeGroupId string         `json:"category_id"`
	EventId       int64          `json:"-"`
	GameId        int64          `json:"-"`
	GameTypeId    int64          `json:"-"`
	AgeGroupId    int64          `json:"-"`
	TeamIds       []int          `json:"-"`
	MatchId       int64          `json:"-"`
	MatchEncId    string         `json:"match_id"`
	MatchName     string         `json:"match_name"`
	ScheduledDate string         `json:"scheduled_date"`
	Venue         string         `json:"venue"`
	VenueLink     string         `json:"venue_link"`
	StartTime     string         `json:"start_time"`
	EndTime       string         `json:"end_time"`
	Teams         []TeamInfo     `json:"teams"`
	History       []MatchHistory `json:"history"`
}

type Match struct {
	EventId            int64       `json:"-"`
	EventEncId         string      `json:"event_id"`
	GameId             int64       `json:"-"`
	GameEncId          string      `json:"game_id"`
	GameTypeId         int64       `json:"-"`
	GameTypeEncId      string      `json:"game_type_id"`
	EventHasGameTypeId int64       `json:"event_has_game_type_id"`
	MatchId            int64       `json:"-"`
	MatchName          *string     `json:"match_name"`
	StadiumName        *string     `json:"stadium_name"`
	VenueLink          *string     `json:"venue_link"`
	VenueName          *string     `json:"venue"`
	MatchEncId         string      `json:"match_id"`
	Team1ID            int64       `json:"-"`
	Team1EncId         string      `json:"team1_id,omitempty"`
	Team1Name          string      `json:"team1_name,omitempty"`
	Team1Logo          string      `json:"team1_logo,omitempty"`
	Team1GroupNo       *int        `json:"team1_group_no,omitempty"`
	Team2ID            int64       `json:"-"`
	Team2EncId         string      `json:"team2_id,omitempty"`
	Team2Name          string      `json:"team2_name,omitempty"`
	Team2Logo          string      `json:"team2_logo,omitempty"`
	Team2GroupNo       *int        `json:"team2_group_no,omitempty"`
	TeamsArray         []Team      `json:"teams_array"`
	WinTeamId          interface{} `json:"winTeamId"` // Encrypted ID or null
	WinTeamEncId       string      `json:"winTeamEncId,omitempty"`
	IsDraw             *bool       `json:"isDraw"`
	ScheduledDate      *string     `json:"scheduled_date"`
	StartTime          *string     `json:"start_time"`
	TournamentType     string      `json:"tournament_type"`
	RoundNo            int64       `json:"round_no"`
}

// Team represents a team participating in the event
type Team struct {
	TeamEncId     string `json:"id"`
	ID            int64  `json:"-"`
	Name          string `json:"name"`
	TeamCaptain   string `json:"team_captain"`
	TeamCaptainID int64  `json:"team_captain_id"`
	TeamLogoPath  string `json:"team_logo_path"`
	GroupNo       *int   `json:"group_no"`
	Slug          string `json:"slug"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	TotalPoints   int64  `json:"total_points"` // Points calculated from matches
}

// MatchHasTeam represents the relationship between matches and teams
type MatchHasTeam struct {
	ID      int64
	MatchID int64
	TeamID  int64
	Points  int64
}

type TeamInfo struct {
	TeamId         int64         `json:"-"`
	EncTeamId      string        `json:"team_id"`
	ID             int64         `json:"-"`
	TeamName       string        `json:"team_name"`
	TeamLogo       string        `json:"team_logo"`
	GroupNo        *int          `json:"group_no"`
	Position       int64         `json:"position"`
	Played         int64         `json:"played"`
	GoalDifference int64         `json:"goal_difference"`
	Points         sql.NullInt64 `json:"points"`
	Goals          int           `json:"goals"`
	GamesPlayed    int           `json:"gamesPlayed"`
}

/*
	type MatchHistory struct {
		Date        string        `json:"date"`
		Team1Points sql.NullInt64 `json:"team_1_points"`
		Team2Points sql.NullInt64 `json:"team_2_points"`
	}
*/
type MatchHistory struct {
	MatchName   string    `json:"match_name"`
	Date        time.Time `json:"date"`
	Team1Points int       `json:"team1_points"`
	Team2Points int       `json:"team2_points"`
	Team1Logo   string    `json:"team1_logo"`
	Team2Logo   string    `json:"team2_logo"`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
func CalculatePointsForLeague(matches []Match) ([]Team, error) {
	// Step 1: Create a map to store total points for each team
	teamPoints := make(map[int64]int64)

	// Step 2: Loop through the matches and calculate total points for each team
	for _, match := range matches {
		// Fetch teams for the current match (raw SQL query)
		rows, err := database.DB.Query(`
			SELECT team_id, points
			FROM matches_has_teams
			WHERE match_id = $1`, match.MatchId)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		// Calculate total points for each team in the match
		for rows.Next() {
			var teamID int64
			var points int64
			if err := rows.Scan(&teamID, &points); err != nil {
				return nil, err
			}
			teamPoints[teamID] += points
		}
	}

	// Step 3: Fetch team details from event_has_teams and add the points
	var teamsWithPoints []Team
	for teamID, points := range teamPoints {
		// Fetch team details (raw SQL query)
		var team Team
		var logo sql.NullString
		err := database.DB.QueryRow(`
			SELECT id, team_name, team_captain, team_logo_path, slug, status, created_at, updated_at
			FROM event_has_teams
			WHERE id = $1`, teamID).Scan(&team.ID, &team.Name, &team.TeamCaptainID, &logo, &team.Slug, &team.Status, &team.CreatedAt, &team.UpdatedAt)
		if err != nil {
			return nil, err
		}
		team.TotalPoints = points

		encryptedTeamID := crypto.NEncrypt(teamID)
		if err != nil {
			return nil, err
		}
		team.TeamEncId = encryptedTeamID
		if logo.Valid && logo.String != "" {
			team.TeamLogoPath = logo.String
		} else {
			team.TeamLogoPath = "public/uploads/static/staticLogo.png"
		}
		teamsWithPoints = append(teamsWithPoints, team)
	}

	// Step 4: Sort teams by total points in descending order
	sort.Slice(teamsWithPoints, func(i, j int) bool {
		return teamsWithPoints[i].TotalPoints > teamsWithPoints[j].TotalPoints
	})

	// Step 5: Handle tie-breakers, returning teams with the highest points
	var topTeams []Team

	highestPoints := teamsWithPoints[0].TotalPoints
	for _, team := range teamsWithPoints {
		if team.TotalPoints == highestPoints {
			teamLogoPath := team.TeamLogoPath
			defaultLogoPath := "public/static/staticTeamLogo.png"
			if team.TeamLogoPath == "" || !fileExists(teamLogoPath) {
				team.TeamLogoPath = defaultLogoPath
			}
			topTeams = append(topTeams, team)
		} else {
			break
		}
	}

	// Step 6: Return the top teams
	return topTeams, nil
}

func CalculatePointsForAtheletics(matches []Match) ([]Team, error) {
	// Step 1: Create a map to store total points for each team
	teamPoints := make(map[int64]int64)

	// Step 2: Loop through the matches and calculate total points for each team
	for _, match := range matches {
		// Fetch teams for the current match (raw SQL query)
		rows, err := database.DB.Query(`
			SELECT team_id, points
			FROM matches_has_teams
			WHERE match_id = $1`, match.MatchId)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		// Calculate total points for each team in the match
		for rows.Next() {
			var teamID int64
			var points int64
			if err := rows.Scan(&teamID, &points); err != nil {
				return nil, err
			}
			teamPoints[teamID] += points
		}
	}

	// Step 3: Fetch team details from event_has_teams and add the points
	var teamsWithPoints []Team
	for teamID, points := range teamPoints {
		// Fetch team details (raw SQL query)
		var team Team
		var logo sql.NullString
		err := database.DB.QueryRow(`
			SELECT id, team_name, team_captain, team_logo_path, slug, status, created_at, updated_at
			FROM event_has_teams
			WHERE id = $1`, teamID).Scan(&team.ID, &team.Name, &team.TeamCaptainID, &logo, &team.Slug, &team.Status, &team.CreatedAt, &team.UpdatedAt)
		if err != nil {
			return nil, err
		}
		team.TotalPoints = points

		encryptedTeamID := crypto.NEncrypt(teamID)
		if err != nil {
			return nil, err
		}
		team.TeamEncId = encryptedTeamID
		if logo.Valid && logo.String != "" {
			team.TeamLogoPath = logo.String
		} else {
			team.TeamLogoPath = "public/uploads/static/staticLogo.png"
		}
		teamsWithPoints = append(teamsWithPoints, team)
	}

	// Step 4: Sort teams by total points in descending order
	sort.Slice(teamsWithPoints, func(i, j int) bool {
		return teamsWithPoints[i].TotalPoints > teamsWithPoints[j].TotalPoints
	})

	// Step 5: Handle tie-breakers, returning teams with the highest points
	var topTeams []Team

	highestPoints := teamsWithPoints[0].TotalPoints
	for _, team := range teamsWithPoints {
		if team.TotalPoints == highestPoints {
			teamLogoPath := team.TeamLogoPath
			defaultLogoPath := "public/static/staticTeamLogo.png"
			if team.TeamLogoPath == "" || !fileExists(teamLogoPath) {
				team.TeamLogoPath = defaultLogoPath
			}
			topTeams = append(topTeams, team)
		} else {
			break
		}
	}

	// Step 6: Return the top teams
	return topTeams, nil
}

func CalculateLeaguePointsForGroupedTeams(matches []Match) ([]Team, error) {
	teamPointsByGroup := make(map[int64]map[int64]int64) // group_no -> team_id -> points

	// Step 1: Loop through matches and calculate points
	for _, match := range matches {
		rows, err := database.DB.Query(`
			SELECT eht.group_no, mht.team_id, mht.points
			FROM matches_has_teams mht
			JOIN event_has_teams eht ON mht.team_id = eht.id
			WHERE mht.match_id = $1`, match.MatchId)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var groupNo, teamID, points int64
			if err := rows.Scan(&groupNo, &teamID, &points); err != nil {
				return nil, err
			}
			if _, ok := teamPointsByGroup[groupNo]; !ok {
				teamPointsByGroup[groupNo] = make(map[int64]int64)
			}
			teamPointsByGroup[groupNo][teamID] += points
		}
	}

	var topTeams []Team // final result - top teams from all groups

	// Step 2: For each group, determine top team(s)
	for groupNo, teamPoints := range teamPointsByGroup {
		var teams []Team
		for teamID, points := range teamPoints {
			var team Team
			var logo sql.NullString
			err := database.DB.QueryRow(`
				SELECT id, team_name, team_captain, team_logo_path, group_no, slug, status, created_at, updated_at
				FROM event_has_teams
				WHERE id = $1`, teamID).Scan(&team.ID, &team.Name, &team.TeamCaptainID, &logo, &team.GroupNo, &team.Slug, &team.Status, &team.CreatedAt, &team.UpdatedAt)
			if err != nil {
				return nil, err
			}
			team.TotalPoints = points

			encryptedTeamID := crypto.NEncrypt(teamID)
			if err != nil {
				return nil, err
			}
			team.TeamEncId = encryptedTeamID
			if logo.Valid && logo.String != "" {
				team.TeamLogoPath = logo.String
			} else {
				team.TeamLogoPath = "public/uploads/static/staticLogo.png"
			}
			teams = append(teams, team)
		}

		// Sort teams by points
		sort.Slice(teams, func(i, j int) bool {
			return teams[i].TotalPoints > teams[j].TotalPoints
		})

		// Get top scoring team(s) from this group
		highestPoints := teams[0].TotalPoints
		for _, team := range teams {
			if team.TotalPoints == highestPoints {
				if team.TeamLogoPath == "" || !fileExists(team.TeamLogoPath) {
					team.TeamLogoPath = "public/static/staticTeamLogo.png"
				}
				topTeams = append(topTeams, team)
			} else {
				break
			}
		}
		fmt.Println(groupNo)
	}

	return topTeams, nil
}

func VerifyTeamDetails(teamIDs []int64) (*MatchScheduleInfo, error) {
	args := []interface{}{}
	placeholders := []string{}
	for i, id := range teamIDs {
		args = append(args, id)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	query := fmt.Sprintf(`
		SELECT
			eht.event_id,
			eht.game_id,
			g.game_name,
			eht.game_type_id,
			eht.age_group_id,
			eht.created_by,
			eg.id AS event_has_game_id,
			egt.id AS event_has_game_type_id,
			eg.type_of_tournament
		FROM
			event_has_teams eht
		JOIN event_has_games eg ON eht.event_id = eg.event_id AND eht.game_id = eg.game_id
		JOIN event_has_game_types egt ON eg.id = egt.event_has_game_id AND eht.game_type_id = egt.game_type_id  AND egt.age_group_id = eht.age_group_id
		JOIN games g ON eg.game_id = g.id
		WHERE eht.id IN (%s)
		GROUP BY eht.event_id, eht.game_id, g.game_name, eht.game_type_id, eht.age_group_id, eht.created_by, eg.id, egt.id, eg.type_of_tournament
	`, strings.Join(placeholders, ", "))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []MatchScheduleInfo{}
	for rows.Next() {
		var info MatchScheduleInfo
		if err := rows.Scan(
			&info.EventID,
			&info.GameID,
			&info.GameName,
			&info.GameTypeID,
			&info.AgeGroupID,
			&info.CreatedBy,
			&info.EventHasGameID,
			&info.EventHasGameTypeID,
			&info.TypeOfTournament,
		); err != nil {
			return nil, err
		}
		results = append(results, info)
	}
	// fmt.Println("results := ", results)
	// fmt.Println(results[0])
	// if len(results) == 0 {
	// 	return nil, errors.New("No matching data found")
	// }0
	// if len(results) > 1 {
	// 	return nil, errors.New("Some team do not belong to the same tournament!")
	// }
	return &results[0], nil
}

func GenerateLeagueMatches(teams []int64) []Match {
	n := len(teams)
	if n%2 != 0 {
		teams = append(teams, 0) // Add a dummy team for bye
		n++
	}

	rounds := n - 1
	half := n / 2
	var matches []Match

	for round := 0; round < rounds; round++ {
		for i := 0; i < half; i++ {
			t1 := teams[i]
			t2 := teams[n-1-i]
			if t1 == 0 || t2 == 0 {
				continue // Skip matches with dummy team
			}
			matches = append(matches, Match{
				Team1ID: t1,
				Team2ID: t2,
			})
		}
		// Rotate teams
		teams = append([]int64{teams[0]}, append([]int64{teams[n-1]}, teams[1:n-1]...)...)
	}
	// fmt.Println("Matches", matches)
	return matches
}

func GetMaxSwissRounds(teamCount int) int64 {
	return int64(math.Ceil(math.Log2(float64(teamCount))))
}

func StoreMatches(matches []Match, eventHasGameTypeID int64, scheduledBy int64, roundNo int64, tournamentType string) error {
	for _, match := range matches {
		// Insert into matches table
		var matchID int

		// Insert both teams into matches_has_teams
		// Special handling for athletics (multiple team1s, no team2s)
		if match.Team1ID != 0 && match.Team2ID == 0 && tournamentType == "Atheletics" {
			// Insert one match ID for all athletics entries
			err := database.DB.QueryRow(`
				INSERT INTO matches (event_has_game_types, schedule_by, round_no)
				VALUES ($1, $2, $3)
				RETURNING id
			`, eventHasGameTypeID, scheduledBy, roundNo).Scan(&matchID)
			if err != nil {
				return fmt.Errorf("failed to insert athletics match: %v", err)
			}

			// Insert all athletics participants individually
			_, err = database.DB.Exec(`
				INSERT INTO matches_has_teams (match_id, team_id)
				VALUES ($1, $2)
			`, matchID, match.Team1ID)
			if err != nil {
				return fmt.Errorf("failed to insert athletics participant: %v", err)
			}
		} else if match.Team2ID == 0 {
			err := database.DB.QueryRow(`
			INSERT INTO matches (event_has_game_types, schedule_by, round_no, isdraw)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, eventHasGameTypeID, scheduledBy, roundNo, false).Scan(&matchID)
			if err != nil {
				return fmt.Errorf("failed to insert match: %v", err)
			}
			// Bye case — insert only one team
			_, err = database.DB.Exec(`
				INSERT INTO matches_has_teams (match_id, team_id, points)
				VALUES ($1, $2, $3)
			`, matchID, match.Team1ID, 2)
			if err != nil {
				return fmt.Errorf("failed to insert bye team: %v", err)
			}
		} else {
			err := database.DB.QueryRow(`
			INSERT INTO matches (event_has_game_types, schedule_by, round_no)
			VALUES ($1, $2, $3)
			RETURNING id
		`, eventHasGameTypeID, scheduledBy, roundNo).Scan(&matchID)
			if err != nil {
				return fmt.Errorf("failed to insert match: %v", err)
			}
			// Normal case — insert both teams
			_, err = database.DB.Exec(`
				INSERT INTO matches_has_teams (match_id, team_id)
				VALUES ($1, $2), ($1, $3)
			`, matchID, match.Team1ID, match.Team2ID)
			if err != nil {
				return fmt.Errorf("failed to insert match teams: %v", err)
			}
		}
	}
	return nil
}

func IsMatchAlreadyScheduled(eventHasGameTypeId int64, roundNo ...int64) (bool, error) {
	var number int
	if roundNo != nil {
		err := database.DB.QueryRow(
			`SELECT COUNT(*) FROM matches
		WHERE event_has_game_types = $1 AND round_no = $2
		`, eventHasGameTypeId, roundNo[0]).Scan(&number)

		if err != nil || number > 0 {
			return true, err
		}
		return false, nil
	} else {
		err := database.DB.QueryRow(
			`SELECT COUNT(*) FROM matches
		WHERE event_has_game_types = $1
		`, eventHasGameTypeId).Scan(&number)

		if err != nil || number > 0 {
			return true, err
		}
		return false, nil
	}
}

func GetByeTeams(eventHasGameTypeId int64, previousRound int64) (int64, error) {
	query := `
	SELECT
    mht.team_id
FROM matches m
JOIN matches_has_teams mht ON m.id = mht.match_id
WHERE m.event_has_game_types = $1
  AND m.round_no = $2
  AND m.id IN (
      SELECT match_id
      FROM matches_has_teams mht2
      JOIN matches m2 ON m2.id = mht2.match_id
      WHERE m2.event_has_game_types = $1
        AND m2.round_no = $2
      GROUP BY match_id
      HAVING COUNT(*) = 1
  )

	`
	var byeTeam sql.NullInt64
	err := database.DB.QueryRow(query, eventHasGameTypeId, previousRound).Scan(&byeTeam)
	if err != nil {
		return 0, err
	}
	// defer rows.Close()

	// var teams []int64
	// for rows.Next() {
	// 	var teamID int64
	// 	if err := rows.Scan(&teamID); err != nil {
	// 		return nil, err
	// 	}
	// 	teams = append(teams, teamID)
	// }
	if byeTeam.Valid {
		return byeTeam.Int64, nil
	} else {
		return 0, nil
	}
}

func HandleMatches(eventHasGameTypeId, scheduleBy int64, roundNo int64, tournamentType string, teams []int64, game_name string) ([]Match, error) {
	switch tournamentType {
	case "League":
		if game_name == "Chess" || game_name == "Carrom" {
			matches := GenerateSwissMatch(eventHasGameTypeId, teams, roundNo)
			if len(matches) == 0 {
				return nil, fmt.Errorf("error generating swiss matches")
			}
			if err := StoreMatches(matches, eventHasGameTypeId, scheduleBy, roundNo, tournamentType); err != nil {
				return nil, fmt.Errorf("error storing matches: %v", err)
			}
			return matches, nil
		} else {
			matches := GenerateLeagueMatches(teams)
			if len(matches) == 0 {
				return nil, fmt.Errorf("error generating league matches")
			}
			if err := StoreMatches(matches, eventHasGameTypeId, scheduleBy, roundNo, tournamentType); err != nil {
				return nil, fmt.Errorf("error storing matches: %v", err)
			}
			return matches, nil
		}

	case "Knockout":
		previousByeTeams, _ := GetByeTeams(eventHasGameTypeId, roundNo-1)
		matches := GenerateKnockoutMatches(teams, previousByeTeams)
		if len(matches) == 0 {
			return nil, fmt.Errorf("error generating knockout matches")
		}
		if err := StoreMatches(matches, eventHasGameTypeId, scheduleBy, roundNo, tournamentType); err != nil {
			return nil, fmt.Errorf("error storing matches: %v", err)
		}
		return matches, nil

	case "League cum knockout":
		var allMatches []Match
		// if roundNo == 1 && len(teams) <= 3 {
		// 	allMatches = GenerateLeagueMatches(teams)

		// } else
		if roundNo == 1 && len(teams) > 3 {
			// STEP 1: Fetch group_no per team
			groupedTeams := make(map[int][]int64)
			for _, teamID := range teams {
				groupNo, err := GetGroupNumber(teamID) // implement this
				if err != nil {
					return nil, fmt.Errorf("error fetching group number for team %d: %v", teamID, err)
				}
				groupedTeams[groupNo] = append(groupedTeams[groupNo], teamID)
			}

			// STEP 2: Generate matches for each group
			for groupNo, group := range groupedTeams {
				groupMatches := GenerateLeagueMatches(group)
				for i := range groupMatches {
					groupMatches[i].Team1GroupNo = &groupNo
					groupMatches[i].Team2GroupNo = &groupNo
				}
				allMatches = append(allMatches, groupMatches...)
			}

		} else if roundNo > 1 {
			previousByeTeams, _ := GetByeTeams(eventHasGameTypeId, roundNo-1)
			allMatches = GenerateKnockoutMatches(teams, previousByeTeams)
		}

		if len(allMatches) == 0 {
			return nil, fmt.Errorf("error generating matches for league-cum-knockout")
		}
		if err := StoreMatches(allMatches, eventHasGameTypeId, scheduleBy, roundNo, tournamentType); err != nil {
			return nil, fmt.Errorf("error storing matches: %v", err)
		}
		return allMatches, nil
	case "Atheletics", "Time Trial", "Mass Start", "Relay", "Fun Ride", "Endurance":
		m := Match{
			Team1ID: 0, // placeholder, teams handled separately
			Team2ID: 0,
		}
		matches := []Match{m}
		if err := StoreMatchesForAthletics(matches, eventHasGameTypeId, scheduleBy, roundNo, teams); err != nil {
			return nil, fmt.Errorf("error storing matches: %v", err)
		}
		return matches, nil

	default:
		return nil, fmt.Errorf("invalid tournament type: %s", tournamentType)
	}
}

func StoreMatchesForAthletics(matches []Match, eventHasGameTypeID int64, scheduledBy int64, roundNo int64, teamIDs []int64) error {
	if len(matches) != 1 {
		return fmt.Errorf("only one match should be created for athletics")
	}

	// Step 1: Insert a single match
	var matchID int
	err := database.DB.QueryRow(`
		INSERT INTO matches (event_has_game_types, schedule_by, round_no)
		VALUES ($1, $2, $3)
		RETURNING id
	`, eventHasGameTypeID, scheduledBy, roundNo).Scan(&matchID)
	if err != nil {
		return fmt.Errorf("failed to insert athletics match: %v", err)
	}

	// Step 2: Insert all team IDs for that one match
	query := `INSERT INTO matches_has_teams (match_id, team_id) VALUES `
	vals := []interface{}{}
	for i, teamID := range teamIDs {
		query += fmt.Sprintf("($%d, $%d),", i*2+1, i*2+2)
		vals = append(vals, matchID, teamID)
	}
	query = query[:len(query)-1] // remove trailing comma

	_, err = database.DB.Exec(query, vals...)
	if err != nil {
		return fmt.Errorf("failed to insert athletics match teams: %v", err)
	}

	return nil
}

func GetGroupNumber(teamID int64) (int, error) {
	var groupNo int
	err := database.DB.QueryRow(`SELECT group_no FROM event_has_teams WHERE id = $1`, teamID).Scan(&groupNo)
	if err != nil {
		return 0, err
	}
	return groupNo, nil
}

func GenerateSwissMatch(eventHasGameTypeId int64, teams []int64, roundNo int64) []Match {
	if roundNo == 1 {
		// Round 1: Top vs Bottom
		sort.Slice(teams, func(i, j int) bool { return teams[i] < teams[j] })
		mid := len(teams) / 2
		var matches []Match
		for i := 0; i < mid; i++ {
			matches = append(matches, Match{
				Team1ID: teams[i],
				Team2ID: teams[mid+i],
			})
		}
		// If odd, give bye to last team
		if len(teams)%2 == 1 {
			matches = append(matches, Match{
				Team1ID: teams[len(teams)-1],
				Team2ID: 0, // Indicates BYE
			})
		}
		return matches
	}

	// Round > 1: group by points
	teamPoints := GetTeamPointsFromPreviousRounds(eventHasGameTypeId, teams)
	// fmt.Println("Team points came", teamPoints)
	// Group teams by points
	pointsMap := make(map[float64][]int64)
	for teamID, pts := range teamPoints {
		pointsMap[pts] = append(pointsMap[pts], teamID)
	}

	// Sort point buckets descending
	var pointsList []float64
	for pts := range pointsMap {
		pointsList = append(pointsList, pts)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(pointsList)))

	var finalMatches []Match
	var leftover *int64 = nil // holds unpaired team from previous group

	for _, pts := range pointsList {
		group := pointsMap[pts]
		sort.Slice(group, func(i, j int) bool { return group[i] < group[j] }) // Optional

		// If there is a leftover from previous group, pair it with the first team in this group
		if leftover != nil && len(group) > 0 {
			finalMatches = append(finalMatches, Match{
				Team1ID: *leftover,
				Team2ID: group[0],
			})
			group = group[1:] // Remove the used team
			leftover = nil    // Clear leftover
		}

		// Now pair remaining teams in this group
		for i := 0; i+1 < len(group); i += 2 {
			finalMatches = append(finalMatches, Match{
				Team1ID: group[i],
				Team2ID: group[i+1],
			})
		}

		// If odd one remains, carry it to next group
		if len(group)%2 == 1 {
			last := group[len(group)-1]
			leftover = &last
		}
	}

	// If still one team is left over after all groups, give a bye
	if leftover != nil {
		finalMatches = append(finalMatches, Match{
			Team1ID: *leftover,
			Team2ID: 0, // Bye
		})
	}
	// fmt.Println("final matches", finalMatches)
	return finalMatches
}

func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	return int(math.Pow(2, math.Ceil(math.Log2(float64(n)))))
}

func GenerateKnockoutMatches(teams []int64, previousByeTeam int64) []Match {
	rand.Seed(time.Now().UnixNano())

	n := len(teams)
	totalSlots := nextPowerOfTwo(n)
	totalByes := totalSlots - n

	// fmt.Println("Total Teams:", n)
	// fmt.Println("Next Power of 2:", totalSlots)
	// fmt.Println("Total Byes:", totalByes)
start:
	// Shuffle teams for fairness
	rand.Shuffle(len(teams), func(i, j int) { teams[i], teams[j] = teams[j], teams[i] })

	// Step 3: Divide into upper and lower halves
	tuh, tlh := 0, 0

	if n%2 == 0 {
		tuh = n / 2
		tlh = n / 2
	} else {
		tuh = (n + 1) / 2
		tlh = (n - 1) / 2
	}

	upper := append([]int64{}, teams[:tuh]...)
	lower := append([]int64{}, teams[tuh:]...)

	// Step 4–5: Allocate byes
	positions := make([]int64, 0, totalSlots)

	bIndex := 0
	fmt.Println(tlh, bIndex)
	byePattern := []string{"lastL", "firstU", "lastU", "firstL"}

	for i := 0; i < totalByes; i++ {
		pattern := byePattern[i%4]
		var team int64
		switch pattern {
		case "lastL":
			if len(lower) > 0 {
				team = lower[len(lower)-1]
				lower = lower[:len(lower)-1]
			}
		case "firstU":
			if len(upper) > 0 {
				team = upper[0]
				upper = upper[1:]
			}
		case "lastU":
			if len(upper) > 0 {
				team = upper[len(upper)-1]
				upper = upper[:len(upper)-1]
			}
		case "firstL":
			if len(lower) > 0 {
				team = lower[0]
				lower = lower[1:]
			}
		}
		positions = append(positions, team)
	}

	// Append the rest of the teams
	positions = append(positions, upper...)
	positions = append(positions, lower...)

	// Add extra empty slots for full pairing
	for len(positions) < totalSlots {
		positions = append(positions, 0)
	}

	// Create matches
	var matches []Match
	for i := 0; i < totalSlots; i += 2 {
		t1 := positions[i]
		t2 := positions[i+1]

		if t1 == 0 || t2 == 0 {
			// Bye
			if t1 != 0 {
				if t1 == previousByeTeam {
					goto start
				}
				matches = append(matches, Match{Team1ID: t1})
			} else if t2 != 0 {
				if t2 == previousByeTeam {
					goto start
				}
				matches = append(matches, Match{Team1ID: t1})
			}
		} else {
			matches = append(matches, Match{Team1ID: t1, Team2ID: t2})
		}
	}
	// fmt.Println(matches)
	return matches
}

func FetchLatestMatchesWithTeams(eventHasGameTypeID int64, roundNo int64) ([]Match, error) {
	var matches []Match

	// 1. Fetch matches
	matchQuery := `
		SELECT id, match_name,scheduled_date, start_time, venue, venue_link, isDraw
		FROM matches
		WHERE event_has_game_types = $1 AND round_no = $2
		GROUP BY id, isDraw, match_name
		ORDER BY id
	`
	rows, err := database.DB.Query(matchQuery, eventHasGameTypeID, roundNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Match
		var matchID int64

		err := rows.Scan(&matchID, &m.MatchName, &m.ScheduledDate, &m.StartTime, &m.VenueName, &m.VenueLink, &m.IsDraw)
		if err != nil {
			return nil, err
		}

		m.MatchId = matchID
		m.MatchEncId = crypto.NEncrypt(matchID)
		// m.MatchEncId, _ = crypto.Encrypt(matchID) // Encrypt if required

		// 2. Fetch teams from matches_has_teams
		teamIDs := []int64{}
		teamQuery := `SELECT team_id FROM matches_has_teams WHERE match_id = $1`
		teamRows, err := database.DB.Query(teamQuery, matchID)
		if err != nil {
			return nil, err
		}

		for teamRows.Next() {
			var tid int64
			if err := teamRows.Scan(&tid); err != nil {
				return nil, err
			}
			teamIDs = append(teamIDs, tid)
		}
		teamRows.Close()

		// 3. Get team details from event_has_teams
		for _, tid := range teamIDs {
			var team Team
			var logo sql.NullString
			err := database.DB.QueryRow(`
				SELECT id, team_name, team_logo_path, group_no, team_captain, slug, status, created_at, updated_at
				FROM event_has_teams
				WHERE id = $1
			`, tid).Scan(
				&team.ID, &team.Name, &logo, &team.GroupNo, &team.TeamCaptainID, &team.Slug,
				&team.Status, &team.CreatedAt, &team.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}

			if logo.Valid && logo.String != "" {
				team.TeamLogoPath = logo.String
			} else {
				team.TeamLogoPath = "public/uploads/static/staticLogo.png"
			}

			team.TeamEncId = crypto.NEncrypt(team.ID) // Encrypt if needed
			m.TeamsArray = append(m.TeamsArray, team)
		}

		// fetching points
		pointsMap, err := GetMatchPoints(matchID)
		if err != nil {
			return nil, err
		}

		var winTeamID *int64
		if m.IsDraw != nil && *m.IsDraw {
			winTeamID = nil
		} else if len(m.TeamsArray) == 2 {
			team1 := m.TeamsArray[0]
			team2 := m.TeamsArray[1]

			team1Points, team1Exists := pointsMap[team1.ID]
			team2Points, team2Exists := pointsMap[team2.ID]

			if (team1Exists || team2Exists) && team1Points >= 0 && team2Points >= 0 {
				if team1Points > team2Points {
					winTeamID = &team1.ID
				} else if team2Points > team1Points {
					winTeamID = &team2.ID
				}
			}
		}

		if winTeamID != nil {
			m.WinTeamId = crypto.NEncrypt(*winTeamID)
		} else {
			m.WinTeamId = nil
		}

		matches = append(matches, m)
	}

	return matches, nil
}

func GetTeamsByIDs(teamIDs []int64) ([]TeamInfo, error) {
	query := `SELECT id, team_name, team_logo_path, group_no FROM event_has_teams WHERE id = ANY($1)`
	rows, err := database.DB.Query(query, pq.Array(teamIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []TeamInfo
	for rows.Next() {
		var t TeamInfo
		var logo sql.NullString
		err := rows.Scan(&t.ID, &t.TeamName, &logo, &t.GroupNo)
		if err != nil {
			return nil, err
		}
		if logo.Valid && logo.String != "" {
			t.TeamLogo = logo.String
		} else {
			t.TeamLogo = "public/uploads/static/staticLogo.png"
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func GetTeamPointsFromPreviousRounds(eventHasGameTypeId int64, teams []int64) map[int64]float64 {
	// Get latest round
	latestRound, err := GetLatestRound(eventHasGameTypeId)
	if err != nil {
		log.Println("Error fetching latest round:", err)
		return nil
	}
	// fmt.Println("Round : ", latestRound)
	// if latestRound <= 1 {
	// 	return make(map[int64]float64) // No previous rounds
	// }

	query := `
	SELECT mht.team_id, COALESCE(SUM(mht.points), 0)
	FROM matches_has_teams mht
	INNER JOIN matches m ON m.id = mht.match_id
	WHERE m.event_has_game_types = $1 AND m.round_no <= $2 AND mht.team_id = ANY($3)
	GROUP BY mht.team_id
	`

	rows, err := database.DB.Query(query, eventHasGameTypeId, latestRound, pq.Array(teams))
	if err != nil {
		fmt.Errorf("Error querying team points:", err)
		return nil
	}
	defer rows.Close()

	pointsMap := make(map[int64]float64)
	for rows.Next() {
		var teamID int64
		var points float64
		if err := rows.Scan(&teamID, &points); err == nil {
			pointsMap[teamID] = points
		}
	}

	// Ensure all teams are present with at least 0 points
	for _, team := range teams {
		if _, exists := pointsMap[team]; !exists {
			pointsMap[team] = 0
		}
	}
	// fmt.Println("points map of previous rounds : ", pointsMap)
	return pointsMap
}

func GetMatchPoints(matchId int64) (map[int64]int, error) {
	rows, err := database.DB.Query(`
		SELECT team_id, points
		FROM matches_has_teams
		WHERE match_id = $1`, matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make(map[int64]int)
	for rows.Next() {
		var teamID int64
		var point sql.NullInt64

		if err := rows.Scan(&teamID, &point); err != nil {
			return nil, err
		}

		if point.Valid {
			points[teamID] = int(point.Int64)
		} else {
			points[teamID] = -1 // or 0 or any sentinel value to indicate missing
		}
	}
	return points, nil
}

func EditMatchData(matchData *MatchData) (*MatchData, error) {
	updateQuery := `Update matches SET match_name = $1, scheduled_date = $2, venue = $3, venue_link = $4, start_time = $5, end_time = $6
	WHERE id = $7
	RETURNING id, match_name, scheduled_date, venue, venue_link, start_time, end_time`

	var updatedData MatchData
	err := database.DB.QueryRow(updateQuery, matchData.MatchName, matchData.ScheduledDate, matchData.Venue, matchData.VenueLink, matchData.StartTime, matchData.EndTime, matchData.MatchId).Scan(
		&updatedData.MatchId,
		&updatedData.MatchName,
		&updatedData.ScheduledDate,
		&updatedData.Venue,
		&updatedData.VenueLink,
		&updatedData.StartTime,
		&updatedData.EndTime)
	if err != nil {
		return nil, fmt.Errorf("Error Updating Match Data", err)
	}
	return &updatedData, nil
}

/*
	func GetMatchById(id int64) (*MatchData, error) {
		query := `SELECT id, match_name, scheduled_date, venue, venue_link, start_time, end_time FROM matches WHERE id = $1`
		var match MatchData

		err := database.DB.QueryRow(query, id).Scan(
			&match.MatchId,
			&match.MatchName,
			&match.ScheduledDate,
			&match.Venue,
			&match.VenueLink,
			&match.StartTime,
			&match.EndTime,
		)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("match is not found")
		} else if err != nil {
			return nil, fmt.Errorf("No data found")
		}
		return &match, nil
	}
*/
func GetMatchById(id int64) (*MatchData, error) {
	query := `
	SELECT
	m.match_name,
	m.scheduled_date,
	m.venue,
	m.venue_link,
	m.start_time,
	m.end_time,
	ehgt.game_type_id,
	ehgt.age_group_id,
	ehg.game_id,                -- actual game_id
	ehg.event_id                -- <-- fetch event_id here
FROM matches m
JOIN event_has_game_types ehgt ON m.event_has_game_types = ehgt.id
JOIN event_has_games ehg ON ehgt.event_has_game_id = ehg.id
WHERE m.id = $1;

	`

	var match MatchData
	match.MatchId = id

	// Fetch match details
	err := database.DB.QueryRow(query, id).Scan(
		&match.MatchName,
		&match.ScheduledDate,
		&match.Venue,
		&match.VenueLink,
		&match.StartTime,
		&match.EndTime,
		&match.GameTypeId,
		&match.AgeGroupId,
		&match.GameId,
		&match.EventId, // <- correctly positioned event_id
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("match is not found")
	} else if err != nil {
		return nil, fmt.Errorf("error retrieving match data: %v", err)
	}

	match.EncGameId = crypto.NEncrypt(match.GameId)
	match.EncGameTypeId = crypto.NEncrypt(match.GameTypeId)
	match.EncAgeGroupId = crypto.NEncrypt(match.AgeGroupId)
	match.EncEventId = crypto.NEncrypt(match.EventId)

	// Fetch teams participating in this match
	teamQuery := `
		SELECT eht.id, eht.team_name, eht.team_logo_path, mht.points
		FROM matches_has_teams mht
		JOIN event_has_teams eht ON mht.team_id = eht.id
		WHERE mht.match_id = $1;
	`

	rows, err := database.DB.Query(teamQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error fetching match teams: %v", err)
	}
	defer rows.Close()

	var teams []TeamInfo
	var team1ID, team2ID int64
	for rows.Next() {
		var team TeamInfo
		var rawTeamId int64
		var logo sql.NullString

		if err := rows.Scan(&rawTeamId, &team.TeamName, &logo, &team.Points); err != nil {
			return nil, fmt.Errorf("error scanning team info: %v", err)
		}

		team.TeamId = rawTeamId
		team.EncTeamId = crypto.NEncrypt(rawTeamId)

		if team1ID == 0 {
			team1ID = rawTeamId
		} else {
			team2ID = rawTeamId
		}

		if logo.Valid && logo.String != "" {
			team.TeamLogo = logo.String
		} else {
			team.TeamLogo = "public/uploads/static/staticLogo.png"
		}

		// Fetch goals for the team from match_team_has_scores
		goalQuery := `
			SELECT COUNT(*) FROM match_team_has_scores
			WHERE match_id = $1 AND team_id = $2;
		`
		err := database.DB.QueryRow(goalQuery, id, rawTeamId).Scan(&team.Goals)
		if err != nil {
			return nil, fmt.Errorf("error fetching goals for team: %v", err)
		}

		// Games played is 1 since they are part of this match
		team.GamesPlayed = 1

		teams = append(teams, team)
	}

	// Fetch previous 5 meetings between the teams
	// historyQuery := `
	// // 	SELECT
	// //     m.scheduled_date,
	// // 	m.match_name,
	// //     MAX(CASE WHEN eht.id = $1 THEN mht.points END) AS team1_points,
	// //     MAX(CASE WHEN eht.id = $2 THEN mht.points END) AS team2_points,
	// //     MAX(CASE WHEN eht.id = $1 THEN eht.team_logo_path END) AS team1_logo,
	// //     MAX(CASE WHEN eht.id = $2 THEN eht.team_logo_path END) AS team2_logo
	// // FROM matches m
	// // JOIN matches_has_teams mht ON m.id = mht.match_id
	// // JOIN event_has_teams eht ON eht.id = mht.team_id
	// // WHERE mht.team_id IN ($1, $2)
	// // AND m.id IN (
	// //     SELECT match_id
	// //     FROM matches_has_teams
	// //     WHERE team_id IN ($1, $2)
	// //     GROUP BY match_id
	// //     HAVING COUNT(DISTINCT team_id) = 2
	// // )
	// // GROUP BY m.id, m.scheduled_date
	// // ORDER BY m.scheduled_date ASC;`
	historyQuery := `
	SELECT
			m.scheduled_date,
			m.match_name,
			COALESCE(MAX(CASE WHEN eht.id = $1 THEN mht.points END), 0) AS team1_points,
			COALESCE(MAX(CASE WHEN eht.id = $2 THEN mht.points END), 0) AS team2_points,
			COALESCE(MAX(CASE WHEN eht.id = $1 THEN eht.team_logo_path END), '') AS team1_logo,
			COALESCE(MAX(CASE WHEN eht.id = $2 THEN eht.team_logo_path END), '') AS team2_logo
	FROM matches m
	JOIN matches_has_teams mht ON m.id = mht.match_id
	JOIN event_has_teams eht ON eht.id = mht.team_id
	WHERE mht.team_id IN ($1, $2)
		AND m.id IN (
			SELECT match_id
			FROM matches_has_teams
			WHERE team_id IN ($1, $2)
			GROUP BY match_id
			HAVING COUNT(DISTINCT team_id) = 2
	)
	GROUP BY m.id, m.scheduled_date
	ORDER BY m.scheduled_date ASC;
	`

	historyRows, err := database.DB.Query(historyQuery, team1ID, team2ID)
	if err != nil {
		return nil, fmt.Errorf("error fetching match history: %v", err)
	}
	defer historyRows.Close()
	var history []MatchHistory
	for historyRows.Next() {
		var historyItem MatchHistory
		if err := historyRows.Scan(
			&historyItem.Date,
			&historyItem.MatchName,
			&historyItem.Team1Points,
			&historyItem.Team2Points,
			&historyItem.Team1Logo,
			&historyItem.Team2Logo,
		); err != nil {
			return nil, fmt.Errorf("error scanning match history: %v", err)
		}
		history = append(history, historyItem)
	}

	match.Teams = teams
	match.History = history

	return &match, nil
}

func GetLatestRound(eventHasGameTypeId int64) (int64, error) {
	query := `SELECT COALESCE(MAX(round_no), 0) FROM matches WHERE event_has_game_types = $1`

	var latestRound int64
	err := database.DB.QueryRow(query, eventHasGameTypeId).Scan(&latestRound)
	if err != nil {
		return 0, err
	}

	return latestRound, nil
}

func CheckForNullPoints(matches []Match) (bool, error) {
	// Iterate through the matches and check for NULL points in the matches_has_teams table
	for _, match := range matches {
		rows, err := database.DB.Query(`
			SELECT points
			FROM matches_has_teams
			WHERE match_id = $1 AND points IS NULL`, match.MatchId)
		if err != nil {
			return false, err
		}
		defer rows.Close()

		// If any NULL points are found, return true
		if rows.Next() {
			return true, nil
		}
	}

	// No NULL points found
	return false, nil
}
