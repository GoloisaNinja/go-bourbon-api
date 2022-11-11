package handlers

import (
	"context"
	"encoding/json"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ControlStuct struct {
	Element []byte
	UserRef []byte
}

func bourbonUpdateValid(b []*models.Bourbon, id primitive.ObjectID, uType string) bool {
	var result bool
	for _, bObj := range b {
		if uType == "add" {
			if bObj.ID == id {
				return false
			}
			result = true
		} else {
			if bObj.ID == id {
				return true
			}
			result = false
		}

	}
	return result
}

// ExistsAndUpdateController has a conditional control flow that determines what kind of collection
// is being asked to update - collection or wishlist
// query filters, updates, and options are constructed based on collection or wishlist
// each element - bourbon/collection/wishlist are checked to be real and if auth user
// is allowed to access - a failure at any point results in a return of an empty control struct
// and an error response where the status is no longer 0 (initial memory allocation)
// this function can be reused across collection model type and wishlist model type which are
// almost identical
func ExistsAndUpdateController(cId, bId, uId primitive.ObjectID, action, cType string) (ControlStuct, responses.ErrorResponse) {
	var b models.Bourbon
	var cm models.Collection
	var wm models.Wishlist
	var result ControlStuct
	var definedError responses.ErrorResponse
	bRef := models.BourbonsRef{
		BourbonID: bId,
	}
	var x bson.M // this interface will hold the collection element regardless of struct type
	bFilter := bson.M{"_id": bId}
	// check if the bourbon exists
	err := bourbonsCollection.FindOne(context.TODO(), bFilter).Decode(&b)
	if err != nil {
		definedError.Build(400, "error", err.Error())
		return result, definedError
	}
	var collectionToUse *mongo.Collection
	// general collection group update and filter queries
	cUpdate := bson.M{"$push": bson.M{"bourbons": b}}                   // update for collection and wishlist bourbons array
	filter := bson.D{{"_id", cId}, {"user.id", uId}}                    // general filter
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After) // general options
	var uUpdate bson.M
	var uFilter bson.M
	if action != "add" {
		cUpdate = bson.M{"$pull": bson.M{"bourbons": bson.M{"_id": b.ID}}}
	}
	if cType == "w" {
		collectionToUse = wishlistsCollection
	} else {
		collectionToUse = collectionsCollection
		uUpdate = bson.M{"$push": bson.M{"collections.$.bourbons": bRef}}
		uFilter = bson.M{"_id": uId, "collections.collection_id": cId}
		if action != "add" {
			uUpdate = bson.M{"$pull": bson.M{"collections.$.bourbons": bRef}}
		}
	}
	dErr := collectionToUse.FindOne(context.TODO(), filter).Decode(&x)
	if dErr != nil {
		definedError.Build(400, "error", err.Error())
		return result, definedError
	}
	// marshal/unmarshal result of query - determine collection type and use the
	// bourbonUpdateValid func to see if the user action can be performed -
	// does the bourbon exist in the collection document
	mX, _ := json.Marshal(x)
	if cType == "w" {
		json.Unmarshal(mX, &wm)
		if !bourbonUpdateValid(wm.Bourbons, b.ID, action) {
			definedError.Build(400, "error", "action not valid")
			return result, definedError
		}
	} else {
		json.Unmarshal(mX, &cm)
		if !bourbonUpdateValid(cm.Bourbons, b.ID, action) {
			definedError.Build(400, "error", "action not valid")
			return result, definedError
		}
	}
	// determine the type of update needed based on action - default is adding bourbon
	cUpErr := collectionToUse.FindOneAndUpdate(context.TODO(), filter, cUpdate, opts).Decode(&x)
	if cUpErr != nil {
		definedError.Build(400, "error", cUpErr.Error())
		return result, definedError
	}
	// find and update user based on collection type and action updates determined above
	var u models.User
	uErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, uUpdate, opts).Decode(&u)
	if uErr != nil {
		definedError.Build(400, "error", uErr.Error())
		return result, definedError
	}
	em, _ := json.Marshal(x)
	um, _ := json.Marshal(u)
	result.Element = em
	result.UserRef = um
	return result, definedError
}
