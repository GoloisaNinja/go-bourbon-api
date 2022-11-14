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
	if len(b) < 1 && uType == "add" {
		return true
	} else if len(b) < 1 && uType != "add" {
		return false
	}
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

func CreateController(rBody []byte, uId primitive.ObjectID, uName string, cType string) (ControlStuct, responses.ErrorResponse) {
	var result ControlStuct
	var definedError responses.ErrorResponse
	collMap := map[string]*mongo.Collection{
		"collection": collectionsCollection,
		"wishlist":   wishlistsCollection,
	}
	collectionToUse := collMap[cType]
	if collectionToUse == nil {
		definedError.Build(400, "error", "bad request")
		return result, definedError
	}
	// collection/wishlist overlapping models
	var cr models.CollectionRequest
	var cm models.Collection
	var uCRef models.UserCollectionRef
	// wishlist unique/specific userRef
	var uWRef models.UserWishlistRef
	// User
	var u models.User
	// filters, updates, and opts
	filter := bson.M{"_id": uId}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	json.Unmarshal(rBody, &cr)
	cr.FillDefaults()
	cm.Build(uId, uName, cr.Name, cr.Private)
	if cType == "collection" {
		uCRef.Build(cm.ID, cm.Name)
	} else {
		uWRef.Build(cm.ID, cm.Name)
	}
	typeQueryMap := map[string]bson.M{
		"collection": {"$push": bson.M{"collections": uCRef}},
		"wishlist":   {"$push": bson.M{"wishlists": uWRef}},
	}
	update := typeQueryMap[cType]
	_, err := collectionToUse.InsertOne(context.TODO(), cm)
	if err != nil {
		definedError.Build(500, "error", err.Error())
		return result, definedError
	}
	uErr := usersCollection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&u)
	if uErr != nil {
		definedError.Build(500, "error", uErr.Error())
		return result, definedError
	}
	cmM, _ := json.Marshal(cm)
	umM, _ := json.Marshal(u)
	result.Element = cmM
	result.UserRef = umM
	return result, definedError
}

// DeleteController is a reusable function between collections and wishlists for deleting
// both full collection or wishlist document as well as the user reference document
func DeleteController(cId, uId primitive.ObjectID, cType string) (models.User, responses.ErrorResponse) {
	var u models.User
	var definedError responses.ErrorResponse
	collMap := map[string]*mongo.Collection{
		"collection": collectionsCollection,
		"wishlist":   wishlistsCollection,
	}
	collectionToUse := collMap[cType]
	if collectionToUse == nil {
		definedError.Build(400, "error", "bad request")
		return u, definedError
	}
	typeQueryMap := map[string]bson.M{
		"collection": {"$pull": bson.M{"collections": bson.M{"collection_id": cId}}},
		"wishlist":   {"$pull": bson.M{"wishlists": bson.M{"wishlist_id": cId}}},
	}

	update := typeQueryMap[cType]
	filter := bson.M{"_id": cId, "user.id": uId}
	result, err := collectionToUse.DeleteOne(context.TODO(), filter)
	if err != nil {
		definedError.Build(400, "error", err.Error())
		return u, definedError
	}
	// we didn't find a collection with the param collection belonging to
	// the authorized user making the request
	if result.DeletedCount == 0 {
		definedError.Build(400, "error", "bad request")
		return u, definedError
	}
	// delete the collectionRef from the user document
	// and return the updated user object doc
	uFilter := bson.M{"_id": uId}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uUpErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, update, opts).Decode(&u)
	if uUpErr != nil {
		definedError.Build(401, "error", "unauthorized")
		return u, definedError
	}
	return u, definedError
}

func UpdateController(rBody []byte, cId, uId primitive.ObjectID, cType string) (ControlStuct, responses.ErrorResponse) {
	var result ControlStuct
	var definedError responses.ErrorResponse
	collMap := map[string]*mongo.Collection{
		"collection": collectionsCollection,
		"wishlist":   wishlistsCollection,
	}
	collectionToUse := collMap[cType]
	if collectionToUse == nil {
		definedError.Build(400, "error", "bad request")
		return result, definedError
	}
	// collection models
	var cr models.CollectionRequest
	var cm models.Collection
	// User
	var u models.User
	var uFilter bson.M
	var uUpdate bson.M
	json.Unmarshal(rBody, &cr)
	cr.FillDefaults()
	typeFilterMap := map[string][]bson.M{
		"collection": {{"_id": uId, "collections.collection_id": cId}, {"$set": bson.M{"collections.$.collection_name": cr.Name}}},
		"wishlist":   {{"_id": uId, "wishlists.wishlist_id": cId}, {"$set": bson.M{"wishlists.$.wishlist_name": cr.Name}}},
	}
	// filters, updates, and opts
	cFilter := bson.M{"_id": cId}
	cUpdate := []bson.D{{{"$set", bson.D{{"name", cr.Name}, {"private", cr.Private}}}}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uFilter = typeFilterMap[cType][0]
	uUpdate = typeFilterMap[cType][1]

	err := collectionToUse.FindOneAndUpdate(context.TODO(), cFilter, cUpdate, opts).Decode(&cm)
	if err != nil {
		definedError.Build(400, "error", err.Error())
		return result, definedError
	}
	uErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, uUpdate, opts).Decode(&u)
	if uErr != nil {
		definedError.Build(400, "error", uErr.Error())
		return result, definedError
	}
	cmM, _ := json.Marshal(cm)
	umM, _ := json.Marshal(u)
	result.Element = cmM
	result.UserRef = umM
	return result, definedError
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
	var result ControlStuct
	var definedError responses.ErrorResponse
	bRef := models.BourbonsRef{
		BourbonID: bId,
	}
	bFilter := bson.M{"_id": bId}
	// check if the bourbon exists
	err := bourbonsCollection.FindOne(context.TODO(), bFilter).Decode(&b)
	if err != nil {
		definedError.Build(400, "error", err.Error())
		return result, definedError
	}
	collMap := map[string]*mongo.Collection{
		"collection": collectionsCollection,
		"wishlist":   wishlistsCollection,
	}
	collectionToUse := collMap[cType]
	if collectionToUse == nil {
		definedError.Build(400, "error", "bad request")
		return result, definedError
	}
	// general collection group update and filter queries
	operator := "$push"
	if action != "add" {
		operator = "$pull"
	}
	cUpdate := bson.M{operator: bson.M{"bourbons": b}}                  // update for collection and wishlist bourbons array
	filter := bson.M{"_id": cId, "user.id": uId}                        // general filter
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After) // general options
	typeQueryMap := map[string][]bson.M{
		"collection": {{"_id": uId, "collections.collection_id": cId}, {operator: bson.M{"collections.$.bourbons": bRef}}},
		"wishlist":   {{"_id": uId, "wishlists.wishlist_id": cId}, {operator: bson.M{"wishlists.$.bourbons": bRef}}},
	}
	uFilter := typeQueryMap[cType][0]
	uUpdate := typeQueryMap[cType][1]
	dErr := collectionToUse.FindOne(context.TODO(), filter).Decode(&cm)
	if dErr != nil {
		definedError.Build(400, "error", dErr.Error())
		return result, definedError
	}
	if !bourbonUpdateValid(cm.Bourbons, b.ID, action) {
		definedError.Build(400, "error", "action not valid")
		return result, definedError
	}

	// determine the type of update needed based on action - default is adding bourbon
	cUpErr := collectionToUse.FindOneAndUpdate(context.TODO(), filter, cUpdate, opts).Decode(&cm)
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
	cMm, _ := json.Marshal(cm)
	uMm, _ := json.Marshal(u)
	result.Element = cMm
	result.UserRef = uMm
	return result, definedError
}
