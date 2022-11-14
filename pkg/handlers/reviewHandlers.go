package handlers

import (
	"context"
	"encoding/json"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	// user models
	var user models.User
	// review models
	var rRef models.UserReviewRef
	var review models.UserReview
	// bourbon model
	var bourbon models.Bourbon
	// response models
	var rr responses.ReviewResponse
	var sr responses.StandardResponse
	// pull userId and username from context
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	userId := ctx.UserId
	username := ctx.Username
	rBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(rBody, &review)
	// check the bourbon_id against the bourbons in the db - is it a valid id?
	bFilter := bson.M{"_id": review.BourbonID}
	bErr := bourbonsCollection.FindOne(context.TODO(), bFilter).Decode(&bourbon)
	if bErr != nil {
		er.Respond(w, 404, "error", "bourbon to be reviewed not found")
		return
	}
	rFilter := bson.M{"bourbon_id": review.BourbonID, "user.id": userId}
	count, cErr := reviewsCollection.CountDocuments(context.TODO(), rFilter)
	if cErr != nil {
		er.Respond(w, 500, "error", cErr.Error())
		return
	}
	if count > 0 {
		er.Respond(w, 400, "error", "user already reviewed this bourbon")
		return
	}
	review.Build(bourbon, userId, username)
	// user ref for the review model
	// insert the review from the request
	_, rErr := reviewsCollection.InsertOne(context.TODO(), review)
	if rErr != nil {
		er.Respond(w, 500, "error", rErr.Error())
		return
	}
	// review ref needed for the user model
	rRef.ReviewID = review.ID
	rRef.ReviewTitle = review.ReviewTitle
	uFilter := bson.M{"_id": userId}
	uUpdate := bson.M{"$push": bson.M{"reviews": rRef}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, uUpdate, opts).Decode(&user)
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	rr.Review = &review
	rr.UserReviews = user.Reviews
	sr.Respond(w, 200, "success", rr)
}
