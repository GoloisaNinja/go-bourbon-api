package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserReview struct {
	ID          primitive.ObjectID `bson:"_id" json:"_id,omitempty"`
	User        *UserRef           `bson:"user" json:"user,omitempty"`
	BourbonName string             `bson:"bourbonName" json:"bourbonName"`
	BourbonID   primitive.ObjectID `bson:"bourbon_id" json:"bourbon_id"`
	ReviewTitle string             `bson:"reviewTitle" json:"reviewTitle"`
	ReviewScore string             `bson:"reviewScore" json:"reviewScore"`
	ReviewText  string             `bson:"reviewText" json:"reviewText"`
}
