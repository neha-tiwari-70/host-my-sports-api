package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/utils"
	"strconv"
	"strings"
	"time"

	"sports-events-api/database"
	"sports-events-api/models" // Replace with your actual models package import path

	"github.com/gin-gonic/gin"
)

func DeleteUsers(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}
	_, err := DeleteUserById(decryptedId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	err = models.SyncUserDetailStatusByUserId(decryptedId)
	if err != nil {
		utils.HandleError(c, "Database Error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    nil,
		"message": "User deleted successfully",
		"status":  "success",
	})

}

func DeleteUserById(id int64) (*models.User, error) {
	checkQuery := `SELECT id, status FROM users WHERE id = $1`
	var user models.User

	err := database.DB.QueryRow(checkQuery, id).Scan(&user.ID, &user.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching users %v", err)
	}

	if user.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	deleteQuery := `UPDATE users SET status = 'Delete', updated_at = $1 WHERE ID = $2`
	_, err = database.DB.Exec(deleteQuery, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("error deleting user %v", err)
	}

	user.Status = "Delete"
	user.UpdatedAt = time.Now()

	return &user, nil
}

func GetAllUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")
	dir := c.DefaultQuery("dir", "DESC")
	status := c.Query("status")
	offset := (page - 1) * limit

	TotalRecords, users, err := GetUsers(search, sort, dir, status, int64(limit), int64(offset))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"data":    err.Error(),
			"message": "Failed to fetch users.",
		})
		return
	}

	encryptedUsers := make([]map[string]interface{}, 0)
	for _, user := range users {
		encryptedId := crypto.NEncrypt(user.ID)

		details := user.Details
		if details != nil {
			details.EncState = crypto.NEncrypt(int64(details.State))
			details.EncCity = crypto.NEncrypt(int64(details.City))
		}

		encryptedUsers = append(encryptedUsers, map[string]interface{}{
			"id":                encryptedId,
			"user_code":         user.UserCode,
			"name":              user.Name,
			"email":             user.Email,
			"mobile_no":         user.MobileNo,
			"role_slug":         user.RoleSlug,
			"status":            user.Status,
			"otp_status":        user.OTPStatus,
			"email_status":      user.EmailStatus,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
			"organization_name": user.OrganizationName,
			"details":           user.Details,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    gin.H{"totalRecords": TotalRecords, "users": encryptedUsers},
		"message": "Fetched all users successfully.",
	})
}

func GetUsers(search, sort, dir, status string, limit, offset int64) (int, []models.User, error) {
	var users []models.User
	args := []interface{}{limit, offset}

	validSortFields := map[string]string{
		"user_code":  "u.user_code",
		"name":       "u.name",
		"email":      "u.email",
		"mobile_no":  "u.mobile_no",
		"role_slug":  "u.role_slug",
		"status":     "u.status",
		"created_at": "u.created_at",
		"updated_at": "u.updated_at",
	}

	sortField, ok := validSortFields[sort]
	if !ok {
		sortField = "u.created_at"
	}

	query := `
    SELECT
      u.id, u.user_code, u.name, u.email, u.mobile_no, u.role_slug, u.status,
      u.otp_status, u.email_status, u.created_at, u.updated_at,
      ud.organization_name, ud.nationality, ud.skill_level, ud.address,
      ud.facebook_link, ud.insta_link, ud.height, ud.weight, ud.current_team,
      ud.gender, ud.state, ud.city, ud.aadhar_no, ud.dob, ud.profile_image_path,
      ud.status, ud.created_at, ud.updated_at,
      s.name AS state_name, c.city AS city_name,
      COUNT(u.id) OVER() AS totalrecords
    FROM
      users u
    LEFT JOIN user_details ud ON ud.user_id = u.id
    LEFT JOIN states s ON s.id = ud.state
    LEFT JOIN cities c ON c.id = ud.city
    WHERE
      u.status != 'Delete'`

	if status != "" {
		statusValues := strings.Split(status, ",")
		statusPlaceholders := []string{}
		for _, s := range statusValues {
			statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, strings.TrimSpace(s))
		}
		query += fmt.Sprintf(" AND u.status IN (%s)", strings.Join(statusPlaceholders, ", "))
	}
	if search != "" {
		len := len(args)
		search = strings.Trim(search, " ")
		sepStr := strings.ReplaceAll(search, " ", "%") //seperate search
		query += fmt.Sprintf(`
			AND
			( u.name ILIKE $%d
			 OR u.mobile_no ILIKE $%d 
			 OR u.email ILIKE $%d
			 OR u.role_slug ILIKE $%d
			 OR u.status ILIKE $%d
			 OR ud.organization_name ILIKE $%d
			 OR ud.nationality ILIKE $%d
			 OR ud.skill_level ILIKE $%d)
		`, len+1, //name
			len+1, //mobile
			len+2, //email
			len+2, //org name
			len+1, //role
			len+1, //status
			len+1, // nationality
			len+1) //sklill level
		args = append(args, "%"+search+"%")
		args = append(args, "%"+sepStr+"%")
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $1 OFFSET $2", sortField, dir)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	TotalRecords := 0
	for rows.Next() {
		var user models.User
		var details models.UserDetails
		// var stateName, cityName string
		var stateName, cityName sql.NullString

		err := rows.Scan(
			&user.ID, &user.UserCode, &user.Name, &user.Email, &user.MobileNo,
			&user.RoleSlug, &user.Status, &user.OTPStatus, &user.EmailStatus,
			&user.CreatedAt, &user.UpdatedAt,

			&details.OrganizationName, &details.Nationality, &details.Skill_level,
			&details.Address, &details.Facebook_link, &details.Insta_link,
			&details.Height, &details.Weight, &details.Current_team,
			&details.Gender, &details.State, &details.City, &details.Aadhar_no,
			&details.DOB, &details.Profile_Image_Path, &details.Status,
			&details.CreatedAt, &details.UpdatedAt,

			&stateName, &cityName,
			&TotalRecords,
		)
		if err != nil {
			return 0, nil, err
		}

		details.User_id = int(user.ID)
		// details.StateName = stateName
		// details.CityName = cityName
		if stateName.Valid {
			details.StateName = stateName.String
		}
		if cityName.Valid {
			details.CityName = cityName.String
		}

		user.Details = &details
		users = append(users, user)
	}

	return TotalRecords, users, nil
}

func UpdateUserStatus(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	currentStatus, err := GetUserStatusByID(decryptedId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to fetch current status.",
		})
		return
	}

	newStatus := "Inactive"
	if currentStatus == "Inactive" {
		newStatus = "Active"
	}

	err = UpdateUserStatusByID(decryptedId, newStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update status.",
		})
		return
	}

	err = models.SyncUserDetailStatusByUserId(decryptedId)
	if err != nil {
		utils.HandleError(c, "Database Error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User status updated successfully.",
	})
}

func GetUserStatusByID(userID int64) (string, error) {
	var status string
	query := `SELECT status FROM users WHERE id=$1`
	err := database.DB.QueryRow(query, userID).Scan(&status)
	if err != nil {
		log.Printf("Error fetching status for user %d : %v \n", userID, err)
		return "", fmt.Errorf("failed to fetch status")
	}
	return status, nil
}

func UpdateUserStatusByID(userID int64, status string) error {
	query := `UPDATE users SET status = $1 WHERE id = $2`
	_, err := database.DB.Exec(query, status, userID)
	if err != nil {
		log.Printf("Error updating status. err : %v", err)
		return fmt.Errorf("failed to update status")
	}
	return nil
}

func GetProfileRoles(c *gin.Context) {
	profileRoles, err := models.GetAllProfileRoles()
	if err != nil {
		log.Printf("Error Fetching profile roles : %v", err)
		utils.HandleError(c, "Failed to fetch profile roles.")
		return
	}

	for i := range profileRoles {
		profileRoles[i].EncId = crypto.NEncrypt(profileRoles[i].Id)
	}

	utils.HandleSuccess(c, "Profile roles fetched successfully", profileRoles)
}

func ImpersonateUser(c *gin.Context) {
	var request struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, "Invalid request", err)
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		utils.HandleError(c, "Missing or malformed Authorization header")
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := utils.VerifyToken(tokenStr)
	if err != nil {
		utils.HandleError(c, "Invalid token", err)
		return
	}

	adminEmail, ok := claims["email"].(string)
	if !ok || adminEmail == "" {
		utils.HandleError(c, "Invalid token claims (missing email)")
		return
	}

	var adminID int
	err = database.DB.QueryRow("SELECT id FROM admin WHERE email = $1", adminEmail).Scan(&adminID)
	if err != nil {
		if err == sql.ErrNoRows {
			// fmt.Println("adminEmail from token:", adminEmail)
			utils.HandleError(c, "Unauthorized admin", err)
		} else {
			utils.HandleError(c, "DB error", err)
		}
		return
	}

	var userID int
	var userRole string
	err = database.DB.QueryRow("SELECT id, role_slug FROM users WHERE email = $1", request.Email).Scan(&userID, &userRole)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.HandleError(c, "User not found", err)
		} else {
			utils.HandleError(c, "DB error", err)
		}
		return
	}

	if userRole == "admin" {
		utils.HandleInvalidEntries(c, "Cannot impersonate another admin")
		return
	}

	token, err := utils.GenerateToken(request.Email, 24, 0)
	// fmt.Println("Generated Token:", token)
	if err != nil {
		utils.HandleError(c, "Failed to generate impersonation token", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":              "success",
		"message":             "Impersonation successful",
		"impersonation_token": token,
		"user_id":             userID,
	})
	// fmt.Println("Impersonating user ID:", userID)
}

func VerifyImpersonationToken(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	// fmt.Println("auth token:", tokenString)
	if tokenString == "" {
		utils.HandleError(c, "Missing token")
		return
	}

	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	claims, err := utils.ParseToken(tokenString)
	if err != nil || claims.UserEmail == "" {
		utils.HandleError(c, "Invalid token")
		return
	}

	//fetch user by email to that func
	user, err := models.GetUserByEmail(claims.UserEmail)
	if err != nil {
		utils.HandleError(c, "User not found", err)
		return
	}

	if user.Status == "Delete" {
		utils.HandleError(c, "User doesn't exist.")
		return
	}

	if user.Status == "Pending" {
		utils.HandleError(c, "User registration incompleted.")
		return
	}

	//fetch user by user details by that func
	user.Details, err = models.GetUserDetails(user.ID)
	if err != nil {
		utils.HandleError(c, "Error fetching user details", err)
		return
	}

	encryptedID := crypto.NEncrypt(int64(user.ID))
	user.Details.EncState = crypto.NEncrypt(int64(user.Details.State))
	user.Details.EncUser_id = crypto.NEncrypt(int64(user.Details.User_id))
	user.Details.EncCity = crypto.NEncrypt(int64(user.Details.City))

	for i := range user.Details.Games {
		user.Details.Games[i].EncGameId = crypto.NEncrypt(user.Details.Games[i].GameId)
	}

	for i := range user.Details.ProfileRoles {
		user.Details.ProfileRoles[i].EncId = crypto.NEncrypt(user.Details.ProfileRoles[i].Id)
	}

	isModerator := false
	if isMod, err := models.IsUserModerator(int(user.ID)); err == nil && isMod {
		isModerator = true
	}

	newToken, err := utils.GenerateToken(user.Email, 24, int(user.ID))
	if err != nil {
		utils.HandleError(c, "Failed to generate token", err)
		return
	}

	// Final response
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Impersonation verified",
		"token":   newToken,
		"data": gin.H{
			"id":           encryptedID,
			"name":         user.Name,
			"user_code":    user.UserCode,
			"email":        user.Email,
			"mobile_no":    user.MobileNo,
			"otp_status":   user.OTPStatus,
			"email_status": user.EmailStatus,
			"status":       user.Status,
			"created_at":   user.CreatedAt,
			"updated_at":   user.UpdatedAt,
			"details":      user.Details,
			"role_slug":    user.RoleSlug,
			"is_moderator": isModerator,
		},
	})
}

func GetUserById(c *gin.Context) {
	decryptedID := DecryptParamId(c, "id", true)
	if decryptedID == 0 {
		return
	}

	user, err := models.GetFullUserByID(int(decryptedID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	userEncID := crypto.NEncrypt(user.ID)

	if user.Details != nil {
		user.Details.EncState = crypto.NEncrypt(int64(user.Details.State))
		user.Details.EncCity = crypto.NEncrypt(int64(user.Details.City))
	}

	response := map[string]interface{}{
		"id":                userEncID,
		"user_code":         user.UserCode,
		"name":              user.Name,
		"email":             user.Email,
		"mobile_no":         user.MobileNo,
		"role_slug":         user.RoleSlug,
		"status":            user.Status,
		"otp_status":        user.OTPStatus,
		"email_status":      user.EmailStatus,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
		"organization_name": user.Details.OrganizationName,
		"dob":               user.Details.DOB,
		"gender":            user.Details.Gender,
		"details":           user.Details,
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User fetched successfully",
		"data":    response,
	})
}
