package helpers

import (
	"context"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func GetUserIdFromAuthCtx(ctx context.Context) (primitive.ObjectID, error) {
	a := ctx.Value("authContext").(*models.AuthContext)
	id, err := primitive.ObjectIDFromHex(a.UserId)
	return id, err
}
