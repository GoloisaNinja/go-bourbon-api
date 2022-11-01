package responses

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Status  int                    `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func RespondWithError(w http.ResponseWriter, s int, m string, d interface{}) {
	response := ErrorResponse{
		Status:  s,
		Message: m,
		Data:    map[string]interface{}{"data": d},
	}
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(response)
}
