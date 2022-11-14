package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserRef struct {
	ID       primitive.ObjectID `bson:"id" json:"id"`
	Username string             `bson:"username" json:"username"`
}

type BourbonsRef struct {
	BourbonID primitive.ObjectID `bson:"bourbon_id" json:"bourbon_id"`
}

// CollectionRequest struct will be the same for a collection collection or a wishlist collection
type CollectionRequest struct {
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

func (c *CollectionRequest) FillDefaults() {
	if c.Private != true && c.Private != false {
		c.Private = true
	}
	if c.Name == "" {
		c.Name = "Unnamed Collection"
	}
}

type ReviewRequest struct {
	ReviewTitle string `json:"reviewTitle"`
	ReviewScore string `json:"reviewScore"`
	ReviewText  string `json:"reviewText"`
}
