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
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"net/http"
)

var collectionsCollection = db.GetCollection(
	db.DB,
	"collections",
)

func bourbonAlreadyInCollection(b []*models.Bourbon, id primitive.ObjectID) bool {
	result := false
	for _, bObj := range b {
		if bObj.ID == id {
			return true
		}
	}
	return result
}

// newCollectionConstructor is called within the CreateCollection handler
// to evaluate the incoming json that has been mapped allowing us to
// manage to some defaults if the incoming request is less than ideal
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

// CreateCollection creates a new collection in the collections collection
// and also adds a UserCollectionRef to the User that created it
func CreateCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		responses.RespondWithError(
			w, http.StatusMethodNotAllowed, "method not allowed",
			"request method not allowed on this endpoint",
		)
		return
	}
	// auth middleware context - need user id to continue
	id, iErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if iErr != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, "error", iErr.Error())
		return
	}
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

// DeleteCollection deletes a collection from the collections
// collection and from the user collection reference - the collection
// must belong to the user that is requesting the deletion
func DeleteCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		responses.RespondWithError(
			w, http.StatusMethodNotAllowed, "method not allowed",
			"request method not allowed on this endpoint",
		)
		return
	}
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	// user id is in the context from auth middleware
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, "error", uErr.Error())
		return
	}
	// deleting the collection document entirely
	cFilter := bson.M{"_id": collectionId}
	result, err := collectionsCollection.DeleteOne(context.TODO(), cFilter)
	if err != nil {
		responses.RespondWithError(w, http.StatusBadRequest, "error", err.Error())
		return
	}
	// we didn't find a collection with the param collection belonging to
	// the authorized user making the request
	if result.DeletedCount == 0 {
		responses.RespondWithError(w, http.StatusBadRequest, "error", "bad request")
		return
	}
	// delete the collectionRef from the user document
	// and return the updated user object doc
	var updatedUser models.User
	uFilter := bson.M{"_id": userId, "collections.collection_id": collectionId}
	update := bson.M{"$pull": bson.D{{"collections", bson.D{{"collection_id", collectionId}}}}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uUpErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, update, opts).Decode(&updatedUser)
	if uUpErr != nil {
		responses.RespondWithError(w, http.StatusUnauthorized, "error", "unauthorized")
		return
	}
	json.NewEncoder(w).Encode(responses.StandardResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": updatedUser.Collections},
	})
}

// AddBourbonToCollection route relies on the auth middleware to
// get us the auth context, we then get the collection id from the
// router params and the bourbon id from the request body
// we check various conditional control flows before finally adding
// the request bourbon into the collection document and into the user
// collections reference document
func AddBourbonToCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		responses.RespondWithError(
			w, http.StatusMethodNotAllowed, "method not allowed",
			"request method not allowed on this endpoint",
		)
		return
	}
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	// user id is in the context from auth middleware
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, "error", uErr.Error())
		return
	}
	// bourbon id is in the request body
	reqBody, _ := ioutil.ReadAll(r.Body)
	var rMap map[string]string
	json.Unmarshal(reqBody, &rMap)
	bourbonId, bErr := primitive.ObjectIDFromHex(rMap["bourbonId"])
	if bErr != nil {
		responses.RespondWithError(w, http.StatusBadRequest, "error", "bad request")
		return
	}
	// order of operations:
	// does the bourbon exist
	// does the collection exist and belong to the user?
	// does the bourbon already exist in the collection?
	// if yes/yes/no -> then we can complete the route handler

	// does the bourbon exist?
	var bourbonFromReq models.Bourbon
	bFilter := bson.M{"_id": bourbonId}
	bDbErr := bourbonsCollection.FindOne(context.TODO(), bFilter).Decode(&bourbonFromReq)
	if bDbErr != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, "error", bDbErr.Error())
		return
	}
	// does the collection exist and belong to the user?
	var collectionFromReq models.Collection
	cFilter := bson.D{{"_id", collectionId}, {"user.id", userId}}
	cDbErr := collectionsCollection.FindOne(context.TODO(), cFilter).Decode(&collectionFromReq)
	if cDbErr != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, "error", "no collection using req params and user prim id")
		return
	}
	// is the bourbon already in the collection?
	if bourbonAlreadyInCollection(collectionFromReq.Bourbons, bourbonId) {
		responses.RespondWithError(w, http.StatusBadRequest, "error", "bourbon is already in this collection")
		return
	}
	// we can now add the bourbon to the collection and to the user collection ref
	// add bourbon to collection model and return updated doc
	var updatedCollection models.Collection
	update := bson.M{"$push": bson.M{"bourbons": bourbonFromReq}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	cUpErr := collectionsCollection.FindOneAndUpdate(context.TODO(), cFilter, update, opts).Decode(&updatedCollection)
	if cUpErr != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, "error", "failure at collection update...")
		return
	}
	// add bourbon to user collection ref
	var updatedUser models.User
	bourbonRef := models.BourbonsRef{
		BourbonID: bourbonId,
	}
	userCollRefFilter := bson.M{"_id": userId, "collections.collection_id": collectionId}
	userCollRefUpdate := bson.M{"$push": bson.M{"collections.$.bourbons": bourbonRef}}
	userOpts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uUpErr := usersCollection.FindOneAndUpdate(context.TODO(), userCollRefFilter, userCollRefUpdate, userOpts).Decode(&updatedUser)
	if uUpErr != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, "error", uUpErr.Error())
		return
	}
	response := responses.CollectionResponse{
		Collection:      &updatedCollection,
		UserCollections: updatedUser.Collections,
	}

	json.NewEncoder(w).Encode(responses.StandardResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": response},
	})
}
