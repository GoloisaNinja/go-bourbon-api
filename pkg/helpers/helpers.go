package helpers

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

func GetGoDotEnv(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return os.Getenv(key)
}
