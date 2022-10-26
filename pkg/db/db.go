package db

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"time"
)

func getGoDotEnv(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return os.Getenv(key)
}

// Connection URI
var uri = getGoDotEnv("PROD_MONGODB_URI")
var testUri = getGoDotEnv("DEV_MONGODB_URI")

func ConnectDB() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(testUri))
	if err != nil {
		log.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	//ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB")
	return client

}

var DB *mongo.Client = ConnectDB()

func GetCollection(
	client *mongo.Client, collectionName string,
) *mongo.Collection {
	coll := client.Database("gobourbon").Collection(collectionName)
	return coll
}
