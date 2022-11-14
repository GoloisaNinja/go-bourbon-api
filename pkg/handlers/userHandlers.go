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
	newUser := r.Context().Value("user").(*models.User)
	tokenFromCtx := newUser.Tokens[0].Token
	_, err := usersCollection.InsertOne(context.TODO(), newUser)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	cleanResponse := responses.CleanUserResponse{
		User:  newUser,
		Token: tokenFromCtx,
	}
	var ur responses.StandardResponse
	ur.Respond(w, 200, "success", cleanResponse)
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req models.UserLoginRequest
	reqErr := json.Unmarshal(reqBody, &req)
	if reqErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", reqErr.Error())
		return
	}
	if req.Email == "" || req.
		Password == "" {
		missing := errors.New("bad request")
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", missing.Error())
		return
	}
	verifiedUser, vError := findByCredentials(
		req.Email,
		req.Password,
	)
	if vError != nil {
		var er responses.ErrorResponse
		er.Respond(w, 401, "error", vError.Error())
		return
	}
	token, tErr := GenerateAuthToken(verifiedUser.ID.Hex())
	if tErr != nil {
		var er responses.ErrorResponse
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
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
	}
	cleanResponse := responses.CleanUserResponse{
		User:  verifiedUser,
		Token: token,
	}
	var ur responses.StandardResponse
	ur.Respond(w, 200, "success", cleanResponse)
}

func LogoutUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	id := ctx.UserId
	t := ctx.Token
	filter := bson.D{{"_id", id}, {"tokens.token", t}}
	update := bson.M{"$pull": bson.M{"tokens": bson.D{{"token", t}}}}
	result, err := usersCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", err.Error())
		return
	}
	if result.MatchedCount != 1 {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bad request")
		return
	}
	var ur responses.StandardResponse
	ur.Respond(w, 200, "logged out", "logout successful")
}
