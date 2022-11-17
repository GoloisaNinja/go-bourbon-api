package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/handlers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/helpers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"regexp"
)

var usersCollection = db.GetCollection(
	db.DB,
	"users",
)

func isValidEmail(e string) bool {
	_, err := mail.ParseAddress(e)
	return err == nil
}

func isExistingUser(e string) bool {
	filter := bson.M{"email": e}
	var existingUser models.User
	err := usersCollection.FindOne(context.TODO(), filter).Decode(&existingUser)
	return err == nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func AuthMiddleware(next http.Handler) http.Handler {
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
					return []byte(helpers.GetGoDotEnv("JWT_SECRET")), nil
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

func NewUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var er responses.ErrorResponse
			reqBody, _ := ioutil.ReadAll(r.Body)
			var reqResult models.RegisterUserRequest
			// try to unmarshal the request
			err := json.Unmarshal(reqBody, &reqResult)
			if err != nil {
				er.Respond(w, 400, "error", err.Error())
				return
			}
			// check if the request had empty body
			if reqResult.Email == "" || reqResult.Password == "" {
				er.Respond(w, 400, "error", err.Error())
				return
			}
			validEmail := isValidEmail(reqResult.Email)
			alreadyExists := isExistingUser(reqResult.Email)
			if !validEmail || alreadyExists {
				er.Respond(w, 400, "error", "user or email invalid")
				return
			}
			// hash new user password from request
			hashed, hashErr := hashPassword(reqResult.Password)
			if hashErr != nil {
				er.Respond(w, 500, "error", hashErr.Error())
				return
			}
			var newUser models.User
			// generate our user primitive object Id
			uid := primitive.NewObjectID()
			// generate JWT
			token, tErr := handlers.GenerateAuthToken(uid.Hex())
			if tErr != nil {
				er.Respond(w, 500, "error", tErr.Error())
				return
			}
			newUser.Build(uid, reqResult.Username, reqResult.Email, hashed, token)
			ctx := context.WithValue(r.Context(), "user", &newUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}
