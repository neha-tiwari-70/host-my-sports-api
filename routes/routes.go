package routes

import (
	"net/http"
	"sports-events-api/controllers"
	adminpayment "sports-events-api/controllers/admin_payment"
	"sports-events-api/middleware" // Import the middleware package

	"github.com/gin-gonic/gin"
)

// "fmt"

//	"os"
//
// "github.com/getsentry/sentry-go"
// sentrygin "github.com/getsentry/sentry-go/gin"
func SetupRoutes(router *gin.Engine) {

	// Health check endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Server is running",
		})
	})

	// Add routes for user-related operations
	publicRoutes := router.Group("/api/v1")
	{
		// Authentication & Verification
		publicRoutes.POST("/signup", controllers.Register)
		publicRoutes.POST("/login", controllers.Login)
		publicRoutes.POST("/verify-otp", controllers.VerifyOTP)
		publicRoutes.POST("/resendOtp", controllers.ResendOTP)
		publicRoutes.POST("/forgot-password", controllers.ForgotPassword)
		publicRoutes.POST("/reset-password", controllers.ResetPassword)
		publicRoutes.POST("/set-password", controllers.SetPassword)
		publicRoutes.POST("/verify-email", controllers.VerifyEmail)
		publicRoutes.POST("add-email", controllers.SendVerificationLink)
		publicRoutes.GET("/organization-search", controllers.OrganizationExistCheck)
		publicRoutes.GET("users/:id", controllers.GetUserById)

		// Event viewing (public access)
		publicRoutes.GET("/events", controllers.GetAllEvents)
		publicRoutes.POST("/events/:id", controllers.GetEventsById)

		//blogs
		publicRoutes.GET("/blogs", controllers.GetAllBlogs)
		publicRoutes.GET("/blogs/:id", controllers.GetBlogById)

		// Game types (public viewing)
		publicRoutes.GET("/games-types", controllers.GetAllGamesTypes)      // list games-types
		publicRoutes.GET("/games-types/:id", controllers.GetGamesTypesById) // view particular games-types
		publicRoutes.GET("/gamestypes", controllers.GetAllTypes)            // alias: to show games types

		// Games (public viewing)
		publicRoutes.GET("/ageGroup", controllers.GetAllAgeGroup)
		publicRoutes.POST("/gamesInfo", controllers.GetGamesInfoByGameIds)
		publicRoutes.GET("/games", controllers.GetAllGames)           // list games
		publicRoutes.GET("/games/:id", controllers.GetGamesById)      // view game
		publicRoutes.GET("/games/:id/delete", controllers.DeleteGame) // delete game (GET used here)

		// Match Listing
		publicRoutes.GET("/match-listing/:event_id/:game_id/:game_type_id/:category_ids/:participant_id", controllers.GetMatchesByEvent) // match-listing
		publicRoutes.GET("/match-results/:event_id/:game_id/:game_type_id/:category_id/:participant_id", controllers.GetMatchResult)     // match-result
		publicRoutes.GET("/match-points/:event_id/:game_id/:game_type_id/:category_ids/:participant_id", controllers.GetPointTable)      // match-points
		publicRoutes.GET("/match-squades/:event_id/:game_id/:game_type_id/:category_ids/:participant_id", controllers.GetSquadStats)     // match-squades
		publicRoutes.GET("/match-listing", controllers.GetAllMatches)                                                                    //Get All Matches Listing
		publicRoutes.GET("/api/matches/total", controllers.GetTotalMatchCount)

		// Miscellaneous lookups
		publicRoutes.GET("/countries", controllers.GetAllCountries)
		publicRoutes.GET("/state/:id", controllers.GetStateByCountry)
		publicRoutes.GET("/city/:id", controllers.GetCityByState)

		// Game Config
		publicRoutes.GET("/game-list", controllers.GetGamesList)
		publicRoutes.POST("/game-list/config", controllers.GetGameConfig)

		publicRoutes.GET("/player/count", controllers.GetTotalEventUsers)
		publicRoutes.GET("/user/count", controllers.GetTotalUsers)
		publicRoutes.POST("/webhook-payment", controllers.RazorpayWebhookHandler)
		// publicRoutes.GET("/organizer-details/:user_id", controllers.CheckOrganizerBankDetailsEmptyOrNot)
		publicRoutes.POST("/webhook-payout", adminpayment.RazorpayWebhook)

		publicRoutes.POST("verify-impersonation-token", controllers.VerifyImpersonationToken) //to verify the impersonate user token at frontside
		//NOTE Apply AuthenticationMiddleware to all routes that require login
		publicRoutes.Use(middleware.AuthenticationMiddleware())

		// User Profile and Account Management
		publicRoutes.POST("/changepassword", controllers.ChangePassword)
		publicRoutes.POST("/update-profile-image", controllers.UpdateProfileImage)
		publicRoutes.POST("/remove-profile-image", controllers.EmptyProfileImage)
		publicRoutes.POST("/user-profile-update", controllers.ProfileUpdateController)
		publicRoutes.GET("/coaches", controllers.GetAllCoaches)
		publicRoutes.POST("/user-bank-details", controllers.AddBankInfo)
		publicRoutes.GET("/user-bank-details/:user_id", controllers.GetBankInfo)
		publicRoutes.GET("/users", controllers.GetAllUsers)
		publicRoutes.POST("/users/:id/delete", controllers.DeleteUsers)
		publicRoutes.POST("/users/:id/status", controllers.UpdateUserStatus)
		publicRoutes.POST("/impersonate-user", controllers.ImpersonateUser) // to generaet impersonate user token at adminside
		publicRoutes.GET("/users/profileRoles", controllers.GetProfileRoles)
		publicRoutes.POST("/contact-us", controllers.ContactUsFunc)

		publicRoutes.POST("/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "",
				"data":    "",
				"message": "Create a new user (not implemented yet)",
			})
		})

		// Event Management (Protected)
		publicRoutes.GET("/your-history", controllers.GetAllEvents) // Possibly user’s event history
		publicRoutes.POST("/create-event", controllers.CreateEvent)
		publicRoutes.POST("/events/:id/delete", controllers.DeleteEvent)
		publicRoutes.POST("/events/:id/status", controllers.UpdateEventStatus)
		publicRoutes.POST("/events/:id/update", controllers.UpdateEvent)
		publicRoutes.POST("events/:id/logo", controllers.DeleteEventLogo)
		publicRoutes.POST("/events/:id/upload-multiple-files", controllers.UploadMultipleEventFiles)
		publicRoutes.POST("/events/:id/update-images", controllers.UpdateEventImages)
		publicRoutes.POST("/events/:id/delete-image", controllers.DeleteEventImage)
		publicRoutes.POST("/events/:id/update-sponsors", controllers.UpdateMultipleSponsors)
		publicRoutes.POST("/upload-multiple-sponsors/:id", controllers.UploadMultipleSponsorLogos)
		publicRoutes.POST("/sponsors/:id/sponsor-logo", controllers.DeleteSponsorLogo)

		// Game Types Management (Protected)
		publicRoutes.POST("/games-types", controllers.CreateGamesTypes)
		publicRoutes.POST("/games-types/:id", controllers.UpdateGamesTypesById)
		publicRoutes.POST("/games-types/:id/delete", controllers.DeleteGamesTypes)
		publicRoutes.POST("/games-types/:id/status", controllers.UpdateGameTypeStatus)

		// Game Management (Protected)
		publicRoutes.POST("/games", controllers.CreateGame)     // insert game
		publicRoutes.POST("/games/:id", controllers.UpdateGame) // update game
		publicRoutes.POST("/games/:id/status", controllers.UpdateGameStatus)

		// Participation & Registration
		publicRoutes.POST("/event-participation/:eventId/:gameId/:ehgtypeId/:userCode", controllers.GetParticipantByUserCode)
		publicRoutes.POST("/teams/verify-name", controllers.VerifyTeamName) //verify team name
		publicRoutes.GET("/get-participated-user/:teamId/:userId", controllers.GetParticipatedUser)
		publicRoutes.GET("/events-registration/:id/:userId", controllers.GetEventsById)
		publicRoutes.POST("/event-participation", controllers.SaveGames)                    // Save in-progress registration
		publicRoutes.POST("/event-participation/update", controllers.SaveGames)             // Update registration
		publicRoutes.POST("/event-participation/submit", controllers.FinalizeParticipation) // Final submission

		// Moderator Management
		publicRoutes.POST("/moderator/:action/:organizationId/:moderatorId/:eventId", controllers.UpdateModerator)
		publicRoutes.POST("/moderators/:organizationId/:moderatorId/:eventId", controllers.AddModerator)
		publicRoutes.GET("/moderators/:organizationId", controllers.GetAllModeratorsForOrg)
		publicRoutes.GET("/get-all-moderators-for-event/:eventId", controllers.GetAllModeratorsForEvent)
		publicRoutes.GET("/events/:id/organization", controllers.GetEventByOrganizationId)
		publicRoutes.GET("/moderator/:organizationId/:eventId/:userCode", controllers.GetModeratorByUserCode)

		// Scoring & Teams
		publicRoutes.POST("/members", controllers.GetTeamPlayers)
		publicRoutes.POST("/match-scores", controllers.GetMatchTeamScores)
		publicRoutes.POST("/match-scores/save", controllers.ProcessMatchTeamScores)
		publicRoutes.POST("/allocate-winner", controllers.AllocateWinner)

		publicRoutes.GET("/your-history/event/:id/user/:userId", controllers.GetParticipatedEventById)

		//Level of competition
		publicRoutes.POST("level-of-competition", controllers.CreateLevelOfCompetition)  // create level of competition
		publicRoutes.GET("/level-of-competition", controllers.GetAllLevelsOfCompetition) // view all level of competition
		// publicRoutes.GET("/level-of-competition/config", controllers.GetLevelConfig)
		publicRoutes.GET("/level-of-competition/all", controllers.GetAllLevelConfigs)                     //fetch all level of competition
		publicRoutes.GET("/level-of-competition/:id/delete", controllers.Deletelevelofcompetition)        // delete particular level of competition
		publicRoutes.POST("/level-of-competition/:id", controllers.UpdateLevelofCompetitionById)          // update level of competition with the particular id
		publicRoutes.POST("/level-of-competition/:id/status", controllers.UpdateLevelofCompetitionStatus) // update status of loc
		publicRoutes.GET("/level-of-competition/:id", controllers.GetLevelofCompetitionById)              // view particualr loc with id

		//Player Dashboard
		publicRoutes.GET("total-tournaments/:userId", controllers.GetTotalTournaments)       // fetch total tournaments
		publicRoutes.GET("/game-participated/:userId", controllers.GetTotalGameParticipated) // for games participated count
		publicRoutes.GET("user-cities/:userId", controllers.GetCityofUser)                   // for user cities (locationwise filter)
		publicRoutes.POST("user-games", controllers.GetUserGames)                            // participated games for user
		publicRoutes.GET("user-levels/:userId", controllers.GetUserCompetitionLevels)        // level of competition (filtering)
		publicRoutes.POST("graph-game", controllers.GetDashboardGraphStats)                  // for the graph & games
		publicRoutes.GET("match-states/:userId", controllers.GetMatchStats)
		publicRoutes.GET("get-cities/:userId", controllers.GetUserCityNamesFromEvents) // for get cities

		//payment Module
		publicRoutes.GET("/createOrder/:userId/:eventId", controllers.InitializeEventPayment)
		publicRoutes.POST("/handleRazorPayResponse", controllers.HandleRazorPayResponse)
		publicRoutes.GET("/get-payment/:event_id/:user_id", controllers.GetPaymentStatus) // Set Last Round

		//organization statistics
		publicRoutes.POST("/get-organization-statistics/:orgId", controllers.GetOrganizerStatisticsById)
		publicRoutes.POST("/get-org-state-graph/:orgId/:stateId", controllers.GetOrgStateGraphById)
		publicRoutes.POST("/get-org-state-graph/:orgId", controllers.GetOrgAllStateGraph)
		publicRoutes.POST("/get-org-game-graph/:orgId", controllers.GetOrgGameGraph)

		// Team and Match info
		publicRoutes.GET("/teams", controllers.GetAllTeams)
		publicRoutes.POST("/match", controllers.ScheduleMatches)                                                      // Schedule match
		publicRoutes.POST("/setLastRound/:event_id/:game_id/:game_type_id/:age_group_id", controllers.SetIsLastRound) // Set Last Round
		publicRoutes.GET("/match/:id", controllers.GetMatchDataById)                                                  // Get match info
		publicRoutes.POST("/match/:id", controllers.AddMatchInfo)                                                     // Add match info
		publicRoutes.POST("/groups", controllers.MakeTeamGroups)
		publicRoutes.POST("/groups/save", controllers.SaveGroupsHandler)
		publicRoutes.POST("/matches/update-teams", controllers.HandleUpdateMatchTeams)

		//Blogs
		publicRoutes.POST("/blogs", controllers.CreateBlogs) // Add blogs
		publicRoutes.POST("/blogs/:id", controllers.UpdateBlogById)
		publicRoutes.POST("/blogs/:id/delete", controllers.DeleteBlog)
		publicRoutes.POST("/blogs/:id/status", controllers.UpdateBlogStatus)

		//Participations data
		publicRoutes.GET("event/:event_id/:game_id/:game_type_id/:age_group_id/participants", controllers.GetParticipantsData)

		// Replace player from team
		publicRoutes.POST("/replace-players", controllers.ReplacePlayerFromTeam)
		publicRoutes.POST("/replace-catain/:team_id/:new_id", controllers.ChangeCaptain)

		// NOTE
		// publicRoutes.GET("/schedule-match", controllers.ScheduleMatches)
		// publicRoutes.POST("", controllers.InitiatePhonePePayment)
		// publicRoutes.GET("/nationalities", controllers.GetAllNationalities)

		// NOTE
		publicRoutes.POST("/payout", adminpayment.PayoutHandling)
		publicRoutes.GET("/organizers", adminpayment.GetAllOrganizers)
		publicRoutes.GET("/organizer-send-mail/:user_id", controllers.SendMailToOrganizer)

	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173") // Allow specific origin
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
