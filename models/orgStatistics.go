package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"sync"

	"github.com/lib/pq"
)

type OrgFilterConfig struct {
	EventEncId        string   `json:"event_id"`
	EventId           int64    `json:"-"`
	SelectedGameEncId string   `json:"selected_game_id"`
	SelectedGameId    int64    `json:"-"`
	Filter            string   `json:"filter"`
	Dates             []string `json:"dates"`
	EventEncIds       []string `json:"event_ids"`
	EventIds          []int64  `json:"-"`
	// MatchFilter string   `json:"match_filter"`
	// MatchDates  []string `json:"match_dates"`
	// FeesFilter  string   `json:"fees_filter"`
	// FeesDates   []string `json:"fees_dates"`
}

type OrgStats struct {
	EventList         []EncData          `json:"event_list"`
	TotalEvents       int                `json:"total_events"`
	TotalTeams        int                `json:"total_teams"`
	TotalMatches      int                `json:"total_matches"`
	TotalFees         int                `json:"total_fees"`
	TotalPlayers      int                `json:"total_players"`
	TypeGraphs        []TypeWiseGraph    `json:"type_graphs"`
	LocationGraphData []LocationWiseData `json:"location_graph"`
	GraphConfig       []GraphConfig      `json:"graph_config"`
}

type AgeWiseData struct {
	AgeGroupEncId    string `json:"age_group_id"`
	AgeGroupId       int64  `json:"-"`
	AgeGroupName     string `json:"age_group_name"`
	ParticipantCount int    `json:"participant_count"`
	MatchCount       int    `json:"match_count"`
}

type TypeWiseGraph struct {
	EncID     string        `json:"id"`
	ID        int64         `json:"-"`
	GraphData []AgeWiseData `json:"graph_data"`
}

type LocationWiseData struct {
	LocationName  string `json:"location_name"`
	LocationEncId string `json:"location_id"`
	LocationId    int64  `json:"-"`
	IsState       bool   `json:"is_state"`
	Count         int    `json:"count"`
}

type GraphConfig struct {
	GameName    string      `json:"game_name"`
	GameId      int64       `json:"-"`
	GameEncId   string      `json:"game_id"`
	PlayerCount int         `json:"player_count"`
	Types       []GraphType `json:"types"`
}

type GraphType struct {
	EncID       string `json:"id"`
	ID          int64  `json:"-"`
	Name        string `json:"name"`
	PlayerCount int    `json:"player_count"`
	Slug        string `json:"slug"`
}

type ConditionPair struct {
	FilterType string
	Keyword    string
	Condition  string
	Arg        []any
}

func GetOrganizerStatisticsById(orgId int64, config OrgFilterConfig) (OrgStats, error) {
	var stats OrgStats
	var orgIdCondition = ConditionPair{
		FilterType: "RoleBased",
		Keyword:    " AND ",
		Condition:  "e.created_by_id =",
		Arg:        []any{orgId},
	}
	// Launch goroutines to fetch each value
	pairs := []ConditionPair{}
	matchPairs := []ConditionPair{}
	if orgId != 0 {
		pairs = append(pairs, orgIdCondition)
		matchPairs = append(matchPairs, orgIdCondition)
	}

	eventErr := SetEventConditionPairs(config, &pairs)
	matchErr := SetMatchConditionPairs(config, &matchPairs)

	if matchErr != nil || eventErr != nil {
		return stats, fmt.Errorf("invalid filter received")
	}

	var EventArr []ShortEventData
	var ids []int64

	//fetch the eligible events according to the filter
	if err := FetchEventIdsByOrgFilter(config, &ids, &EventArr, pairs...); err != nil {
		return stats, err
	}
	stats.TotalEvents = len(EventArr)
	var EventListConditon []ConditionPair
	if config.EventId == 0 {
		temp := ConditionPair{
			FilterType: "EventRange",
			Condition:  " e.id = ANY ",
			Arg:        []any{pq.Array(ids)}}
		EventListConditon = append(EventListConditon, temp)

		temp.Keyword = " AND "
		matchPairs = append(matchPairs, temp)
		pairs = append(pairs, temp)
	}
	for i := range EventArr {
		stats.EventList = append(stats.EventList, EncData{
			Name:  EventArr[i].Name,
			EncId: crypto.NEncrypt(EventArr[i].Id),
		})
	}
	errChan := make(chan error, 6)

	var wg sync.WaitGroup
	wg.Add(6)
	// Fetch Fees
	go func() {
		defer wg.Done()
		fees, err := FetchFeesCountForOrg(config.EventId, pairs...)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch fees: %w", err)
			return
		}
		stats.TotalFees = fees
	}()

	// Fetch Match Count
	go func() {
		defer wg.Done()
		matches, err := FetchMatchCountForOrg(config.EventId, matchPairs...)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch match count: %w", err)
			return
		}
		stats.TotalMatches = matches
	}()

	// Fetch Team Count
	go func() {
		defer wg.Done()
		teams, err := FetchTeamCountForOrg(config.EventId, EventListConditon...)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch team count: %w", err)
			return
		}
		stats.TotalTeams = teams
	}()

	// Fetch Player Count
	go func() {
		defer wg.Done()
		players, err := FetchPlayerCountForOrg(config.EventId, EventListConditon...)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch player count: %w", err)
			return
		}
		stats.TotalPlayers = players
	}()

	//Fetch State-wise count
	go func() {
		defer wg.Done()
		GraphData, err := FetchStateWiseGraphData(config.EventId, EventListConditon...)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch event count: %w", err)
			return
		}
		stats.LocationGraphData = GraphData
	}()

	// Fetch Known Games
	go func() {
		defer wg.Done()
		games, typeGraphs, err := EnrichKnownGames(config, matchPairs...)
		if err != nil {
			errChan <- err
			return
		}

		stats.GraphConfig = games
		stats.TypeGraphs = typeGraphs
	}()

	// Wait and close channel
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	for err := range errChan {
		if err != nil {
			return stats, err
		}
	}

	return stats, nil
}

func SetEventConditionPairs(config OrgFilterConfig, pairs *[]ConditionPair) error {
	switch config.Filter {
	case "all":
	case "to":
		*pairs = append(*pairs, ConditionPair{
			FilterType: "DateRange",
			Keyword:    " AND ",
			Condition:  "e.to_date <=",
			Arg:        []any{config.Dates[0]},
		})

	case "from":
		*pairs = append(*pairs, ConditionPair{
			FilterType: "DateRange",
			Keyword:    " AND ",
			Condition:  " e.from_date >=",
			Arg:        []any{config.Dates[0]},
		})
	case "fromTo":
		*pairs = append(*pairs, []ConditionPair{
			{
				FilterType: "DateRange",
				Keyword:    " AND ",
				Condition:  "e.from_date <=",
				Arg:        []any{config.Dates[1]},
			}, {
				Keyword:   " AND ",
				Condition: "e.to_date >=",
				Arg:       []any{config.Dates[0]},
			},
		}...)
	default:
		return fmt.Errorf("invalid filter")
	}
	return nil
}
func SetMatchConditionPairs(config OrgFilterConfig, matchPairs *[]ConditionPair) error {
	switch config.Filter {
	case "all":
	case "to":
		*matchPairs = append(*matchPairs, ConditionPair{
			FilterType: "DateRange",
			Keyword:    " AND ",
			Condition:  "m.scheduled_date <=",
			Arg:        []any{config.Dates[0]},
		})

	case "from":
		*matchPairs = append(*matchPairs, ConditionPair{
			FilterType: "DateRange",
			Keyword:    " AND ",
			Condition:  " m.scheduled_date >= ",
			Arg:        []any{config.Dates[0]},
		})
	case "fromTo":
		*matchPairs = append(*matchPairs, []ConditionPair{
			{
				FilterType: "DateRange",
				Keyword:    " AND ",
				Condition:  "m.scheduled_date <=",
				Arg:        []any{config.Dates[1]},
			}, {
				FilterType: "DateRange",
				Keyword:    " AND ",
				Condition:  "m.scheduled_date >=",
				Arg:        []any{config.Dates[0]},
			},
		}...)
	default:
		return fmt.Errorf("invalid filter")
	}
	return nil
}

func buildDynamicConditionString(Condtions []ConditionPair, startIndex int) (string, []interface{}) {
	conditionString := " AND ("
	var args []interface{}
	for i := range Condtions {
		if i > 0 {
			conditionString += "   " + Condtions[i].Keyword + " "
		}
		if Condtions[i].Arg != nil {
			conditionString += fmt.Sprintf(" (%v ", Condtions[i].Condition)
			conditionString += "("
			for j, arg := range Condtions[i].Arg {
				if j > 0 {
					conditionString += ", "
				}
				conditionString += fmt.Sprintf("$%d", startIndex)
				startIndex++
				args = append(args, arg)
			}
			conditionString += " )) "
		} else {
			conditionString += Condtions[i].Condition
		}

	}
	conditionString += ")"
	return conditionString, args
}

func FetchFeesCountForOrg(EventId int64, Conditions ...ConditionPair) (int, error) {
	var count int
	var Condition string
	Args := []interface{}{}
	startIndex := 1

	if EventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{EventId},
		}
		Conditions = append(Conditions, eventCondition)
	}

	var query string

	query = `SELECT  COALESCE(SUM(t.fees), 0) AS total_fees
	    FROM events e
	    JOIN event_transactions t ON e.id = t.event_id
	    WHERE e.status != 'Delete' AND t.payment_status = 'Success'
	`

	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		Args = append(Args, extraArgs...)

	}
	err := database.DB.QueryRow(query, Args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return 0, fmt.Errorf("database error while fetching fees_count--> %v", err)
	}
	return count, nil
}

func FetchMatchCountForOrg(EventId int64, Conditions ...ConditionPair) (int, error) {
	var count int
	var Condition string
	Args := []interface{}{}
	startIndex := 1

	if EventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{EventId},
		}
		Conditions = append(Conditions, eventCondition)
	}

	var query string

	query = `SELECT count(*)
	    FROM matches m
	    JOIN event_has_game_types gt ON gt.id = m.event_has_game_types
	    JOIN event_has_games eg ON eg.id = gt.event_has_game_id
		JOIN events e ON e.id= eg.event_id
	    WHERE e.status != 'Delete'
		`

	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		Args = append(Args, extraArgs...)

	}
	err := database.DB.QueryRow(query, Args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return 0, fmt.Errorf("database error while fetching match_count--> %v", err)
	}
	return count, nil
}

func FetchPlayerCountForOrg(EventId int64, Conditions ...ConditionPair) (int, error) {
	var count int
	var Condition string
	Args := []interface{}{}
	startIndex := 1

	if EventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{EventId},
		}
		Conditions = append(Conditions, eventCondition)
	}

	var query string

	query = `SELECT count(distinct(user_id))
	    FROM events e
		JOIN event_has_users ehu ON ehu.event_id=e.id 
	    WHERE status != 'Delete'
		`

	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		Args = append(Args, extraArgs...)

	}
	err := database.DB.QueryRow(query, Args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return 0, fmt.Errorf("database error while fetching player_count--> %v", err)
	}
	return count, nil
}

func FetchTeamCountForOrg(EventId int64, Conditions ...ConditionPair) (int, error) {
	var count int
	var Condition string
	Args := []interface{}{}
	startIndex := 1

	if EventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{EventId},
		}
		Conditions = append(Conditions, eventCondition)
	}

	var query string
	query = `SELECT count(*)
	    FROM event_has_teams t
		JOIN events e ON t.event_id=e.id
	    WHERE e.status != 'Delete' AND t.status = 'Active'
		`
	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		Args = append(Args, extraArgs...)

	}
	err := database.DB.QueryRow(query, Args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return 0, fmt.Errorf("database error while fetching team_count--> %v", err)
	}
	return count, nil
}

func EnrichKnownGames(filterConfig OrgFilterConfig, Conditions ...ConditionPair) ([]GraphConfig, []TypeWiseGraph, error) {
	games, err := FetchKnownGames(filterConfig.EventId, RemoveDateRangeConditions(Conditions)...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch known games: %w", err)
	}

	var innerWg sync.WaitGroup
	var mu sync.Mutex
	innerErrChan := make(chan error, len(games)*10)
	var typeGraphs []TypeWiseGraph

	for i := range games {
		data := TypeWiseData{
			FilterConfig: filterConfig,
			GameID:       games[i].GameId,
			Conditions:   Conditions,
		}
		SetPlayerCountsAndTypeGraphs(data, i, &games[i], &typeGraphs, &innerWg, &mu, &innerErrChan)
	}

	innerWg.Wait()
	close(innerErrChan)

	for err := range innerErrChan {
		if err != nil {
			return nil, nil, err
		}
	}

	return games, typeGraphs, nil
}

type TypeWiseData struct {
	FilterConfig OrgFilterConfig
	GameID       int64
	TypeID       int64
	Conditions   []ConditionPair
}

func SetPlayerCountsAndTypeGraphs(data TypeWiseData, i int, game *GraphConfig, typeGraphs *[]TypeWiseGraph, innerWg *sync.WaitGroup, mu *sync.Mutex, innerErrChan *chan error) {

	// Game-level player count
	innerWg.Add(1)
	go func(i int, gameID int64) {
		defer innerWg.Done()
		count, err := FetchPlayerCountForGame(gameID, data.FilterConfig.EventId, RemoveDateRangeConditions(data.Conditions)...)
		if err != nil {
			*innerErrChan <- fmt.Errorf("failed to fetch player count for game: %v", err)
			return
		}
		mu.Lock()
		game.PlayerCount = count
		mu.Unlock()
	}(i, data.GameID)

	for j := range game.Types {
		typeID := game.Types[j].ID
		count, err := FetchPlayerCountForGameType(game.GameId, typeID, data.FilterConfig.EventId, RemoveDateRangeConditions(data.Conditions)...)
		if err != nil {
			*innerErrChan <- fmt.Errorf("failed to fetch player count for game type: %w", err)
			return
		}
		mu.Lock()
		game.Types[j].PlayerCount = count
		mu.Unlock()
		if i == 0 {
			innerData := TypeWiseData{
				FilterConfig: data.FilterConfig,
				GameID:       data.GameID,
				Conditions:   data.Conditions,
				TypeID:       typeID,
			}
			// Only for first game: fetch graph data
			SetTypeWiseOrgGraphs(innerData, typeGraphs, mu, innerErrChan)
		}
	}
}

func SetTypeWiseOrgGraphs(data TypeWiseData, typeGraphs *[]TypeWiseGraph, mu *sync.Mutex, innerErrChan *chan error) {
	var typeGraph TypeWiseGraph
	typeGraph.ID = data.TypeID
	var graphData []AgeWiseData
	err := FetchParticipantGraphDataForGame(data.GameID, data.TypeID, data.FilterConfig.EventId, &graphData, RemoveDateRangeConditions(data.Conditions)...)
	if err != nil {
		*innerErrChan <- fmt.Errorf("failed to fetch participant graph: %w", err)
		return
	}
	if err := FetchMatchGraphDataForGame(data.GameID, data.TypeID, &graphData, data.FilterConfig.EventId, data.Conditions...); err != nil {
		*innerErrChan <- fmt.Errorf("failed to fetch match graph: %w", err)
		return
	}

	typeGraph.GraphData = graphData
	mu.Lock()
	if typeGraph.GraphData != nil {
		*typeGraphs = append(*typeGraphs, typeGraph)
	}
	mu.Unlock()
}

func FetchKnownGames(eventId int64, Conditions ...ConditionPair) ([]GraphConfig, error) {
	var KnownGames []GraphConfig
	var query string
	var Condition string
	args := []interface{}{}
	startIndex := 1

	if eventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{eventId},
		}
		Conditions = append(Conditions, eventCondition)
	}
	query = `
		SELECT game_id, game_name
		FROM (
			SELECT DISTINCT ehg.game_id, g.id as gid, g.game_name
			FROM events e
			JOIN event_has_games ehg ON ehg.event_id = e.id 
			JOIN games g ON ehg.game_id = g.id
			WHERE e.status != 'Delete'
	`

	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}
	query += `
			) AS sub
			ORDER BY gid ASC;`
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return KnownGames, fmt.Errorf("database error while fetching--> %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var game GraphConfig
		err := rows.Scan(&game.GameId, &game.GameName)
		if err != nil {
			return KnownGames, fmt.Errorf("error scanning a row for known games--> %v", err)
		}
		game.Types, err = FetchTypeConfig(int64(game.GameId), eventId, Conditions...)
		if err != nil {
			return KnownGames, err
		}
		KnownGames = append(KnownGames, game)
	}
	return KnownGames, nil
}

func FetchTypeConfig(GameId int64, eventId int64, Conditions ...ConditionPair) ([]GraphType, error) {
	var TypeConfig []GraphType
	var query string
	var Condition string
	args := []interface{}{GameId}
	startIndex := 2
	if eventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{eventId},
		}
		Conditions = append(Conditions, eventCondition)
	}
	query = `
		SELECT distinct(ehgt.game_type_id), gt.name
		FROM events e
		JOIN event_has_games ehg ON ehg.event_id= e.id 
		JOIN event_has_game_types ehgt ON ehgt.event_has_game_id= ehg.id 
		JOIN games_types gt ON gt.id= ehgt.game_type_id 
		WHERE e.status != 'Delete' AND ehg.game_id=$1
	`

	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return TypeConfig, fmt.Errorf("database error while fetching TypeConfig--> %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var gameType GraphType
		err := rows.Scan(&gameType.ID, &gameType.Name)
		if err != nil {
			return TypeConfig, fmt.Errorf("error scanning a row for TypeConfig--> %v", err)
		}
		TypeConfig = append(TypeConfig, gameType)
	}
	return TypeConfig, nil
}

func FetchPlayerCountForGame(GameId int64, eventId int64, Conditions ...ConditionPair) (int, error) {
	var count int
	var query string
	var Condition string
	args := []interface{}{GameId}
	startIndex := 2
	if eventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{eventId},
		}
		Conditions = append(Conditions, eventCondition)
	}
	query = `
			SELECT COUNT(DISTINCT(ehu.user_id))
			FROM events e
			JOIN event_has_teams eht on eht.event_id=e.id
			JOIN event_has_users ehu ON ehu.event_has_team_id=eht.id
			WHERE e.status != 'Delete' AND eht.game_id=$1 
		`
	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}
	err := database.DB.QueryRow(query, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return 0, fmt.Errorf("database error in FetchPlayerCountForGame--> %v", err)
	}
	return count, nil
}

func FetchPlayerCountForGameType(GameId int64, TypeId int64, eventId int64, Conditions ...ConditionPair) (int, error) {
	var count int
	var query string
	var Condition string
	args := []interface{}{GameId, TypeId}
	var startIndex = 3
	if eventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{eventId},
		}
		Conditions = append(Conditions, eventCondition)
	}
	query = `
		SELECT COUNT(DISTINCT(ehu.user_id))
		FROM events e
		JOIN event_has_teams eht on eht.event_id=e.id
		JOIN event_has_users ehu ON ehu.event_has_team_id=eht.id
		WHERE e.status != 'Delete'
		AND eht.game_id=$1  
		AND eht.game_type_id=$2
	`
	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}
	err := database.DB.QueryRow(query, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return 0, fmt.Errorf("database error in FetchPlayerCountForGameType--> %v", err)
	}
	return count, nil
}

func FetchMatchGraphDataForGame(GameId int64, TypeId int64, GraphData *[]AgeWiseData, eventId int64, Conditions ...ConditionPair) error {

	var query string
	var Condition string
	args := []interface{}{GameId, TypeId}
	startIndex := 3
	if eventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{eventId},
		}
		Conditions = append(Conditions, eventCondition)
	}
	query = `
		SELECT ag.id, ag.category, count(distinct(m.id)) as match_count
		FROM events e
		JOIN event_has_games eg ON eg.event_id=e.id AND eg.Game_id=$1
		JOIN event_has_game_types ehgt ON ehgt.event_has_game_id=eg.id AND ehgt.game_type_id=$2
		JOIN age_group ag ON ag.id=ehgt.age_group_id
		JOIN matches m ON m.event_has_game_types=ehgt.id
		WHERE e.status != 'Delete'
	`
	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}

	query += `
		GROUP BY ag.id, ag.category
	`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return fmt.Errorf("query error in FetchMatchGraphDataForGame: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dataPoint AgeWiseData
		err := rows.Scan(&dataPoint.AgeGroupId, &dataPoint.AgeGroupName, &dataPoint.MatchCount)
		if err != nil {
			return fmt.Errorf("scan error in FetchMatchGraphDataForGame: %w", err)
		}
		found := false
		for i := range *GraphData {
			if (*GraphData)[i].AgeGroupId == dataPoint.AgeGroupId {
				(*GraphData)[i].MatchCount = dataPoint.MatchCount
				found = true
				break
			}
		}
		if !found {
			*GraphData = append(*GraphData, dataPoint)
		}
	}
	return nil
}

func FetchParticipantGraphDataForGame(GameId int64, TypeId int64, eventId int64, GraphData *[]AgeWiseData, Conditions ...ConditionPair) error {

	var query string
	var Condition string
	args := []interface{}{GameId, TypeId}
	startIndex := 3
	if eventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{eventId},
		}
		Conditions = append(Conditions, eventCondition)
	}
	query = `
		SELECT
			ag.id, 
			ag.category, 
			COUNT(DISTINCT ehu.user_id) AS total_users
		FROM 
			event_has_users ehu
		JOIN 
			event_has_teams eht ON eht.id = ehu.event_has_team_id
		JOIN 
			events e ON e.id = ehu.event_id
		JOIN 
			age_group ag ON ag.id = eht.age_group_id
		WHERE 
			ag.status = 'Active'
			AND eht.game_id = $1
			AND eht.game_type_id = $2
	`
	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}

	query += `
		GROUP BY 
			ag.id,
			ag.category;
	`
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return fmt.Errorf("query error in FetchParticipantGraphDataForGame: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dataPoint AgeWiseData
		err := rows.Scan(&dataPoint.AgeGroupId, &dataPoint.AgeGroupName, &dataPoint.ParticipantCount)
		if err != nil {
			return fmt.Errorf("scan error in FetchParticipantGraphDataForGame: %w", err)
		}
		*GraphData = append(*GraphData, dataPoint)
	}
	return nil
}

func FetchCityWiseGraphData(stateId int64, eventId int64, Conditions ...ConditionPair) ([]LocationWiseData, error) {
	var GraphData []LocationWiseData
	var Condition string
	var query string
	startIndex := 2
	args := []interface{}{stateId}

	query = `	
		SELECT c.id, c.city, count(distinct(ehu.user_id))
		FROM event_has_users ehu
		JOIN events e ON  e.id= ehu.event_id
		JOIN user_details usd ON ehu.user_id=usd.user_id
		JOIN cities c ON c.id= usd.city		
		WHERE 
		e.status='Active'
		AND c.state_id=$1
	`

	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}
	query += `
		Group By c.id,c.city`
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return GraphData, fmt.Errorf("query error in FetchCityWiseGraphData: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dataPoint LocationWiseData
		err := rows.Scan(&dataPoint.LocationId, &dataPoint.LocationName, &dataPoint.Count)
		dataPoint.IsState = false
		if err != nil {
			return GraphData, fmt.Errorf("scan error in FetchCityWiseGraphData: %w", err)
		}
		GraphData = append(GraphData, dataPoint)
	}
	return GraphData, nil
}

func FetchStateWiseGraphData(eventId int64, Conditions ...ConditionPair) ([]LocationWiseData, error) {
	var GraphData []LocationWiseData
	var Condition string
	var query string
	args := []interface{}{}
	startIndex := 1
	if eventId > 0 {
		eventCondition := ConditionPair{
			FilterType: "EventFileter",
			Keyword:    " AND ",
			Condition:  "e.id=",
			Arg:        []any{eventId},
		}
		Conditions = append(Conditions, eventCondition)
	}
	query = `		
		SELECT s.id, s.name, count(distinct(ehu.user_id))
		FROM event_has_users ehu
		JOIN events e ON  e.id= ehu.event_id
		JOIN user_details usd ON ehu.user_id=usd.user_id
		JOIN states s ON s.id= usd.state		
		WHERE 
		e.status='Active'
	`
	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, startIndex)

		query += Condition

		args = append(args, extraArgs...)

	}
	query += `
		GROUP BY s.id, s.name`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return GraphData, fmt.Errorf("query error in FetchStateWiseGraphData: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dataPoint LocationWiseData
		err := rows.Scan(&dataPoint.LocationId, &dataPoint.LocationName, &dataPoint.Count)

		dataPoint.IsState = true
		if err != nil {
			return GraphData, fmt.Errorf("scan error in FetchStateWiseGraphData: %w", err)
		}
		GraphData = append(GraphData, dataPoint)
	}
	return GraphData, nil
}

func FetchEventIdsByOrgFilter(config OrgFilterConfig, ids *[]int64, eventArr *[]ShortEventData, Conditions ...ConditionPair) error {
	var Condition string
	Args := []interface{}{}
	query := `
	SELECT json_agg(result)
	FROM (
	  SELECT
	    id AS id,
	    name AS name
	  FROM events e 
	  WHERE status !='Delete'
	`
	if len(Conditions) != 0 {
		var extraArgs []interface{}
		Condition, extraArgs = buildDynamicConditionString(Conditions, 1)

		query += Condition

		Args = append(Args, extraArgs...)

	}
	query += `
	) AS result;`

	var rawJSON []byte
	err := database.DB.QueryRow(query, Args...).Scan(&rawJSON)
	if err == sql.ErrNoRows || rawJSON == nil {
		*eventArr = []ShortEventData{}
		return nil
	} else if err != nil {
		return fmt.Errorf("database error while fetching event list--> %v", err)
	}

	//unmarshall the struct
	if err := json.Unmarshal(rawJSON, eventArr); err != nil {
		return fmt.Errorf("error unmarshaling event list JSON --> %v", err)
	}

	//populate the array of ids
	for _, event := range *eventArr {
		*ids = append(*ids, event.Id)
	}

	return nil
}

func RemoveDateRangeConditions(allConditions []ConditionPair) []ConditionPair {
	var EventRangeCondition []ConditionPair
	for i := range allConditions {
		if allConditions[i].FilterType != "DateRange" {
			EventRangeCondition = append(EventRangeCondition, allConditions[i])
		}
	}
	return EventRangeCondition
}
