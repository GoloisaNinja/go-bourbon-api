package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type UserReview struct {
	ID          primitive.ObjectID `bson:"_id" json:"_id,omitempty"`
	User        *UserRef           `bson:"user" json:"user,omitempty"`
	BourbonName string             `bson:"bourbonName" json:"bourbonName"`
	BourbonID   primitive.ObjectID `bson:"bourbon_id" json:"bourbon_id"`
	ReviewTitle string             `bson:"reviewTitle" json:"reviewTitle"`
	ReviewScore string             `bson:"reviewScore" json:"reviewScore"`
	ReviewText  string             `bson:"reviewText" json:"reviewText"`
	CreatedAt   primitive.DateTime `bson:"createdAt" json:"createdAt"`
	UpdatedAt   primitive.DateTime `bson:"updatedAt" json:"updatedAt"`
}

func (r *UserReview) Build(b Bourbon, uId primitive.ObjectID, uname string) {
	r.ID = primitive.NewObjectID()
	r.User = &UserRef{
		ID:       uId,
		Username: uname,
	}
	r.BourbonName = b.Title
	r.BourbonID = b.ID
	r.CreatedAt = primitive.NewDateTimeFromTime(time.Now())
	r.UpdatedAt = primitive.NewDateTimeFromTime(time.Now())
}
