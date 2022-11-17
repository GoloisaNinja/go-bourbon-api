package main

import (
	appHandlers "github.com/GoloisaNinja/go-bourbon-api/pkg/handlers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/middleware"
	"github.com/gorilla/mux"
	"net/http"
)

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		},
	)
}

func routes() http.Handler {
	//Init Router
	r := mux.NewRouter()
	r.Use(commonMiddleware)
	// handler functions for routes
	// bourbon appHandlers
	getBourbons := http.HandlerFunc(appHandlers.GetBourbons)
	getRandomBourbon := http.HandlerFunc(appHandlers.GetRandomBourbon)
	getBourbonById := http.HandlerFunc(appHandlers.GetBourbonById)
	// user appHandlers.
	createNewUser := http.HandlerFunc(appHandlers.CreateUser)
	loginUser := http.HandlerFunc(appHandlers.LoginUser)
	logoutUserHandler := http.HandlerFunc(appHandlers.LogoutUser)

	// base database collection type appHandlers. for collections and wishlists
	// appHandlers. manage both database collection document types by extracting a cType from router params
	getCollectionTypeById := http.HandlerFunc(appHandlers.GetCollectionTypeById)
	getAllCollectionsType := http.HandlerFunc(appHandlers.GetCollectionsType)
	createCollection := http.HandlerFunc(appHandlers.CreateCollection)
	updateCollection := http.HandlerFunc(appHandlers.UpdateCollection)
	deleteCollection := http.HandlerFunc(appHandlers.DeleteCollection)
	updateBourbonsToCollection := http.HandlerFunc(appHandlers.UpdateBourbonsInCollection)

	// review appHandlers.
	getReviewById := http.HandlerFunc(appHandlers.GetReviewById)
	getAllReviewsByFilterId := http.HandlerFunc(appHandlers.GetAllReviewsByFilterId)
	createReview := http.HandlerFunc(appHandlers.CreateReview)
	deleteReview := http.HandlerFunc(appHandlers.DeleteReview)
	updateReview := http.HandlerFunc(appHandlers.UpdateReview)

	// define routes

	// **bourbon routes**
	// get paginated bourbons
	r.Handle("/api/bourbons", getBourbons).Methods("GET")
	// get a randomized bourbon
	r.Handle(
		"/api/bourbons/random", getRandomBourbon,
	).Methods("GET")
	// get a bourbon by id
	r.Handle("/api/bourbons/{id}", getBourbonById).Methods("GET")

	// **user routes**
	// create a new user
	r.Handle("/api/user", middleware.NewUserMiddleware(createNewUser)).Methods("POST")
	// login an existing user
	r.Handle("/api/user/login", loginUser).Methods("POST")
	// logout a user
	r.Handle("/api/user/logout", middleware.AuthMiddleware(logoutUserHandler)).Methods("POST")

	// review routes
	// create a review
	r.Handle("/api/review", middleware.AuthMiddleware(createReview)).Methods("POST")
	// get a single review by id
	r.Handle("/api/review/{id}", getReviewById).Methods("GET")
	// get all reviews by a filter type (either by bourbon id or by user id)
	r.Handle("/api/reviews/{fType}/{id}", getAllReviewsByFilterId).Methods("GET")
	// delete a review by id - auth route - user requesting delete must be owner of review
	r.Handle("/api/review/delete/{id}", middleware.AuthMiddleware(deleteReview)).Methods("DELETE")
	// update a single review
	r.Handle("/api/review/update/{id}", middleware.AuthMiddleware(updateReview)).Methods("POST")

	// **database collections routes (collection & wishlist cTypes)**
	// create a new collection or wishlist based on cType param
	r.Handle(
		"/api/type/{cType}", middleware.AuthMiddleware(createCollection),
	).Methods("POST")
	// get a collection or wishlist collection by id based on cType param
	r.Handle("/api/type/{cType}/{id}", middleware.AuthMiddleware(getCollectionTypeById)).Methods("GET")
	// get a slice of collections or wishlists based on the cType param the auth user making the request
	r.Handle("/api/type/{cType}", middleware.AuthMiddleware(getAllCollectionsType)).Methods("GET")
	// delete an existing collection or wishlist based on cType param
	r.Handle(
		"/api/type/{cType}/{id}", middleware.AuthMiddleware(deleteCollection),
	).Methods("DELETE")
	// update an existing collection or wishlist name and private flag based on cType param
	r.Handle("/api/type/{cType}/update/{id}", middleware.AuthMiddleware(updateCollection)).Methods("POST")
	// add or delete a bourbon by id into a collection and a usercollectionref
	// add or delete determined by action placeholder in route as well as cType router param
	r.Handle(
		"/api/type/{cType}/{action}/{collectionId}/{bourbonId}", middleware.AuthMiddleware(updateBourbonsToCollection),
	).Methods("POST", "DELETE")

	return r
}
