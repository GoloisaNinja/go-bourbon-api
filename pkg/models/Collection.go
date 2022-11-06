package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Collection struct {
	ID       primitive.ObjectID `bson:"_id" json:"_id"`
	User     *UserRef           `bson:"user" json:"user"`
	Name     string             `bson:"name" json:"name"`
	Private  bool               `bson:"private" json:"private"`
	Bourbons []*Bourbon         `bson:"bourbons" json:"bourbons"`
}
