package responses

import "github.com/GoloisaNinja/go-bourbon-api/pkg/models"

type CollectionResponse struct {
	Collection      *models.Collection          `json:"collection"`
	UserCollections []*models.UserCollectionRef `json:"user_collections"`
}
