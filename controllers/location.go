package controllers

import (
	"sports-events-api/models"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

func GetAllCountries(c *gin.Context) {
	Countries, err := models.GetAllCountries()
	if err != nil {
		utils.HandleError(c, "Error fetching countries", err)
		return
	}
	utils.HandleSuccess(c, "Countries fetched successfully", map[string]any{"countries": Countries})
}

func GetStateByCountry(c *gin.Context) {
	country_id := DecryptParamId(c, "id", true)
	if country_id == 0 {
		return
	}

	states, err := models.GetStateByCountry(country_id)
	if err != nil {
		utils.HandleError(c, "Error fetching states", err)
		return
	}

	utils.HandleSuccess(c, "States fetched successfully", map[string]any{"states": states})
}

func GetCityByState(c *gin.Context) {
	state_id := DecryptParamId(c, "id", true)
	if state_id == 0 {
		return
	}
	cities, err := models.GetCityByState(state_id)
	if err != nil {
		utils.HandleError(c, "Error fetching cities", err)
		return
	}

	utils.HandleSuccess(c, "Cities fetched successfully", map[string]any{"cities": cities})
}
