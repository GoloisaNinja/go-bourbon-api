package main

import (
	"github.com/GoloisaNinja/go-bourbon-api/pkg/handlers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/middleware"
	"github.com/gorilla/mux"
	"net/http"
)

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			next.ServeHTTP(w, r)
		},
	)
}

func routes() http.Handler {
	//Init Router
	r := mux.NewRouter()
	// basic middleware
	r.Use(commonMiddleware)

	// handler functions for routes
	// user handlers
	createNewUser := http.HandlerFunc(handlers.CreateUser)
	logoutUserHandler := http.HandlerFunc(handlers.LogoutUser)

	// base database collection type handlers for collections and wishlists
	// handlers manage both database collection document types by extracting a cType from router params
	getCollectionTypeById := http.HandlerFunc(handlers.GetCollectionTypeById)
	createCollection := http.HandlerFunc(handlers.CreateCollection)
	updateCollection := http.HandlerFunc(handlers.UpdateCollection)
	deleteCollection := http.HandlerFunc(handlers.DeleteCollection)
	updateBourbonsToCollection := http.HandlerFunc(handlers.UpdateBourbonsInCollection)

	// review handlers
	createReview := http.HandlerFunc(handlers.CreateReview)
	deleteReview := http.HandlerFunc(handlers.DeleteReview)
	updateReview := http.HandlerFunc(handlers.UpdateReview)

	// define routes

	// **bourbon routes**
	// get paginated bourbons
	r.HandleFunc("/api/bourbons", handlers.GetBourbons).Methods("GET")
	// get a randomized bourbon
	r.HandleFunc(
		"/api/bourbons/random", handlers.GetRandomBourbon,
	).Methods("GET")
	// get a bourbon by id
	r.HandleFunc("/api/bourbons/{id}", handlers.GetBourbonById).Methods("GET")

	// **user routes**
	// create a new user
	r.Handle("/api/user", middleware.NewUserMiddleware(createNewUser)).Methods("POST")
	// login an existing user
	r.HandleFunc("/api/user/login", handlers.LoginUser).Methods("GET")
	// logout a user
	r.Handle("/api/user/logout", middleware.AuthMiddleware(logoutUserHandler)).Methods("POST")

	// review routes
	// create a review
	r.Handle("/api/review", middleware.AuthMiddleware(createReview)).Methods("POST")
	// get a single review by id
	r.HandleFunc("/api/review/{id}", handlers.GetReviewById).Methods("GET")
	// get all reviews by a filter type (either by bourbon id or by user id)
	r.HandleFunc("/api/reviews/{fType}/{id}", handlers.GetAllReviewsByFilterId).Methods("GET")
	// delete a review by id - auth route - user requesting delete must be owner of review
	r.Handle("/api/review/delete/{id}", middleware.AuthMiddleware(deleteReview)).Methods("DELETE")
	// update a single review
	r.Handle("/api/review/update/{id}", middleware.AuthMiddleware(updateReview)).Methods("POST")

	// **database collections routes (collection & wishlist cTypes)**
	// create a new collection or wishlist based on cType param
	r.Handle(
		"/api/{cType}", middleware.AuthMiddleware(createCollection),
	).Methods("POST")
	// get a collection or wishlist collection by id based on cType param
	r.Handle("/api/{cType}/{id}", middleware.AuthMiddleware(getCollectionTypeById)).Methods("GET")
	// delete an existing collection or wishlist based on cType param
	r.Handle(
		"/api/{cType}/{id}", middleware.AuthMiddleware(deleteCollection),
	).Methods("DELETE")
	// update an existing collection or wishlist name and private flag based on cType param
	r.Handle("/api/{cType}/update/{id}", middleware.AuthMiddleware(updateCollection)).Methods("POST")
	// add or delete a bourbon by id into a collection and a usercollectionref
	// add or delete determined by action placeholder in route as well as cType router param
	r.Handle(
		"/api/{cType}/{action}/{collectionId}/{bourbonId}", middleware.AuthMiddleware(updateBourbonsToCollection),
	).Methods("POST", "DELETE")

	return r
}
