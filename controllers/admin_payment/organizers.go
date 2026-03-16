package adminpayment

import (
	"strconv"
	"strings"

	adminPaymentModel "sports-events-api/models/adminPayment"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

func GetAllOrganizers(c *gin.Context) {
	// Extract query parameters for pagination, search, sorting
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "u.name")
	dir := strings.ToUpper(c.DefaultQuery("dir", "ASC")) // ASC or DESC
	offset := (page - 1) * limit

	// Fetch data from model
	totalRecords, organizers, err := adminPaymentModel.GetOrganizers(search, sort, dir, int64(limit), int64(offset))
	if err != nil {
		utils.HandleError(c, "Failed to fetch organizers.", err)
		return
	}

	// Respond with paginated and filtered data
	utils.HandleSuccess(c, "Fetched all organizers successfully.", gin.H{
		"totalRecords": totalRecords,
		"organizers":   organizers,
	})
}
