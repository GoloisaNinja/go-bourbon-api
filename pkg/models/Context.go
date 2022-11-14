package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type AuthContext struct {
	UserId   primitive.ObjectID
	Username string
	Token    string
}
