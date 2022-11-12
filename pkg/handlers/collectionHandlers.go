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
	"go.mongodb.org/mongo-driver/mongo"
	"io/ioutil"
	"net/http"
)

var collectionsCollection = db.GetCollection(
	db.DB,
	"collections",
)
var wishlistsCollection = db.GetCollection(db.DB, "wishlists")

// GetCollectionById returns a collection - if the collection is
// private then the user making the request must be the owner of
// the collection
func GetCollectionById(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	cType, _ := params["cType"]
	var er responses.ErrorResponse
	if cType != "collection" && cType != "wishlist" {
		er.Respond(w, 404, "error", "not found")
		return
	}
	// auth middleware context - need user id to continue
	id, iErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if iErr != nil {
		er.Respond(w, 500, "error", iErr.Error())
		return
	}
	var collection models.Collection
	var collectionToUse *mongo.Collection
	if cType == "collection" {
		collectionToUse = collectionsCollection
	} else {
		collectionToUse = wishlistsCollection
	}
	filter := bson.M{"_id": collectionId}
	err := collectionToUse.FindOne(context.TODO(), filter).Decode(&collection)
	if err != nil {
		er.Respond(w, 400, "error", err.Error())
		return
	}
	if collection.Private {
		if collection.User.ID != id {
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
	params := mux.Vars(r)
	cType := params["cType"]
	var er responses.ErrorResponse
	if cType != "collection" && cType != "wishlist" {
		er.Respond(w, 404, "error", "not found")
		return
	}
	// auth middleware context - need user id to continue
	id, iErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if iErr != nil {
		er.Respond(w, 500, "error", iErr.Error())
		return
	}
	// get the user from the database
	filter := bson.M{"_id": id}
	var user models.User
	uErr := usersCollection.FindOne(context.TODO(), filter).Decode(&user)
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	// get the request body
	rBody, _ := ioutil.ReadAll(r.Body)
	controlStruct, err := CreateController(rBody, user.ID, user.Username, cType)
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var cm models.Collection
	var um models.User
	var cr responses.CollectionResponse
	var wr responses.WishlistResponse
	var sr responses.StandardResponse
	json.Unmarshal(controlStruct.Element, &cm)
	json.Unmarshal(controlStruct.UserRef, &um)
	if cType == "collection" {
		cr.Collection = &cm
		cr.UserCollections = um.Collections
		sr.Respond(w, 200, "success", cr)
		return
	} else {
		wr.Wishlist = &cm
		wr.UserWishlists = um.Wishlists
		sr.Respond(w, 200, "success", wr)
		return
	}
}

func UpdateCollection(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	cType := params["cType"]
	var er responses.ErrorResponse
	if cType != "collection" && cType != "wishlist" {
		er.Respond(w, 404, "error", "not found")
		return
	}
	// user id is in the context from auth middleware
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	rBody, _ := ioutil.ReadAll(r.Body)
	controlStuct, err := UpdateController(rBody, collectionId, userId, cType)
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var cm models.Collection
	var um models.User
	var cr responses.CollectionResponse
	var wr responses.WishlistResponse
	var sr responses.StandardResponse
	json.Unmarshal(controlStuct.Element, &cm)
	json.Unmarshal(controlStuct.UserRef, &um)
	if cType == "collection" {
		cr.Collection = &cm
		cr.UserCollections = um.Collections
		sr.Respond(w, 200, "success", cr)
		return
	} else {
		wr.Wishlist = &cm
		wr.UserWishlists = um.Wishlists
		sr.Respond(w, 200, "success", wr)
		return
	}
}

// DeleteCollection deletes a collection or wishlist from the respective database
// collection and from the associated user reference - the db collection type
// must belong to the user that is requesting the deletion
func DeleteCollection(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	// user id is in the context from auth middleware
	cType := params["cType"]
	var er responses.ErrorResponse
	if cType != "collection" && cType != "wishlist" {
		er.Respond(w, 404, "error", "not found")
		return
	}
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	user, err := DeleteController(collectionId, userId, cType)
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var sr responses.StandardResponse
	if cType == "collection" {
		sr.Respond(w, 200, "success", user.Collections)
		return
	} else {
		sr.Respond(w, 200, "success", user.Wishlists)
	}
}

// UpdateBourbonsInCollection route relies on the auth middleware to
// get us the auth context, we then get the database collection id, bourbon id,
// and the action to update (add/delete) from the params as well as the cType
// note that the database collection id could be a collection or wishlist

func UpdateBourbonsInCollection(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var er responses.ErrorResponse
	collectionId, _ := primitive.ObjectIDFromHex(params["collectionId"])
	bourbonId, _ := primitive.ObjectIDFromHex(params["bourbonId"])
	cType := params["cType"]
	if cType != "collection" && cType != "wishlist" {
		er.Respond(w, 404, "error", "not found")
		return
	}
	action := params["action"]
	if action != "add" && action != "delete" {
		er.Respond(w, 404, "error", "not found")
		return
	}
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	// ExistsAndUpdateController order of operations:
	// does the bourbon exist -> does the collection exist and belong to the user
	// does the bourbon already exist in the collection -> if yes/yes/no -> success

	controlStruct, err := ExistsAndUpdateController(collectionId, bourbonId, userId, action, cType)
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var cm models.Collection
	var um models.User
	json.Unmarshal(controlStruct.Element, &cm)
	json.Unmarshal(controlStruct.UserRef, &um)
	var cr responses.CollectionResponse
	var wr responses.WishlistResponse
	var sr responses.StandardResponse
	if cType == "collection" {
		cr.Collection = &cm
		cr.UserCollections = um.Collections
		sr.Respond(w, 200, "success", cr)
		return
	} else {
		wr.Wishlist = &cm
		wr.UserWishlists = um.Wishlists
		sr.Respond(w, 200, "success", wr)
		return
	}
}
