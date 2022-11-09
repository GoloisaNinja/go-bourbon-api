package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Collection struct {
	ID       primitive.ObjectID `bson:"_id" json:"_id"`
	User     *UserRef           `bson:"user" json:"user"`
	Name     string             `bson:"name" json:"name"`
	Private  bool               `bson:"private" json:"private"`
	Bourbons []*Bourbon         `bson:"bourbons" json:"bourbons"`
}

func (c *Collection) Build(uId primitive.ObjectID, n string, p bool) {
	c.ID = primitive.NewObjectID()
	c.User = &UserRef{
		ID:       uId,
		Username: n,
	}
	c.Name = n
	c.Private = p
	c.Bourbons = make([]*Bourbon, 0)
}
