package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Collection struct should work for both the bourbon collection and the bourbon wishlist
// data base collection types
type Collection struct {
	ID        primitive.ObjectID `bson:"_id" json:"_id"`
	User      *UserRef           `bson:"user" json:"user"`
	Name      string             `bson:"name" json:"name"`
	Private   bool               `bson:"private" json:"private"`
	Bourbons  []*Bourbon         `bson:"bourbons" json:"bourbons"`
	CreatedAt primitive.DateTime `bson:"createdAt" json:"createdAt"`
	UpdatedAt primitive.DateTime `bson:"updatedAt" json:"updatedAt"`
}

func (c *Collection) Build(uId primitive.ObjectID, un, n string, p bool) {
	c.ID = primitive.NewObjectID()
	c.User = &UserRef{
		ID:       uId,
		Username: un,
	}
	c.Name = n
	c.Private = p
	c.Bourbons = make([]*Bourbon, 0)
	c.CreatedAt = primitive.NewDateTimeFromTime(time.Now())
	c.UpdatedAt = primitive.NewDateTimeFromTime(time.Now())
}
