package middleware

import (
	"context"
	"encoding/json"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"net/http"
	"net/mail"
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

func NewUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			reqBody, _ := ioutil.ReadAll(r.Body)
			var reqResult models.UserLoginRequest
			// try to unmarshal the request
			err := json.Unmarshal(reqBody, &reqResult)
			if err != nil {
				responses.RespondWithError(
					w, http.StatusBadRequest, "error",
					err.Error(),
				)
				return
			}
			// check if the request had empty body
			if reqResult.Email == "" || reqResult.Password == "" {
				responses.RespondWithError(
					w, http.StatusBadRequest, "error",
					err.Error(),
				)
				return
			}
			validEmail := isValidEmail(reqResult.Email)
			alreadyExists := isExistingUser(reqResult.Email)
			if !validEmail || alreadyExists {
				responses.RespondWithError(
					w, http.StatusBadRequest, "error",
					"bad user or email",
				)
				return
			}
			// hash new user password from request
			hashed, hashErr := hashPassword(reqResult.Password)
			if hashErr != nil {
				responses.RespondWithError(
					w, http.StatusInternalServerError,
					"error", hashErr.Error(),
				)
			}
			var newUser models.User
			newUser.ID = primitive.NewObjectID()
			newUser.Username = reqResult.Username
			newUser.Email = reqResult.Email
			newUser.Password = hashed
			newUser.Collections = make([]*models.UserCollectionRef, 0)
			newUser.Reviews = make([]*models.UserReviewRef, 0)
			newUser.Wishlists = make([]*models.UserWishlistRef, 0)
			newUser.Tokens = make([]*models.UserTokenRef, 0)
			ctx := context.WithValue(r.Context(), "user", &newUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}
