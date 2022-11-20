package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type APIKey struct {
	ID         primitive.ObjectID `bson:"_id"`
	AppName    string             `bson:"app_name"`
	Active     bool               `bson:"active"`
	CreatedAt  primitive.DateTime `bson:"createdAt"`
	LastAccess primitive.DateTime `bson:"lastAccess"`
}
