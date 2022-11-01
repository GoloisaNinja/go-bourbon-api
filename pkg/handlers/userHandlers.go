package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"net/http"
)

// declare and set collections to collection vars
var usersCollection = db.GetCollection(
	db.DB,
	"users",
)

func verifyPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func findByCredentials(email, password string) (
	*models.
		User, error,
) {
	filter := bson.M{"email": email}
	var result bson.M
	var user models.User
	err := usersCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return &models.User{}, err
	}
	requestPass := fmt.Sprint(result["password"])
	if verifyPasswordHash(password, requestPass) {
		byteResult, _ := json.Marshal(result)
		err := json.Unmarshal(byteResult, &user)
		if err != nil {
			return &models.User{}, err
		}
		return &user, nil
	} else {
		err = errors.New("unauthorized")
		return &models.User{}, err
	}
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		responses.RespondWithError(
			w, http.StatusMethodNotAllowed, "method not allowed",
			"request method not allowed on this endpoint",
		)
		return
	}
	userFromReq := r.Context().Value("user")
	newUser := userFromReq.(*models.User)
	_, err := usersCollection.InsertOne(context.TODO(), userFromReq)
	if err != nil {
		responses.RespondWithError(
			w, http.StatusInternalServerError, "error",
			err.Error(),
		)
		return
	}
	cleanResponse := responses.CleanUserResponse{
		User:  newUser,
		Token: "fake-token-for-now",
	}
	json.NewEncoder(w).Encode(
		responses.UserResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": cleanResponse},
		},
	)
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req models.UserLoginRequest
	reqErr := json.Unmarshal(reqBody, &req)
	if reqErr != nil {
		responses.RespondWithError(
			w, http.StatusBadRequest, "error",
			reqErr.Error(),
		)
		return
	}
	if req.Email == "" || req.
		Password == "" {
		missingCreds := errors.New("bad request")
		responses.RespondWithError(
			w, http.StatusBadRequest, "error",
			missingCreds.Error(),
		)
		return
	}
	verifiedUser, validationError := findByCredentials(
		req.Email,
		req.Password,
	)
	if validationError != nil {
		responses.RespondWithError(
			w, http.StatusUnauthorized,
			"unauthorized", "bad email or password",
		)
		return
	}
	cleanResponse := responses.CleanUserResponse{
		User:  verifiedUser,
		Token: "fake-Login-token-for-now",
	}
	json.NewEncoder(w).Encode(
		responses.UserResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": cleanResponse},
		},
	)
}
