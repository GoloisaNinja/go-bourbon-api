package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/helpers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"net/http"
	"time"
)

// declare and set collections to collection vars
var usersCollection = db.GetCollection(
	db.DB,
	"users",
)

type JWTCustomClaims struct {
	UserId string
	jwt.StandardClaims
}

func GenerateAuthToken(userId string) (string, error) {
	jwtSecret := []byte(helpers.GetGoDotEnv("JWT_SECRET"))
	t := time.Now()
	claims := JWTCustomClaims{
		userId,
		jwt.StandardClaims{
			Issuer:   "helloBourbon",
			IssuedAt: t.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, err
}

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
	userFromCtx := r.Context().Value("user")
	newUser := userFromCtx.(*models.User)
	tokenFromCtx := newUser.Tokens[0].Token
	_, err := usersCollection.InsertOne(context.TODO(), userFromCtx)
	if err != nil {
		responses.RespondWithError(
			w, http.StatusInternalServerError, "error",
			err.Error(),
		)
		return
	}
	cleanResponse := responses.CleanUserResponse{
		User:  newUser,
		Token: tokenFromCtx,
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
	token, tErr := GenerateAuthToken(verifiedUser.ID.Hex())
	if tErr != nil {
		responses.RespondWithError(
			w, http.StatusInternalServerError,
			"error", tErr.Error(),
		)
		return
	}
	dbToken := models.UserTokenRef{
		Token: token,
	}
	filter := bson.M{"_id": verifiedUser.ID}
	tokenUpdate := bson.M{"$push": bson.M{"tokens": dbToken}}
	_, updateErr := usersCollection.UpdateOne(
		context.TODO(), filter,
		tokenUpdate,
	)
	if updateErr != nil {
		responses.RespondWithError(
			w, http.StatusInternalServerError,
			"error", updateErr.Error(),
		)
	}
	cleanResponse := responses.CleanUserResponse{
		User:  verifiedUser,
		Token: token,
	}
	json.NewEncoder(w).Encode(
		responses.UserResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": cleanResponse},
		},
	)
}

func LogoutUser(w http.ResponseWriter, r *http.Request) {
	authFromCtx := r.Context().Value("authContext")
	auth := authFromCtx.(*models.AuthContext)
	id, _ := primitive.ObjectIDFromHex(auth.UserId)
	t := auth.Token
	filter := bson.D{{"_id", id}, {"tokens.token", t}}
	update := bson.M{"$pull": bson.M{"tokens": bson.D{{"token", t}}}}
	result, err := usersCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		responses.RespondWithError(
			w, http.StatusBadRequest, "error", err.Error(),
		)
		return
	}
	if result.MatchedCount != 1 {
		responses.RespondWithError(
			w, http.StatusBadRequest, "error",
			"bad request",
		)
		return
	}
	type response struct {
		Status  int
		Message string
	}
	json.NewEncoder(w).Encode(response{Status: 200, Message: "logged out"})
}
