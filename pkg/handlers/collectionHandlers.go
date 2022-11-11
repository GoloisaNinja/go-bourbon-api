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

var collectionsCollection = db.GetCollection(
	db.DB,
	"collections",
)

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
	var user models.User
	uErr := usersCollection.FindOne(context.TODO(), filter).Decode(&user)
	if uErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
	// get the request body
	rBody, _ := ioutil.ReadAll(r.Body)
	controlStruct, err := CreateController(rBody, user.ID, user.Username, "c")
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var cm models.Collection
	var um models.User
	json.Unmarshal(controlStruct.Element, &cm)
	json.Unmarshal(controlStruct.UserRef, &um)
	cr := responses.CollectionResponse{
		Collection:      &cm,
		UserCollections: um.Collections,
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
	controlStuct, err := UpdateController(rBody, userId, collectionId, "c")
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var cm models.Collection
	var um models.User
	json.Unmarshal(controlStuct.Element, &cm)
	json.Unmarshal(controlStuct.UserRef, &um)
	response := responses.CollectionResponse{
		Collection:      &cm,
		UserCollections: um.Collections,
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
	user, err := DeleteController(collectionId, userId, "c")
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var ur responses.StandardResponse
	ur.Respond(w, 200, "success", user.Collections)
}

// AddBourbonToCollection route relies on the auth middleware to
// get us the auth context, we then get the collection id from the
// router params and the bourbon id from the request body
// we check various conditional control flows before finally adding
// the request bourbon into the collection document and into the user
// collections reference document
func AddBourbonToCollection(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["id"])
	userId, uErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if uErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", uErr.Error())
		return
	}
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
	controlStruct, err := ExistsAndUpdateController(collectionId, bourbonId, userId, "add", "c")
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var c models.Collection
	var u models.User
	json.Unmarshal(controlStruct.Element, &c)
	json.Unmarshal(controlStruct.UserRef, &u)

	response := responses.CollectionResponse{
		Collection:      &c,
		UserCollections: u.Collections,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", response)
}

// DeleteBourbonFromCollection does the inverse of AddBourbonToCollection
// the conditional control flow is almost identical with the exception
// being that now we WANT the bourbon to exist in the collection so
// that we can delete it
func DeleteBourbonFromCollection(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	collectionId, _ := primitive.ObjectIDFromHex(params["collectionId"])
	bourbonId, _ := primitive.ObjectIDFromHex(params["bourbonId"])
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
	controlStruct, err := ExistsAndUpdateController(collectionId, bourbonId, userId, "remove", "c")
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var c models.Collection
	var u models.User
	json.Unmarshal(controlStruct.Element, &c)
	json.Unmarshal(controlStruct.UserRef, &u)

	response := responses.CollectionResponse{
		Collection:      &c,
		UserCollections: u.Collections,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", response)
}
