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
	"go.mongodb.org/mongo-driver/mongo"
	"io/ioutil"
	"net/http"
)

var collectionsCollection = db.GetCollection(
	db.DB,
	"collections",
)
var wishlistsCollection = db.GetCollection(db.DB, "wishlists")

// GetCollectionTypeById returns a collection/wishlist - if the collection is
// private then the user making the request must be the owner of
// the collection - collection to be used is dependent on cType from router params
func GetCollectionTypeById(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	cType, _ := params["cType"]
	// request cType map
	rMap := map[string]*mongo.Collection{
		"collection": collectionsCollection,
		"wishlist":   wishlistsCollection,
	}
	var collectionToUse *mongo.Collection
	var er responses.ErrorResponse
	collectionToUse = rMap[cType]
	if collectionToUse == nil {
		er.Respond(w, 404, "error", "not found")
		return
	}
	id, _ := primitive.ObjectIDFromHex(params["id"])
	// auth middleware context - need user id to continue
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	uId := ctx.UserId
	var cm models.Collection
	var sr responses.StandardResponse
	filter := bson.M{"_id": id}
	err := collectionToUse.FindOne(context.TODO(), filter).Decode(&cm)
	if err != nil {
		er.Respond(w, 400, "error", err.Error())
		return
	}
	if cm.Private && cm.User.ID != uId {
		er.Respond(w, 401, "error", "unauthorized")
		return
	}
	sr.Respond(w, 200, "success", cm)
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
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	id := ctx.UserId
	username := ctx.Username
	// get the request body
	rBody, _ := ioutil.ReadAll(r.Body)
	controlStruct, err := CreateController(rBody, id, username, cType)
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
	} else {
		wr.Wishlist = &cm
		wr.UserWishlists = um.Wishlists
		sr.Respond(w, 200, "success", wr)
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
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	userId := ctx.UserId
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
	} else {
		wr.Wishlist = &cm
		wr.UserWishlists = um.Wishlists
		sr.Respond(w, 200, "success", wr)
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
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	userId := ctx.UserId
	user, err := DeleteController(collectionId, userId, cType)
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var sr responses.StandardResponse
	if cType == "collection" {
		sr.Respond(w, 200, "success", user.Collections)
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
	ctx := r.Context().Value("authContext").(*models.AuthContext)
	userId := ctx.UserId
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
	} else {
		wr.Wishlist = &cm
		wr.UserWishlists = um.Wishlists
		sr.Respond(w, 200, "success", wr)
	}
}
