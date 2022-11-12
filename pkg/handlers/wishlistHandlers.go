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

var wishlistsCollection = db.GetCollection(db.DB, "wishlists")

// GetWishlistById returns a wishlist - if the wishlist is
// private then the user making the request must be the owner of
// the wishlist
func GetWishlistById(w http.ResponseWriter, r *http.Request) {
	// params id contains collection id
	params := mux.Vars(r)
	wId, _ := primitive.ObjectIDFromHex(params["id"])
	// auth middleware context - need user id to continue
	id, iErr := helpers.GetUserIdFromAuthCtx(r.Context())
	if iErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", iErr.Error())
		return
	}
	var wishlist models.Collection
	filter := bson.M{"_id": wId}
	err := wishlistsCollection.FindOne(context.TODO(), filter).Decode(&wishlist)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", err.Error())
		return
	}
	if wishlist.Private {
		if wishlist.User.ID != id {
			var er responses.ErrorResponse
			er.Respond(w, 401, "error", "unauthorized")
			return
		}
	}
	var wr responses.StandardResponse
	wr.Respond(w, 200, "success", wishlist)
}

// CreateWishlist creates a new wishlist (identical to collection struct)
// in the wishlist database collection
// and also adds a UserWishlistRef to the User that created it
func CreateWishlist(w http.ResponseWriter, r *http.Request) {
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
	controlStruct, err := CreateController(rBody, user.ID, user.Username, "w")
	if err.Status != 0 {
		err.Respond(w, err.Status, err.Message, err.Data)
		return
	}
	var wm models.Collection
	var um models.User
	json.Unmarshal(controlStruct.Element, &wm)
	json.Unmarshal(controlStruct.UserRef, &um)
	wr := responses.WishlistResponse{
		Wishlist:      &wm,
		UserWishlists: um.Wishlists,
	}
	var sr responses.StandardResponse
	sr.Respond(w, 200, "success", wr)
}
