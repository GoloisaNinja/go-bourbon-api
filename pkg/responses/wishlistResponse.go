package responses

import "github.com/GoloisaNinja/go-bourbon-api/pkg/models"

type WishlistResponse struct {
	Wishlist      *models.Collection        `json:"wishlist"`
	UserWishlists []*models.UserWishlistRef `json:"user_wishlists"`
}
