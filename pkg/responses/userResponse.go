package responses

import "github.com/GoloisaNinja/go-bourbon-api/pkg/models"

type UserResponse struct {
	Status  int                    `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type CleanUserResponse struct {
	User  *models.User `json:"user"`
	Token string       `json:"token"`
}
