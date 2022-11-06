package responses

import "github.com/GoloisaNinja/go-bourbon-api/pkg/models"

type CleanUserResponse struct {
	User  *models.User `json:"user"`
	Token string       `json:"token"`
}
