package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var AESKey []byte // Exported variable to store the AES key

func LoadConfig() {
	// Load .env file (optional, if needed)
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found. Ensure environment variables are set.")
	}

	// Retrieve AES key from environment
	key := os.Getenv("AES_KEY")
	if len(key) == 0 {
		log.Fatal("AES_KEY is not set or is empty")
	}

	// Set AESKey for external use
	AESKey = []byte(key)

	// log.Println("AES_KEY loaded successfully")
}
