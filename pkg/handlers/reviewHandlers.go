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
	"time"
)

var reviewsCollection = db.GetCollection(db.DB, "reviews")

// GetReviewById returns a single review based on the REVIEW ID passed in url params
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
	var rr responses.ReviewsResponse
	var sr responses.StandardResponse
	var reviews []*models.UserReview
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
		var review *models.UserReview
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
		rr.Reviews = reviews
		sr.Respond(w, 200, "success", rr)
	} else {
		er.Respond(w, 404, "error", "not found")
	}
}

func CreateReview(w http.ResponseWriter, r *http.Request) {
	var er responses.ErrorResponse
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
	_, uErr := usersCollection.UpdateOne(context.TODO(), uFilter, uUpdate)
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	rr.Review = &review
	rr.UserReview = &rRef
	sr.Respond(w, 200, "success", rr)
}

func DeleteReview(w http.ResponseWriter, r *http.Request) {
	var er responses.ErrorResponse
	var sr responses.StandardResponse
	params := mux.Vars(r)
	reviewId, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		er.Respond(w, 400, "error", err.Error())
		return
	}
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	userId := ctx.UserId
	filter := bson.M{"_id": reviewId, "user.id": userId}
	result, rErr := reviewsCollection.DeleteOne(context.TODO(), filter)
	if rErr != nil {
		er.Respond(w, 500, "error", rErr.Error())
		return
	}
	if result.DeletedCount == 0 {
		er.Respond(w, 404, "error", "no review with that id could be deleted")
		return
	}
	rFilter := bson.M{"_id": userId, "reviews.review_id": reviewId}
	rUpdate := bson.M{"$pull": bson.M{"reviews": bson.M{"review_id": reviewId}}}
	_, uErr := usersCollection.UpdateOne(context.TODO(), rFilter, rUpdate)
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	sr.Respond(w, 200, "success", "delete review was successful")
}

func UpdateReview(w http.ResponseWriter, r *http.Request) {
	var er responses.ErrorResponse
	var rReq models.ReviewRequest
	var review models.UserReview
	var rr responses.ReviewResponse
	var uRRef models.UserReviewRef
	var sr responses.StandardResponse
	params := mux.Vars(r)
	reviewId, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		er.Respond(w, 400, "error", err.Error())
		return
	}
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	userId := ctx.UserId
	rBody, iErr := ioutil.ReadAll(r.Body)
	if iErr != nil {
		er.Respond(w, 500, "error", iErr.Error())
		return
	}
	json.Unmarshal(rBody, &rReq)
	if rReq.ReviewScore == "" || rReq.ReviewText == "" || rReq.ReviewTitle == "" {
		er.Respond(w, 400, "error", "bad request")
		return
	}
	// user review ref construction for the response
	uRRef.ReviewID = reviewId
	uRRef.ReviewTitle = rReq.ReviewTitle
	// update time for db
	updatedTime := primitive.NewDateTimeFromTime(time.Now())
	filter := bson.M{"_id": reviewId, "user.id": userId}
	update := bson.M{"$set": bson.M{"reviewTitle": rReq.ReviewTitle, "reviewScore": rReq.ReviewScore, "reviewText": rReq.ReviewText, "updatedAt": updatedTime}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	rUpErr := reviewsCollection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&review)
	if rUpErr != nil {
		er.Respond(w, 500, "error", rUpErr.Error())
		return
	}
	uFilter := bson.M{"_id": userId, "reviews.review_id": reviewId}
	uRefUpdate := bson.M{"$set": bson.M{"reviews.$.review_title": rReq.ReviewTitle, "updatedAt": updatedTime}}
	_, uRefUpErr := usersCollection.UpdateOne(context.TODO(), uFilter, uRefUpdate)
	if uRefUpErr != nil {
		er.Respond(w, 500, "error", uRefUpErr.Error())
		return
	}
	rr.Review = &review
	rr.UserReview = &uRRef
	sr.Respond(w, 200, "success", rr)
}
