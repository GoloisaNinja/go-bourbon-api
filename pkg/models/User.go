package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserCollectionRef struct {
	CollectionID   primitive.ObjectID `bson:"collection_id" json:"collection_id"`
	CollectionName string             `bson:"collection_name" json:"collection_name"`
	Bourbons       []*BourbonsRef     `bson:"bourbons" json:"bourbons"`
}

func (u *UserCollectionRef) Build(cId primitive.ObjectID, n string) {
	u.CollectionID = cId
	u.CollectionName = n
	u.Bourbons = make([]*BourbonsRef, 0)
}

type UserReviewRef struct {
	ReviewID    string `bson:"review_id" json:"review_id"`
	ReviewTitle string `bson:"review_title" json:"review_title"`
}

type UserWishlistRef struct {
	WishlistID   string         `bson:"wishlist_id" json:"wishlist_id"`
	WishlistName string         `bson:"wishlist_name" json:"wishlist_name"`
	Bourbons     []*BourbonsRef `bson:"bourbons" json:"bourbons"`
}

type UserTokenRef struct {
	Token string `bson:"token" json:"token"`
}

type User struct {
	ID          primitive.ObjectID   `bson:"_id" json:"_id"`
	Username    string               `bson:"username" json:"username"`
	Email       string               `bson:"email" json:"email"`
	Password    string               `bson:"password" json:"-"`
	Collections []*UserCollectionRef `bson:"collections" json:"collections"`
	Reviews     []*UserReviewRef     `bson:"reviews" json:"reviews"`
	Wishlists   []*UserWishlistRef   `bson:"wishlists" json:"wishlists"`
	Tokens      []*UserTokenRef      `bson:"tokens" json:"-"`
}

type UserLoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
