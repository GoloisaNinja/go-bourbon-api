package handlers

import (
	"context"
	"encoding/json"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/helpers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"net/http"
)

var reviewsCollection = db.GetCollection(db.DB, "reviews")

func GetReviewById(w http.ResponseWriter, r *http.Request) {
	var review models.UserReview
	var er responses.ErrorResponse
	var sr responses.StandardResponse
	params := mux.Vars(r)
	reviewId, rErr := primitive.ObjectIDFromHex(params["id"])
	if rErr != nil {
		er.Respond(w, 400, "error", rErr.Error())
	}
	filter := bson.M{"_id": reviewId}
	err := reviewsCollection.FindOne(context.TODO(), filter).Decode(&review)
	if err != nil {
		er.Respond(w, 400, "error", err.Error())
		return
	}
	sr.Respond(w, 200, "success", review)
}

// GetAllReviewsByFilterId is able to return all reviews based on either
// a user Id or a bourbon Id passed in the params - result is an slice
// of userreview models
func GetAllReviewsByFilterId(w http.ResponseWriter, r *http.Request) {
	var er responses.ErrorResponse
	var sr responses.StandardResponse
	var reviews []models.UserReview
	params := mux.Vars(r)
	filterType := params["fType"]
	id, bErr := primitive.ObjectIDFromHex(params["id"])
	if bErr != nil {
		er.Respond(w, 400, "error", bErr.Error())
		return
	}
	var filter bson.M
	if filterType == "bourbon" {
		filter = bson.M{"bourbon_id": id}
	} else {
		filter = bson.M{"user.id": id}
	}

	cursor, cErr := reviewsCollection.Find(context.TODO(), filter)
	if cErr != nil {
		er.Respond(w, 400, "error", cErr.Error())
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		var review models.UserReview
		err := cursor.Decode(&review)
		if err != nil {
			er.Respond(w, 500, "error", err.Error())
			return
		}
		reviews = append(reviews, review)
	}
	if curErr := cursor.Err(); curErr != nil {
		er.Respond(w, 500, "error", curErr.Error())
	}
	if len(reviews) > 0 {
		sr.Respond(w, 200, "success", reviews)
	} else {
		er.Respond(w, 404, "error", "not found")
	}
}

func CreateReview(w http.ResponseWriter, r *http.Request) {
	var er responses.ErrorResponse
	var user models.User
	var uRef models.UserRef
	var review models.UserReview
	var sr responses.StandardResponse
	userId, err := helpers.GetUserIdFromAuthCtx(r.Context())
	if err != nil {
		er.Respond(w, 500, "error", err.Error())
		return
	}
	filter := bson.M{"_id": userId}
	uErr := usersCollection.FindOne(context.TODO(), filter).Decode(&user)
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	uRef.ID = user.ID
	uRef.Username = user.Username
	rBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(rBody, &review)
	rFilter := bson.M{"bourbon_id": review.BourbonID, "user.id": user.ID}
	count, cErr := reviewsCollection.CountDocuments(context.TODO(), rFilter)
	if cErr != nil {
		er.Respond(w, 500, "error", cErr.Error())
		return
	}
	if count > 0 {
		er.Respond(w, 400, "error", "user already reviewed this bourbon")
		return
	}
	review.ID = primitive.NewObjectID()
	review.User = &uRef
	_, rErr := reviewsCollection.InsertOne(context.TODO(), review)
	if rErr != nil {
		er.Respond(w, 500, "error", rErr.Error())
		return
	}
	sr.Respond(w, 200, "success", review)
}
