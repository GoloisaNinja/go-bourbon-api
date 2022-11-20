package middleware

import (
	"context"
	"errors"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"os"
	"regexp"
)

var usersCollection = db.GetCollection(
	db.DB,
	"users",
)

var jwSec = os.Getenv("JWT_SECRET")

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// extract token from request header under "Authorization" where
			// token is formatted as "Bearer: <token>"
			var er responses.ErrorResponse
			x, err := regexp.Compile(`^(?P<B>Bearer\s+)(?P<T>.*)$`)
			if err != nil {
				er.Respond(w, 401, "error", "authorization failed")
				return
			}
			authHeader := x.FindStringSubmatch(r.Header.Get("Authorization"))
			if len(authHeader) != 3 {
				er.Respond(w, 401, "error", "authorization process failed")
				return
			}
			tokenIndex := x.SubexpIndex("T")
			tokenString := authHeader[tokenIndex]
			// verify the token
			token, vErr := jwt.Parse(
				tokenString,
				func(token *jwt.Token) (interface{}, error) {
					_, ok := token.Method.(*jwt.SigningMethodHMAC)
					if !ok {
						authErr := errors.New("unauthorized")
						return nil, authErr
					}
					return []byte(jwSec), nil
				},
			)
			if vErr != nil {
				er.Respond(w, 401, "error", "unauthorized")
				return
			}
			var userId string
			claims, claimOk := token.Claims.(jwt.MapClaims)
			if claimOk && token.Valid {
				userId = claims["UserId"].(string)
			}
			userIdAsPrimitive, iErr := primitive.ObjectIDFromHex(userId)
			if iErr != nil {
				er.Respond(w, 500, "error", iErr.Error())
				return
			}
			// get a full user to include the username in the context to alleviate pulling
			// a full user in certain handlers that only need a username for a ref
			var user models.User
			filter := bson.M{"_id": userIdAsPrimitive}
			uErr := usersCollection.FindOne(context.TODO(), filter).Decode(&user)
			if uErr != nil {
				er.Respond(w, 500, "error", uErr.Error())
				return
			}
			authContext := models.AuthContext{
				UserId:   userIdAsPrimitive,
				Username: user.Username,
				Token:    tokenString,
			}
			ctx := context.WithValue(r.Context(), "authContext", &authContext)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}
