package responses

import (
	"encoding/json"
	"net/http"
)

type StandardResponse struct {
	Status  int                    `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func (r StandardResponse) Respond(w http.ResponseWriter, status int, m string, d interface{}) {
	r.Status = status
	r.Message = m
	r.Data = map[string]interface{}{"data": d}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(r)
}
