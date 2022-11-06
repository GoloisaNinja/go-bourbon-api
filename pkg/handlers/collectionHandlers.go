package handlers

import (
	"context"
	"encoding/json"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"net/http"
)

var collectionsCollection = db.GetCollection(
	db.DB,
	"collections",
)

func newCollectionConstructor(
	m map[string]string,
	i primitive.ObjectID,
	u *models.User,
) *models.Collection {
	var isPrivate bool
	var colName string
	if m["private"] != "" {
		switch m["private"] {
		case "true":
			isPrivate = true
		case "false":
			isPrivate = false
		default:
			isPrivate = true
		}
	} else {
		isPrivate = true
	}
	if m["name"] == "" {
		colName = "Unnamed Collection"
	} else {
		colName = m["name"]
	}
	return &models.Collection{
		ID: i,
		User: &models.UserRef{
			ID:       u.ID,
			Username: u.Username,
		},
		Name:     colName,
		Private:  isPrivate,
		Bourbons: make([]*models.Bourbon, 0),
	}
}

func CreateCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		responses.RespondWithError(
			w, http.StatusMethodNotAllowed, "method not allowed",
			"request method not allowed on this endpoint",
		)
		return
	}
	// auth middleware context - need user id to continue
	authFromCtx := r.Context().Value("authContext")
	auth := authFromCtx.(*models.AuthContext)
	id, _ := primitive.ObjectIDFromHex(auth.UserId)
	// get the user from the database
	filter := bson.M{"_id": id}
	var dbUser models.User
	err := usersCollection.FindOne(context.TODO(), filter).Decode(&dbUser)
	if err != nil {
		responses.RespondWithError(
			w, http.StatusInternalServerError,
			"error", err.Error(),
		)
	}
	// get the request body
	rBody, _ := ioutil.ReadAll(r.Body)
	var rMap map[string]string
	json.Unmarshal(rBody, &rMap)
	newColl := newCollectionConstructor(rMap, primitive.NewObjectID(), &dbUser)
	_, colErr := collectionsCollection.InsertOne(context.TODO(), newColl)
	if colErr != nil {
		responses.RespondWithError(
			w, http.StatusInternalServerError,
			"error", colErr.Error(),
		)
		return
	}
	newUserColl := models.UserCollectionRef{
		CollectionID:   newColl.ID,
		CollectionName: newColl.Name,
		Bourbons:       make([]*models.BourbonsRef, 0),
	}
	var updatedUser models.User
	update := bson.M{"$push": bson.M{"collections": newUserColl}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uErr := usersCollection.FindOneAndUpdate(
		context.TODO(), filter, update,
		opts,
	).Decode(&updatedUser)
	if uErr != nil {
		responses.RespondWithError(
			w, http.StatusInternalServerError,
			"error", uErr.Error(),
		)
		return
	}
	response := responses.CollectionResponse{
		Collection:      newColl,
		UserCollections: updatedUser.Collections,
	}
	json.NewEncoder(w).Encode(
		responses.StandardResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": response},
		},
	)
}
