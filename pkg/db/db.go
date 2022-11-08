package db

import (
	"context"
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/config"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

// Connection URI
var uri = helpers.GetGoDotEnv("PROD_MONGODB_URI")
var testUri = helpers.GetGoDotEnv("DEV_MONGODB_URI")

var app config.AppConfig

func ConnectDB() *mongo.Client {
	app.IsProduction = false
	var uriToUse string
	if app.IsProduction {
		uriToUse = uri
	} else {
		uriToUse = testUri
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(uriToUse))
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
