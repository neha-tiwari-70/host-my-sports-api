package seeders

import (
	"fmt"
	"log"
	"math/rand"
	"sports-events-api/database"
	"sports-events-api/models"
	"time"

	"github.com/bxcodec/faker/v3"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func AdminSeeder() {
	adminPassword, err := hashPassword("Admin@123")
	if err != nil {
		log.Fatalf("Failed to hash password for Admin: %v", err)
	}

	// user1Password, err := hashPassword("password123")
	if err != nil {
		log.Fatalf("Failed to hash password for User1: %v", err)
	}

	query := `
	INSERT INTO admin (name, email, password, mobile_no)
	VALUES
	('Admin', 'admin@gmail.com', $1, null)`

	_, err = database.DB.Exec(query, adminPassword)
	if err != nil {
		panic(fmt.Sprintf("Failed to seed users table: %v", err))
	}

	fmt.Println("Users table seeded successfully.")
}

func SeedFakeUsers(minAge, maxAge, count int) {
	if minAge < 1 || maxAge < minAge || count <= 0 {
		log.Fatalf("Invalid seeding parameters: minAge=%d, maxAge=%d, count=%d", minAge, maxAge, count)
		return
	}

	startUserCode, err := models.GetMaxUserCode()
	if err != nil {
		log.Fatal("error generating code: ", err)
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 1; i <= count; i++ {
		password, err := hashPassword("abcd1234")
		if err != nil {
			log.Fatalf("Failed to hash password for Admin: %v", err)
		}
		email := fmt.Sprintf("mohira.shaikh+%v@finchwebtech.com", startUserCode+i)

		// assignOrganizer := r.Intn(4) == 0 // 25% chance
		// var role string
		// if assignOrganizer {
		// 	role = "organization"
		// } else {
		// 	role = "individual"
		// }

		user := models.User{
			Name:     faker.Name(),
			MobileNo: fmt.Sprintf("9%09d", r.Intn(1000000000)),
			Email:    email,
			RoleSlug: "individual",
			Password: password,
			UserCode: startUserCode + i,
		}
		user.Details = &models.UserDetails{}

		createdUser, err := models.CreateUser(&user)
		if err != nil {
			log.Fatal("Error creating user: ", err)
			return
		}

		// Update user status
		_, err = database.DB.Exec(`
			UPDATE users
			SET otp_status = 'Verified',
				email_status = 'Verified',
				status = 'Active',
				updated_at = $2
			WHERE id = $1
		`, createdUser.ID, time.Now())
		if err != nil {
			log.Printf("Failed to update statuses for user %d: %v", createdUser.ID, err)
		}

		// Generate DOB within age range
		age := r.Intn(maxAge-minAge+1) + minAge
		randomDOB := time.Now().AddDate(-age, 0, -r.Intn(365))
		dob := randomDOB.Format("2006-01-02")

		// Random gender
		genders := []string{"Male", "Female"}
		gender := genders[rand.Intn(len(genders))]

		// Update DOB and gender
		_, err = database.DB.Exec(`
			UPDATE user_details
			SET dob = $1, gender = $2, updated_at = $3
			WHERE user_id = $4
		`, dob, gender, time.Now(), createdUser.ID)
		if err != nil {
			log.Printf("Failed to update DOB and gender for user_id %d: %v", createdUser.ID, err)
		}
	}

	log.Println(count, " fake users seeded successfully.")
}

func SimulateStateCitySelection(startUserId int, endUserId int) {
	tx, _ := database.DB.Begin()
	for i := startUserId; i > 0 && i <= endUserId; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		stateId := r.Intn(12)
		stateId++
		cityArr, err := models.GetCityByState(int64(stateId))
		if err != nil {
			fmt.Println("error fetching cities")
			tx.Rollback()
			return
		}

		fmt.Println("\nlength", len(cityArr), "stateId:", stateId)

		j := r.Intn((len(cityArr) / 3) + 1)

		cityId := cityArr[j].ID
		if cityId != int64(len(cityArr)-1) {
			cityId++
		}
		query := `UPDATE public.user_details SET
			state = $1,
			city = $2
		WHERE user_id = $3;`
		_, err = tx.Exec(query, stateId, cityId, i)
		if err != nil {
			fmt.Println(err)
			tx.Rollback()
			return
		}
		fmt.Printf("user:%d, city:%d", i, cityId)
	}
	tx.Commit()
}
