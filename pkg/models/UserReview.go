package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserReview struct {
	User        *UserRef           `bson:"user" json:"user"`
	BourbonName string             `bson:"bourbonName" json:"bourbonName"`
	BourbonID   primitive.ObjectID `bson:"bourbon_id" json:"bourbon_id"`
	ReviewTitle string             `bson:"reviewTitle" json:"reviewTitle"`
	ReviewScore int                `bson:"reviewScore" json:"reviewScore"`
	ReviewText  string             `bson:"reviewText" json:"reviewText"`
}
