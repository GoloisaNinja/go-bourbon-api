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
	"time"
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

func (cs *ControlStuct) setControlStructUserRef(u *models.User, cId primitive.ObjectID, cType string) {
	if cType == "collection" {
		for _, collection := range u.Collections {
			if collection.CollectionID == cId {
				ucr, _ := json.Marshal(collection)
				cs.UserRef = ucr
			}
		}
	} else {
		for _, wishlist := range u.Wishlists {
			if wishlist.WishlistID == cId {
				uwr, _ := json.Marshal(wishlist)
				cs.UserRef = uwr
			}
		}
	}
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
	// filters, updates, and opts
	filter := bson.M{"_id": uId}
	json.Unmarshal(rBody, &cr)
	cr.FillDefaults()
	cm.Build(uId, uName, cr.Name, cr.Private)
	if cType == "collection" {
		uCRef.Build(cm.ID, cm.Name)
		uCRefMarshal, _ := json.Marshal(uCRef)
		result.UserRef = uCRefMarshal
	} else {
		uWRef.Build(cm.ID, cm.Name)
		uWRefMarshal, _ := json.Marshal(uWRef)
		result.UserRef = uWRefMarshal
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
	_, uErr := usersCollection.UpdateOne(context.TODO(), filter, update)
	if uErr != nil {
		definedError.Build(500, "error", uErr.Error())
		return result, definedError
	}
	cmM, _ := json.Marshal(cm)
	//umM, _ := json.Marshal(u)
	result.Element = cmM
	//result.UserRef = umM

	return result, definedError
}

// DeleteController is a reusable function between collections and wishlists for deleting
// both full collection or wishlist document as well as the user reference document
func DeleteController(cId, uId primitive.ObjectID, cType string) responses.ErrorResponse {
	var definedError responses.ErrorResponse
	collMap := map[string]*mongo.Collection{
		"collection": collectionsCollection,
		"wishlist":   wishlistsCollection,
	}
	collectionToUse := collMap[cType]
	if collectionToUse == nil {
		definedError.Build(400, "error", "bad request")
		return definedError
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
		return definedError
	}
	// we didn't find a collection with the param collection belonging to
	// the authorized user making the request
	if result.DeletedCount == 0 {
		definedError.Build(400, "error", "bad request")
		return definedError
	}
	// delete the collectionRef from the user document
	// and return the updated user object doc
	uFilter := bson.M{"_id": uId}
	_, uUpErr := usersCollection.UpdateOne(context.TODO(), uFilter, update)
	if uUpErr != nil {
		definedError.Build(401, "error", "unauthorized")
		return definedError
	}
	definedError.Build(0, "no errors", "delete success")
	return definedError
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
	// user filters and update bsons
	var uFilter bson.M
	var uUpdate bson.M
	json.Unmarshal(rBody, &cr)
	cr.FillDefaults()
	// set an update time for records
	updateTime := primitive.NewDateTimeFromTime(time.Now())
	typeFilterMap := map[string][]bson.M{
		"collection": {{"_id": uId, "collections.collection_id": cId}, {"$set": bson.M{"collections.$.collection_name": cr.Name, "updatedAt": updateTime}}},
		"wishlist":   {{"_id": uId, "wishlists.wishlist_id": cId}, {"$set": bson.M{"wishlists.$.wishlist_name": cr.Name, "updatedAt": updateTime}}},
	}
	// filters, updates, and opts
	cFilter := bson.M{"_id": cId}
	cUpdate := []bson.M{{"$set": bson.M{"name": cr.Name, "private": cr.Private, "updatedAt": updateTime}}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	uFilter = typeFilterMap[cType][0]
	uUpdate = typeFilterMap[cType][1]

	err := collectionToUse.FindOneAndUpdate(context.TODO(), cFilter, cUpdate, opts).Decode(&cm)
	if err != nil {
		definedError.Build(400, "error", err.Error())
		return result, definedError
	}
	var u models.User
	uErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, uUpdate, opts).Decode(&u)
	if uErr != nil {
		definedError.Build(400, "error", uErr.Error())
		return result, definedError
	}
	result.setControlStructUserRef(&u, cId, cType)
	cmM, _ := json.Marshal(cm)
	result.Element = cmM
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
	updateTime := primitive.NewDateTimeFromTime(time.Now())
	// collection filter and collection update
	filter := bson.M{"_id": cId, "user.id": uId}
	cUpdate := bson.M{operator: bson.M{"bourbons": b}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	// user filter and user ref updates
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
	// update the updatedAt
	tUpdate := bson.M{"$set": bson.M{"updatedAt": updateTime}}
	_, tUp := collectionToUse.UpdateOne(context.TODO(), filter, tUpdate)
	if tUp != nil {
		definedError.Build(500, "error", "update failed")
		return result, definedError
	}
	// determine the type of update needed based on action - default is adding bourbon
	cUpErr := collectionToUse.FindOneAndUpdate(context.TODO(), filter, cUpdate, opts).Decode(&cm)
	if cUpErr != nil {
		definedError.Build(400, "error", cUpErr.Error())
		return result, definedError
	}
	// update the user updatedAt
	_, utUp := usersCollection.UpdateOne(context.TODO(), uFilter, tUpdate)
	if utUp != nil {
		definedError.Build(500, "error", "update failed")
		return result, definedError
	}
	// find and update user based on collection type and action updates determined above
	var u models.User
	uErr := usersCollection.FindOneAndUpdate(context.TODO(), uFilter, uUpdate, opts).Decode(&u)
	if uErr != nil {
		definedError.Build(400, "error", uErr.Error())
		return result, definedError
	}
	result.setControlStructUserRef(&u, cId, cType)
	cMm, _ := json.Marshal(cm)
	result.Element = cMm
	return result, definedError
}
