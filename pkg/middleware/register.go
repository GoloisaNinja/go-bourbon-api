package middleware

import (
	"context"
	"encoding/json"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/handlers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"net/http"
	"net/mail"
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

func Register(next http.Handler) http.Handler {
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
