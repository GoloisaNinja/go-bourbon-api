package responses

import (
	"encoding/json"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"net/http"
)

type StandardResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (r StandardResponse) Respond(w http.ResponseWriter, status int, m string, d interface{}) {
	r.Status = status
	r.Message = m
	r.Data = d
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(r)
}

// user responses

type UserTokenResponse struct {
	User  *models.User `json:"user"`
	Token string       `json:"token"`
}

// bourbon responses

type SingleBourbonResponse struct {
	Bourbon *models.Bourbon `json:"bourbon"`
}

type BourbonsResponse struct {
	Bourbons     []*models.Bourbon `json:"bourbons"`
	TotalRecords int               `json:"total_records"`
}

// collection responses

type CollectionResponse struct {
	Collection      *models.Collection          `json:"collection"`
	UserCollections []*models.UserCollectionRef `json:"user_collections"`
}

type CollectionsResponse struct {
	Collections []*models.Collection `json:"collections"`
}

// wishlist responses

type WishlistResponse struct {
	Wishlist      *models.Collection        `json:"wishlist"`
	UserWishlists []*models.UserWishlistRef `json:"user_wishlists"`
}

type WishlistsResponse struct {
	Wishlists []*models.Collection
}

// review responses

type ReviewResponse struct {
	Review      *models.UserReview      `json:"review"`
	UserReviews []*models.UserReviewRef `json:"user_reviews"`
}
