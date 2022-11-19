package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

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
	ReviewID    primitive.ObjectID `bson:"review_id" json:"review_id"`
	ReviewTitle string             `bson:"review_title" json:"review_title"`
}

type UserWishlistRef struct {
	WishlistID   primitive.ObjectID `bson:"wishlist_id" json:"wishlist_id"`
	WishlistName string             `bson:"wishlist_name" json:"wishlist_name"`
	Bourbons     []*BourbonsRef     `bson:"bourbons" json:"bourbons"`
}

func (u *UserWishlistRef) Build(wId primitive.ObjectID, n string) {
	u.WishlistID = wId
	u.WishlistName = n
	u.Bourbons = make([]*BourbonsRef, 0)
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
	CreatedAt   primitive.DateTime   `bson:"createdAt" json:"createdAt"`
	UpdatedAt   primitive.DateTime   `bson:"updatedAt" json:"updatedAt"`
}

func (u *User) Build(i primitive.ObjectID, n, e, hp, t string) {
	u.ID = i
	u.Username = n
	u.Email = e
	u.Password = hp
	u.Collections = make([]*UserCollectionRef, 0)
	u.Reviews = make([]*UserReviewRef, 0)
	u.Wishlists = make([]*UserWishlistRef, 0)
	u.Tokens = append(u.Tokens, &UserTokenRef{
		Token: t,
	})
	u.CreatedAt = primitive.NewDateTimeFromTime(time.Now())
	u.UpdatedAt = primitive.NewDateTimeFromTime(time.Now())
}

type RegisterUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
