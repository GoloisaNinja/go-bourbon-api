package middleware

import (
	"context"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

var keysCollection = db.GetCollection(db.DB, "keys")

func ApiAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var er responses.ErrorResponse
		q := r.URL.Query()
		str := q.Get("apiKey")
		key, kErr := primitive.ObjectIDFromHex(str)
		if kErr != nil {
			er.Respond(w, 400, "error", "invalid apikey")
			return
		}
		filter := bson.M{"_id": key, "active": true}
		keyCount, err := keysCollection.CountDocuments(context.TODO(), filter)
		if err != nil {
			er.Respond(w, 500, "error", err.Error())
			return
		}
		if keyCount != 1 {
			er.Respond(w, 401, "error", "unauthorized - requires valid api key")
			return
		}
		next.ServeHTTP(w, r)
	})
}
