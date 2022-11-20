package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// declare and set collections to collection vars
var usersCollection = db.GetCollection(
	db.DB,
	"users",
)

var jwtSec = os.Getenv("JWT_SECRET")

type JWTCustomClaims struct {
	UserId string
	jwt.StandardClaims
}

func GenerateAuthToken(userId string) (string, error) {
	jwtSecret := []byte(jwtSec)
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
	newUser := r.Context().Value("user").(*models.User)
	tokenFromCtx := newUser.Tokens[0].Token
	_, err := usersCollection.InsertOne(context.TODO(), newUser)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	ur := responses.UserTokenResponse{
		User:  newUser,
		Token: tokenFromCtx,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", ur)
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	var er responses.ErrorResponse
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req models.RegisterUserRequest
	reqErr := json.Unmarshal(reqBody, &req)
	if reqErr != nil {
		er.Respond(w, 400, "error", reqErr.Error())
		return
	}
	if req.Email == "" || req.
		Password == "" {
		missing := errors.New("bad request")
		er.Respond(w, 500, "error", missing.Error())
		return
	}
	verifiedUser, vError := findByCredentials(
		req.Email,
		req.Password,
	)
	if vError != nil {
		er.Respond(w, 401, "error", vError.Error())
		return
	}
	token, tErr := GenerateAuthToken(verifiedUser.ID.Hex())
	if tErr != nil {
		er.Respond(w, 500, "error", tErr.Error())
		return
	}
	dbToken := models.UserTokenRef{
		Token: token,
	}
	filter := bson.M{"_id": verifiedUser.ID}
	tokenUpdate := bson.M{"$push": bson.M{"tokens": dbToken}}
	_, uErr := usersCollection.UpdateOne(
		context.TODO(), filter,
		tokenUpdate,
	)
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	updatedTime := primitive.NewDateTimeFromTime(time.Now())
	tUpdate := bson.M{"$set": bson.M{"updatedAt": updatedTime}}
	_, tUErr := usersCollection.UpdateOne(context.TODO(), filter, tUpdate)
	if tUErr != nil {
		er.Respond(w, 500, "error", "updated failed")
		return
	}
	ur := responses.UserTokenResponse{
		User:  verifiedUser,
		Token: token,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", ur)
}

func LogoutUser(w http.ResponseWriter, r *http.Request) {
	var er responses.ErrorResponse
	var sr responses.StandardResponse
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	id := ctx.UserId
	t := ctx.Token
	filter := bson.D{{"_id", id}, {"tokens.token", t}}
	update := bson.M{"$pull": bson.M{"tokens": bson.D{{"token", t}}}}
	result, err := usersCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		er.Respond(w, 400, "error", err.Error())
		return
	}
	if result.MatchedCount != 1 {
		er.Respond(w, 400, "error", "bad request")
		return
	}
	updatedTime := primitive.NewDateTimeFromTime(time.Now())
	tFilter := bson.M{"_id": id}
	tUpdate := bson.M{"$set": bson.M{"updatedAt": updatedTime}}
	_, tUErr := usersCollection.UpdateOne(context.TODO(), tFilter, tUpdate)
	if tUErr != nil {
		er.Respond(w, 500, "error", "time update failed")
		return
	}
	sr.Respond(w, 200, "logged out", "logout successful")
}
