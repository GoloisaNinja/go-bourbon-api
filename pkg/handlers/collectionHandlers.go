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

type CollectionRequest struct {
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

func bourbonAlreadyInCollection(b []*models.Bourbon, id primitive.ObjectID) bool {
	result := false
	for _, bObj := range b {
		if bObj.ID == id {
			return true
		}
	}
	return result
}

// collectionReq_defaults is called within the CreateCollection handler
// to evaluate the incoming json that has been mapped allowing us to
// manage to some defaults if the incoming request is less than ideal
func (req *CollectionRequest) collectionReqDefaults() {
	if req.Private != true && req.Private != false {
		req.Private = true
	}
	if req.Name == "" {
		req.Name = "Unnamed Collection"
	}
}

// GetCollectionById returns a collection - if the collection is
// private then the user making the request must be the owner of
// the collection
func GetCollectionById(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	// auth middleware context - need user id to continue
	id, iErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if iErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", iErr.Error())
		return
	}
	var collection models.Collection
	filter := bson.M{"_id": collectionId}
	err := collectionsCollection.FindOne(context.TODO(), filter).Decode(&collection)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", err.Error())
		return
	}
	if collection.Private {
		if collection.User.ID != id {
			var er responses.ErrorResponse
			er.Respond(w, 401, "error", "unauthorized")
			return
		}
	}
	var cr responses.StandardResponse
	cr.Respond(w, 200, "success", collection)
}

// CreateCollection creates a new collection in the collections collection
// and also adds a UserCollectionRef to the User that created it
func CreateCollection(w http.ResponseWriter, r *http.Request) {
	// auth middleware context - need user id to continue
	id, iErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if iErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", iErr.Error())
		return
	}
	// get the user from the database
	filter := bson.M{"_id": id}
	var dbUser models.User
	err := usersCollection.FindOne(context.TODO(), filter).Decode(&dbUser)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	// get the request body
	rBody, _ := ioutil.ReadAll(r.Body)
	var cReq CollectionRequest
	json.Unmarshal(rBody, &cReq)
	cReq.collectionReqDefaults()
	var newColl models.Collection
	newColl.Build(dbUser.ID, dbUser.Username, cReq.Name, cReq.Private)
	_, colErr := collectionsCollection.InsertOne(context.TODO(), newColl)
	if colErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", colErr.Error())
		return
	}
	var newUserColl models.UserCollectionRef
	newUserColl.Build(newColl.ID, newColl.Name)
	var updatedUser models.User
	update := bson.M{"$push": bson.M{"collections": newUserColl}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uErr := usersCollection.FindOneAndUpdate(
		context.TODO(), filter, update,
		opts,
	).Decode(&updatedUser)
	if uErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	cr := responses.CollectionResponse{
		Collection:      &newColl,
		UserCollections: updatedUser.Collections,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", cr)
}

func UpdateCollection(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	// user id is in the context from auth middleware
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	rBody, _ := ioutil.ReadAll(r.Body)
	var cr CollectionRequest
	var c models.Collection
	json.Unmarshal(rBody, &cr)
	cr.collectionReqDefaults()
	cFilter := bson.M{"_id": collectionId}
	cUpdate := []bson.D{bson.D{{"$set", bson.D{{"name", cr.Name}, {"private", cr.Private}}}}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	cErr := collectionsCollection.FindOneAndUpdate(context.TODO(), cFilter, cUpdate, opts).Decode(&c)
	if cErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", cErr.Error())
		return
	}
	var u models.User
	uFilter := bson.M{"_id": userId, "collections.collection_id": collectionId}
	uUpdate := bson.M{"$set": bson.M{"collections.$.collection_name": cr.Name}}
	uOpts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uUpErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, uUpdate, uOpts).Decode(&u)
	if uUpErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", uUpErr.Error())
		return
	}
	response := responses.CollectionResponse{
		Collection:      &c,
		UserCollections: u.Collections,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", response)
}

// DeleteCollection deletes a collection from the collections
// collection and from the user collection reference - the collection
// must belong to the user that is requesting the deletion
func DeleteCollection(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	// user id is in the context from auth middleware
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	// deleting the collection document entirely
	cFilter := bson.M{"_id": collectionId}
	result, err := collectionsCollection.DeleteOne(context.TODO(), cFilter)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", err.Error())
		return
	}
	// we didn't find a collection with the param collection belonging to
	// the authorized user making the request
	if result.DeletedCount == 0 {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bad request")
		return
	}
	// delete the collectionRef from the user document
	// and return the updated user object doc
	var updatedUser models.User
	uFilter := bson.M{"_id": userId}
	update := bson.M{"$pull": bson.M{"collections": bson.M{"collection_id": collectionId}}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uUpErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, update, opts).Decode(&updatedUser)
	if uUpErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 401, "error", "unauthorized")
		return
	}
	var ur responses.StandardResponse
	ur.Respond(w, 200, "success", updatedUser.Collections)
}

// AddBourbonToCollection route relies on the auth middleware to
// get us the auth context, we then get the collection id from the
// router params and the bourbon id from the request body
// we check various conditional control flows before finally adding
// the request bourbon into the collection document and into the user
// collections reference document
func AddBourbonToCollection(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	// user id is in the context from auth middleware
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	// bourbon id is in the request body
	reqBody, _ := ioutil.ReadAll(r.Body)
	var rMap map[string]string
	json.Unmarshal(reqBody, &rMap)
	bourbonId, bErr := primitive.ObjectIDFromHex(rMap["bourbonId"])
	if bErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bad request")
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
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", bDbErr.Error())
		return
	}
	// does the collection exist and belong to the user?
	var collectionFromReq models.Collection
	cFilter := bson.D{{"_id", collectionId}, {"user.id", userId}}
	cDbErr := collectionsCollection.FindOne(context.TODO(), cFilter).Decode(&collectionFromReq)
	if cDbErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bad request or collection unauthorized")
		return
	}
	// is the bourbon already in the collection?
	if bourbonAlreadyInCollection(collectionFromReq.Bourbons, bourbonId) {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bourbon already in collection")
		return
	}
	// we can now add the bourbon to the collection and to the user collection ref
	// add bourbon to collection model and return updated doc
	var updatedCollection models.Collection
	update := bson.M{"$push": bson.M{"bourbons": bourbonFromReq}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	cUpErr := collectionsCollection.FindOneAndUpdate(context.TODO(), cFilter, update, opts).Decode(&updatedCollection)
	if cUpErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", cUpErr.Error())
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
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uUpErr.Error())
		return
	}
	response := responses.CollectionResponse{
		Collection:      &updatedCollection,
		UserCollections: updatedUser.Collections,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", response)
}
