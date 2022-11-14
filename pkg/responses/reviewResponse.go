package responses

import "github.com/GoloisaNinja/go-bourbon-api/pkg/models"

type ReviewResponse struct {
	Review      *models.UserReview      `json:"review"`
	UserReviews []*models.UserReviewRef `json:"user_reviews"`
}
