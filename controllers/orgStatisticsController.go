package controllers

import (
	"fmt"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"sync"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

func GetOrganizerStatisticsById(c *gin.Context) {
	var config models.OrgFilterConfig
	var OrgId int64

	decryptOrgIdAndEventId(c, &config, &OrgId)

	OrgStats, err := models.GetOrganizerStatisticsById(OrgId, config)
	if err != nil {
		utils.HandleError(c, "Error fetching you statistics", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := range OrgStats.LocationGraphData {
			OrgStats.LocationGraphData[i].LocationEncId = crypto.NEncrypt(int64(OrgStats.LocationGraphData[i].LocationId))
		}
	}()

	go func() {
		defer wg.Done()
		for i := range OrgStats.GraphConfig {
			OrgStats.GraphConfig[i].GameEncId = crypto.NEncrypt(OrgStats.GraphConfig[i].GameId)
			for j := range OrgStats.GraphConfig[i].Types {
				OrgStats.GraphConfig[i].Types[j].EncID = crypto.NEncrypt(OrgStats.GraphConfig[i].Types[j].ID)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := range OrgStats.TypeGraphs {
			OrgStats.TypeGraphs[i].EncID = crypto.NEncrypt(OrgStats.TypeGraphs[i].ID)
			for j := range OrgStats.TypeGraphs[i].GraphData {
				OrgStats.TypeGraphs[i].GraphData[j].AgeGroupEncId = crypto.NEncrypt(OrgStats.TypeGraphs[i].GraphData[j].AgeGroupId)
			}
		}
	}()
	wg.Wait()

	utils.HandleSuccess(c, "Statistics retrieved successfully", OrgStats)
}

func GetOrgStateGraphById(c *gin.Context) {
	var OrgId int64
	var config models.OrgFilterConfig

	decryptOrgIdAndEventId(c, &config, &OrgId)
	// Decrypt the encrypted ID
	StateId := DecryptParamId(c, "stateId", true)
	if StateId == 0 {
		return
	}
	err := error(nil)
	var orgIdCondition = models.ConditionPair{
		FilterType: "RoleBased",
		Keyword:    " AND ",
		Condition:  "e.created_by_id =",
		Arg:        []any{OrgId},
	}
	var EventListConditon []models.ConditionPair
	if OrgId != 0 {
		EventListConditon = []models.ConditionPair{orgIdCondition}
	}

	if config.EventId == 0 && len(config.EventIds) > 0 {
		EventListConditon = append(EventListConditon, models.ConditionPair{
			FilterType: "EventRange",
			Keyword:    " AND ",
			Condition:  " e.id = ANY ",
			Arg:        []any{pq.Array(config.EventIds)}})
	}
	GraphData, err := models.FetchCityWiseGraphData(StateId, config.EventId, EventListConditon...)
	if err != nil {
		utils.HandleError(c, "Could not fetch graph data for this state", err)
		return
	}
	for i := range GraphData {
		GraphData[i].LocationEncId = crypto.NEncrypt(GraphData[i].LocationId)
	}
	utils.HandleSuccess(c, "Graph data retrieved successfully", GraphData)
}

func GetOrgAllStateGraph(c *gin.Context) {
	var OrgId int64
	var config models.OrgFilterConfig

	decryptOrgIdAndEventId(c, &config, &OrgId)
	var OrgIdCondition = models.ConditionPair{
		FilterType: "RoleBased",
		Keyword:    " AND ",
		Condition:  "e.created_by_id =",
		Arg:        []any{OrgId},
	}
	var EventListConditon []models.ConditionPair
	if config.EventId == 0 {
		EventListConditon = []models.ConditionPair{
			{
				FilterType: "EventRange",
				Keyword:    " AND ",
				Condition:  " e.id = ANY ",
				Arg:        []any{pq.Array(config.EventIds)},
			},
		}
	}
	EventListConditon = append(EventListConditon, OrgIdCondition)
	GraphData, err := models.FetchStateWiseGraphData(config.EventId, EventListConditon...)
	if err != nil {
		utils.HandleError(c, "Could not fetch graph data all states", err)
		return
	}
	for i := range GraphData {
		GraphData[i].LocationEncId = crypto.NEncrypt(GraphData[i].LocationId)
	}
	utils.HandleSuccess(c, "Graph data retrieved successfully", GraphData)
}

func GetOrgGameGraph(c *gin.Context) {
	var TypeGraphs []models.TypeWiseGraph
	var OrgId int64
	var config models.OrgFilterConfig

	decryptOrgIdAndEventId(c, &config, &OrgId)
	var orgIdCondition = models.ConditionPair{
		FilterType: "RoleBased",
		Keyword:    " AND ",
		Condition:  "e.created_by_id =",
		Arg:        []any{OrgId},
	}
	matchPairs := []models.ConditionPair{orgIdCondition}
	err := models.SetMatchConditionPairs(config, &matchPairs)
	if err != nil {
		utils.HandleError(c, "Invalid Filter", err)
		return
	}
	if config.EventId == 0 {
		temp := models.ConditionPair{
			FilterType: "EventRange",
			Keyword:    " AND ",
			Condition:  " e.id = ANY ",
			Arg:        []any{pq.Array(config.EventIds)}}
		matchPairs = append(matchPairs, temp)
	}

	Types, err := models.FetchTypeConfig(config.SelectedGameId, config.EventId, models.RemoveDateRangeConditions(matchPairs)...)
	if err != nil {
		utils.HandleError(c, "Error fetching graph types", fmt.Errorf("error in function FetchTypeConfig-->%v", err))
		return
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(Types)) // enough buffer
	for i := range Types {
		data := models.TypeWiseData{
			FilterConfig: config,
			GameID:       config.SelectedGameId,
			Conditions:   matchPairs,
			TypeID:       Types[i].ID,
		}
		models.SetTypeWiseOrgGraphs(data, &TypeGraphs, &mu, &errChan)
	}
	// Wait and close channel
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	for err := range errChan {
		if err != nil {
			utils.HandleError(c, "Error fetching graph data")
			return
		}
	}

	var encWg sync.WaitGroup

	for i := range TypeGraphs {
		encWg.Add(1)
		go func() {
			defer encWg.Done()
			TypeGraphs[i].EncID = crypto.NEncrypt(TypeGraphs[i].ID)
			for j := range TypeGraphs[i].GraphData {
				TypeGraphs[i].GraphData[j].AgeGroupEncId = crypto.NEncrypt(TypeGraphs[i].GraphData[j].AgeGroupId)
			}
		}()
	}
	encWg.Wait()
	utils.HandleSuccess(c, "Graph retreived successfully", TypeGraphs)
}

func decryptOrgIdAndEventId(c *gin.Context, config *models.OrgFilterConfig, OrgId *int64) {
	var err error
	// Decrypt the encrypted ID
	*OrgId = DecryptParamId(c, "orgId", false)

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request", "data": err.Error()})
		sentry.CaptureException(err)
		return
	}

	if config.EventEncId != "" {
		config.EventId, err = crypto.NDecrypt(config.EventEncId)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting EventEncId(value:'%v')->%v", config.EventEncId, err))
			return
		}
		for i := range config.EventEncIds {
			decId, err := crypto.NDecrypt(config.EventEncIds[i])
			if err != nil {
				utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting EventEncId(value:'%v')->%v", config.EventEncIds[i], err))
				return
			}
			config.EventIds = append(config.EventIds, decId)
		}
	}
	if config.SelectedGameEncId != "" {
		config.SelectedGameId, err = crypto.NDecrypt(config.SelectedGameEncId)
		if err != nil {
			utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting SelectedGameEncId(value:'%v')->%v", config.SelectedGameEncId, err))
			return
		}
	}
}
