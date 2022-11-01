package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserRef struct {
	ID       primitive.ObjectID `bson:"id" json:"id"`
	Username string             `bson:"username" json:"username"`
}

type BourbonsRef struct {
	BourbonID primitive.ObjectID `bson:"bourbon_id" json:"bourbon_id"`
}
