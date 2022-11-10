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

func bourbonExistsInDb(id primitive.ObjectID) (models.Bourbon, error) {
	var b models.Bourbon
	bFilter := bson.M{"_id": id}
	err := bourbonsCollection.FindOne(context.TODO(), bFilter).Decode(&b)
	return b, err
}

func collectionExistsInDb(cid, uid primitive.ObjectID) (models.Collection, error) {
	var c models.Collection
	cFilter := bson.D{{"_id", cid}, {"user.id", uid}}
	err := collectionsCollection.FindOne(context.TODO(), cFilter).Decode(&c)
	return c, err
}

func updateBInC(b *models.Bourbon, cid, uid primitive.ObjectID, t string) (models.Collection, error) {
	var c models.Collection
	update := bson.M{"$pull": bson.M{"bourbons": bson.M{"_id": b.ID}}}
	if t == "add" {
		update = bson.M{"$push": bson.M{"bourbons": b}}
	}
	filter := bson.D{{"_id", cid}, {"user.id", uid}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := collectionsCollection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&c)
	return c, err
}

func updateBrefInUC(bid, cid, uid primitive.ObjectID, t string) (models.User, error) {
	var u models.User
	bRef := models.BourbonsRef{
		BourbonID: bid,
	}
	update := bson.M{"$pull": bson.M{"collections.$.bourbons": bRef}}
	if t == "add" {
		update = bson.M{"$push": bson.M{"collections.$.bourbons": bRef}}
	}
	filter := bson.M{"_id": uid, "collections.collection_id": cid}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := usersCollection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&u)
	return u, err
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
	bourbon, err := bourbonExistsInDb(bourbonId)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	// does the collection exist and belong to the user?
	collection, cErr := collectionExistsInDb(collectionId, userId)
	if cErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bad request or collection unauthorized")
		return
	}
	// is the bourbon already in the collection?
	if bourbonAlreadyInCollection(collection.Bourbons, bourbonId) {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bourbon already in collection")
		return
	}
	// we can now add the bourbon to the collection and to the user collection ref
	// add bourbon to collection model and return updated doc
	uc, cUpErr := updateBInC(&bourbon, collectionId, userId, "add")
	if cUpErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", cUpErr.Error())
		return
	}
	// add bourbon to user collection ref
	uu, uUpErr := updateBrefInUC(bourbonId, collectionId, userId, "add")
	if uUpErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uUpErr.Error())
		return
	}
	response := responses.CollectionResponse{
		Collection:      &uc,
		UserCollections: uu.Collections,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", response)
}

// DeleteBourbonFromCollection does the inverse of AddBourbonToCollection
// the conditional control flow is almost identical with the exception
// being that now we WANT the bourbon to exist in the collection so
// that we can delete it
func DeleteBourbonFromCollection(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["collectionId"])
	bourbonId, _ := primitive.ObjectIDFromHex(params["bourbonId"])
	// user id is in the context from auth middleware
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	//order of operations:
	//does the bourbon exist
	//does the collection exist and belong to the user?
	//does the bourbon already exist in the collection?
	//if yes/yes/yes -> then we can complete the route handler
	bourbon, err := bourbonExistsInDb(bourbonId)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	// does the collection exist and belong to the user?
	collection, cErr := collectionExistsInDb(collectionId, userId)
	if cErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bad request or collection unauthorized")
		return
	}
	// is the bourbon already in the collection?
	if !bourbonAlreadyInCollection(collection.Bourbons, bourbonId) {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", "bourbon not in collection")
		return
	}
	// we can now delete the bourbon from the collection and from the user collection ref
	// remove bourbon from collection model and return updated doc
	uc, cUpErr := updateBInC(&bourbon, collectionId, userId, "remove")
	if cUpErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", cUpErr.Error())
		return
	}
	// remove bourbon from user collection ref
	uu, uUpErr := updateBrefInUC(bourbonId, collectionId, userId, "remove")
	if uUpErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uUpErr.Error())
		return
	}
	response := responses.CollectionResponse{
		Collection:      &uc,
		UserCollections: uu.Collections,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", response)
}
