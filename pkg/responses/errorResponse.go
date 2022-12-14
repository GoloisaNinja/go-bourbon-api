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

func (r *ErrorResponse) Build(s int, m string, d interface{}) {
	r.Status = s
	r.Message = m
	r.Data = map[string]interface{}{"data": d}
}

func (r ErrorResponse) Respond(w http.ResponseWriter, status int, m string, d interface{}) {
	r.Status = status
	r.Message = m
	r.Data = map[string]interface{}{"data": d}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(r)
}
