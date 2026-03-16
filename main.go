package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	config "sports-events-api/common"
	"sports-events-api/database"
	"sports-events-api/database/migrations"
	"sports-events-api/database/seeders"
	"sports-events-api/models"
	"sports-events-api/routes"

	"github.com/gin-contrib/cors"
	"github.com/robfig/cron/v3"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	database.InitDB()
	config.LoadConfig()
	// Check for command-line arguments for migration and seeder commands
	if len(os.Args) > 2 {
		command := os.Args[1]
		target := os.Args[2]
		switch command {
		case "migrate":
			switch target {
			case "all":
				migrateAll()
			case "admin":
				migrations.AdminMigration("create")
			case "roles":
				migrations.RolesMigration("create")
			case "users":
				migrations.UserMigration("create")
			case "all_for_users":
				migrations.UserMigration("create")
				migrations.UserDetailsMigration("create")
			case "user_details":
				migrations.UserDetailsMigration("create")
			case "user_has_interested_games":
				migrations.UserHasInterestedGames("create")
			case "past_teams":
				migrations.PastTeamsMigration("create")
			case "forgot_users":
				migrations.ForgotUsersMigration("create")
			case "games_types":
				migrations.GamesTypesMigration("create")
			case "games":
				migrations.GamesMigration("create")
			case "game_has_types":
				migrations.GameHasTypesMigration("create")
			case "all_for_game":
				migrations.GamesTypesMigration("create")
				migrations.GamesMigration("create")
				migrations.GameHasTypesMigration("create")
			case "events":
				migrations.EventsMigration("create")
			case "event_has_games":
				migrations.EventHasGamesMigration("create")
			case "event_has_game_types":
				migrations.EventHasGameTypesMigration("create")
			case "event_has_image":
				migrations.EventHasImageMigration("create")
			case "event_has_users":
				migrations.EventHasUsersMigration("create")
			case "event_has_teams":
				migrations.EventHasTeamsMigration("create")
			case "matches":
				migrations.MatchesMigration("create")
			case "matches_has_teams":
				migrations.MatchesHasTeamsMigration("create")
			case "all_for_event":
				migrations.EventsMigration("create")
				migrations.EventHasGamesMigration("create")
				migrations.EventHasGameTypesMigration("create")
				migrations.EventHasImageMigration("create")
				migrations.EventHasIamgeOriginalName("alter")
				migrations.EventHasTeamsMigration("create")
				migrations.EventHasUsersMigration("create")
				migrations.MatchesHasTeamsMigration("create")
				migrations.MatchesMigration("create")
				migrations.SponsorMigration("create")
				// migrations.EventHasUsersMigration("create")
				// migrations.EventHasTeamsMigration("create")
			case "event_has_sponsors":
				migrations.SponsorMigration("create")
			case "otp_verification":
				migrations.OTPMigration("create")
			case "organization_has_score_moderator":
				migrations.OrganizationHasScoreMigration("create")
			case "match_team_has_scores":
				migrations.MatchTeamHasScoresMigration("create")
			case "add_match_name_column":
				migrations.AddMatchNameColumn("alter")
			case "add_is_penalty_column":
				migrations.IsPenaltyMigration()
			case "add_is_last_round_column":
				migrations.AddIsLastRoundColumn("alter")
			case "age_group":
				migrations.AgeGroupMigration("create")
			case "game_has_age_group":
				migrations.GameHasAgeGroupMigration("create")
			case "add_age_group_id_column":
				migrations.AddAgeGroupIdColumn()
				migrations.AddAgeGroupIdColumnTeams()
			case "delete_ageGroup_and_category_column":
				migrations.DeleteCategoryAgeGroupColumn()
			case "add_event_id_column":
				migrations.AddEventIdColumn()
			case "sanitize_user_code_column":
				migrations.SanitizeUserCodeColumn()
			case "assign_gender_to_users":
				migrations.AssignGenderToUsers()
			case "update_gender_constraint":
				migrations.UpdateGenderConstraint()
			case "profile_roles":
				migrations.ProfileRolesMigration("create")
			case "user_has_profile_roles":
				migrations.UserHasProfileRoles("create")
			case "add_contact_person_name_column":
				migrations.AddContactPersonNameColumn()
			case "bank_details":
				migrations.BankDetailsMigration("create")
			case "contact_person_name_column_name_change":
				migrations.ChangeContactPersonNameFieldToOrganizationName()
			case "level_of_competition_alter":
				migrations.LevelOfCompetitionMigration("alter") // add level_of_competition_id column in events table
			case "level_of_competition":
				migrations.CreateLevelOfCompetitionTable("create") // create level_of_competitions table
			case "event_transaction":
				migrations.EventTransactionsMigration("create")
			case "verified_transaction":
				migrations.VerifiedTransactionMigration("create")
			case "add_group_no_column":
				migrations.AddGroupNoColumn()
			case "blogs":
				migrations.BlogsMigration("create")
			case "add_tshirt_size":
				migrations.AddTshirtSize("alter")
			case "add_weight_column":
				migrations.AddEventHasGamesWeight("alter")
			case "add_new_age_group":
				migrations.AddUnder14AgeGroup()
			case "add_max_set_point":
				migrations.AddMaximumSetPoint("alter")
			case "add_tshirt_size_flag":
				migrations.AddIsTshirtSizeRequiredColumn()
			case "new_location_data":
				migrations.MigrateNewLocationData()
			case "drop_current_team_column":
				migrations.DropCurrentTeamColumn()
			case "sync_status_with_user_table":
				migrations.SyncStatusWithUserTable()
			case "add_distance_and_metric_columns":
				migrations.AddDistanceAndMetricColumns()
			case "add_linkedin_link":
				migrations.EventsAlterMigration("alter")
			case "modify_fields_datatype":
				migrations.EventsModifyDatatype("alter")
			case "image_original_name":
				migrations.EventHasIamgeOriginalName("alter")
			case "add_number_of_overs_column":
				migrations.AddNumberOfOversColumn()
			case "add_ball_type_column":
				migrations.AddBallTypeColumn()
			case "add_coach_name_column":
				migrations.AddCoachNameColumn()
			case "match_player_card":
				migrations.MatchPlayerCards("create")
			case "add_cycle_type_column":
				migrations.AddCycleTypeColumn()
			case "add_distance_category_column":
				migrations.AddDistanceCategoryColumn()
			case "contacts":
				migrations.ContactsMigration("create")
			case "fund_accounts":
				migrations.FundAccountsMigration("create")
			case "payouts":
				migrations.PayoutMigration("create")
			case "add_min-max_column":
				migrations.AddMinMaxPlayerColumn()
			default:
				fmt.Printf("No migration found for: %s\n", target)
			}

		case "seed":
			switch target {
			case "all":
				seedAll()
			case "admin":
				seeders.AdminSeeder()
			case "users":
				fmt.Println("Usage Update:",
					"[if you want to use default value for any param use keyword 'def' in it's place",
					"\n eg:\n  'go run main.go seed users 12 18 def' will insert default(100) number of users between the age of 12 and 18",
					"\n  'go run main.go seed users def 18 26' will insert 26 users between the age of default(0) and 18",
					"\n  'go run main.go seed users 12 def 20' will insert 20 users between the age of 12 and default(45)]",
					"\n  'go run main.go seed users' will use all default values")

				var minAge, maxAge, count int
				defaultMinAge := 0
				defaultMaxAge := 45
				defaultCount := 100

				argCount := len(os.Args)

				switch argCount {
				case 3:
					minAge = defaultMinAge
					maxAge = defaultMaxAge
					count = defaultCount

				case 6:
					var err1, err2, err3 error
					minAgeParsed := defaultMinAge
					maxAgeParsed := defaultMaxAge
					countParsed := defaultCount

					if os.Args[3] != "def" {
						minAgeParsed, err1 = strconv.Atoi(os.Args[3])
					}
					if os.Args[4] != "def" {
						maxAgeParsed, err2 = strconv.Atoi(os.Args[4])
					}
					if os.Args[5] != "def" {
						countParsed, err3 = strconv.Atoi(os.Args[5])
					}
					if err1 != nil || err2 != nil || err3 != nil {
						fmt.Println("Invalid input. Use: minAge maxAge count")
						return
					}
					minAge = minAgeParsed
					maxAge = maxAgeParsed
					count = countParsed
				default:
					fmt.Println("Usage:")
					fmt.Println("  go run main.go seed users")
					fmt.Println("  go run main.go seed users 15              # ages 0–15, 100 users")
					fmt.Println("  go run main.go seed users 12 18           # ages 12–18, 100 users")
					fmt.Println("  go run main.go seed users 12 18 300       # ages 12–18, 300 users")
					return
				}

				seeders.SeedFakeUsers(minAge, maxAge, count)

			case "roles":
				seeders.RolesSeeder()
			case "states":
				if err := seeders.StatesSeeder(); err != nil {
					log.Fatal(fmt.Errorf("error seeding states table-->%v", err))
				}
			case "cities":
				if err := seeders.CitiesSeeder(); err != nil {
					log.Fatal(fmt.Errorf("error seeding cities table-->%v", err))
				}
			case "countries":
				seeders.CountriesSeeder()
			case "age_group":
				seeders.AgeGroupSeeder()
			case "new_age_groups":
				seeders.FillMissingAgeGroups()
			case "add_above_23_age_group":
				seeders.AddAbove23AgeGroup()
			case "profile_roles":
				seeders.ProfileRolesSeeder()
			case "add_above_18_age_group":
				seeders.AddAbove18AgeGroup()
			case "user_location":
				var startUser = 0
				var endUser = 0
				var err error
				argCount := len(os.Args)
				if argCount == 5 {
					startUser, err = strconv.Atoi(os.Args[3])
					fmt.Println(startUser)
					if err != nil {
						fmt.Println("invalid input example:'go run main.go seed user_location *startUserId (integer)* *endUserId (integer)*'")
					}
					endUser, err = strconv.Atoi(os.Args[4])
					fmt.Println(endUser)
					if err != nil {
						fmt.Println("invalid input example:'go run main.go seed user_location *startUserId (integer)* *endUserId (integer)*'")
					}
					if endUser < startUser {
						fmt.Println("invalid input example:'go run main.go seed user_location *startUserId (integer)* *endUserId (integer)*'\nwhere start user is lower or equal to the end user")
					}
				}
				seeders.SimulateStateCitySelection(startUser, endUser)
				fmt.Println("done")
			default:
				fmt.Printf("No seeder found for: %s\n", target)
			}

		default:
			fmt.Println("Invalid command. Use 'migrate' or 'seed'.")
		}

		return
	}
	// Start Scheduled Jobs
	c := cron.New()

	// Run every 5 minutes
	_, cronerr := c.AddFunc("*/5 * * * *", models.ExpireOldPendingTransactions)
	if cronerr != nil {
		fmt.Println("Failed to schedule cron job:", cronerr)
	}
	c.Start()

	// Start the Gin server on port 8080
	fmt.Println("Starting server on port ", os.Getenv("PORT"))

	// To initialize Sentry's handler, you need to initialize Sentry itself beforehand
	env := os.Getenv("NODE_ENV") // or use a flag/config
	fmt.Print(env)
	router := gin.Default()
	if env == "production" {
		sentryDsn := os.Getenv("SENTRY_DSN") // put your real DSN here in env var
		err := sentry.Init(sentry.ClientOptions{
			Dsn: sentryDsn,
		})
		if err != nil {
			fmt.Printf("Sentry initialization failed: %v\n", err)
		} else {
			fmt.Println("Sentry initialized.")
			// Add the Sentry middleware
			router.Use(sentrygin.New(sentrygin.Options{}))
		}
	}

	router.Static("/public", "./public")
	// Add CORS middleware with default settings
	// router.Use(cors.Default())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Add allowed origins
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // Preflight request caching
	}))
	// Define application routes
	routes.SetupRoutes(router)

	err := router.Run(os.Getenv("PORT")) // Listen and serve on port 8080
	if err != nil {
		panic(fmt.Sprintf("Failed to start server: %v", err))
	}
}

func migrateAll() {
	migrations.AdminMigration("create")
	migrations.EventHasGameTypesMigration("create")
	migrations.EventHasGamesMigration("create")
	migrations.EventHasImageMigration("create")
	migrations.EventHasTeamsMigration("create")
	migrations.EventHasUsersMigration("create")
	migrations.EventsMigration("create")
	migrations.ForgotUsersMigration("create")
	migrations.GameHasTypesMigration("create")
	migrations.GamesMigration("create")
	migrations.GamesTypesMigration("create")
	migrations.MatchTeamHasScoresMigration("create")
	migrations.MatchesHasTeamsMigration("create")
	migrations.MatchesMigration("create")
	migrations.OTPMigration("create")
	migrations.OrganizationHasScoreMigration("create")
	migrations.RolesMigration("create")
	migrations.SponsorMigration("create")
	migrations.UserDetailsMigration("create")
	migrations.UserHasInterestedGames("create")
	migrations.UserMigration("create")
	migrations.AgeGroupMigration("create")
	migrations.GameHasAgeGroupMigration("create")
	migrations.ProfileRolesMigration("create")
	migrations.UserHasProfileRoles("create")
	migrations.BankDetailsMigration("create")
	migrations.CreateLevelOfCompetitionTable("create")
	migrations.EventTransactionsMigration("create")
	migrations.VerifiedTransactionMigration("create")
	migrations.BlogsMigration("create")
	migrations.MatchPlayerCards("create")
	seeders.CountriesSeeder()
	if err := seeders.StatesSeeder(); err != nil {
		log.Fatal(fmt.Errorf("error seeding states table-->%v", err))
	}
	if err := seeders.CitiesSeeder(); err != nil {
		log.Fatal(fmt.Errorf("error seeding cities table-->%v", err))
	}
}

func seedAll() {
	seeders.CountriesSeeder()
	seeders.CitiesSeeder()
	seeders.StatesSeeder()
	seeders.RolesSeeder()
	seeders.AdminSeeder()
	seeders.AgeGroupSeeder()
	seeders.ProfileRolesSeeder()
}
