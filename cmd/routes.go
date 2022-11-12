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
	createNewUserHandler := http.HandlerFunc(handlers.CreateUser)
	logoutUserHandler := http.HandlerFunc(handlers.LogoutUser)

	// collection handlers
	getCollectionById := http.HandlerFunc(handlers.GetCollectionById)
	createCollectionHandler := http.HandlerFunc(handlers.CreateCollection)
	updateCollectionHandler := http.HandlerFunc(handlers.UpdateCollection)
	deleteCollectionHandler := http.HandlerFunc(handlers.DeleteCollection)
	addBourbonToCollectionHandler := http.HandlerFunc(handlers.AddBourbonToCollection)
	deleteBourbonFromCollectionHandler := http.HandlerFunc(handlers.DeleteBourbonFromCollection)

	// wishlist handlers
	getWishlistById := http.HandlerFunc(handlers.GetWishlistById)
	createWishlistHandler := http.HandlerFunc(handlers.CreateWishlist)

	// define routes
	// bourbon routes
	// get paginated bourbons
	r.HandleFunc("/api/bourbons", handlers.GetBourbons).Methods("GET")
	// get a randomized bourbon
	r.HandleFunc(
		"/api/bourbons/random", handlers.GetRandomBourbon,
	).Methods("GET")
	// get a bourbon by id
	r.HandleFunc("/api/bourbons/{id}", handlers.GetBourbonById).Methods("GET")

	// user routes
	// create a new user
	r.Handle("/api/user", middleware.NewUserMiddleware(createNewUserHandler)).Methods("POST")
	// login an existing user
	r.HandleFunc("/api/user/login", handlers.LoginUser).Methods("GET")
	// logout a user
	r.Handle("/api/user/logout", middleware.AuthMiddleware(logoutUserHandler)).Methods("POST")

	// collections routes
	// get a collection by id
	r.Handle("/api/collection/{id}", middleware.AuthMiddleware(getCollectionById)).Methods("GET")
	// create a new collection
	r.Handle(
		"/api/collection", middleware.AuthMiddleware(createCollectionHandler),
	).Methods("POST")
	// update an existing collection name and private flag
	r.Handle("/api/collection/update/{id}", middleware.AuthMiddleware(updateCollectionHandler)).Methods("POST")
	// delete an existing collection
	r.Handle(
		"/api/collection/{id}", middleware.AuthMiddleware(deleteCollectionHandler),
	).Methods("DELETE")
	// add a bourbon by id into a collection and a usercollectionref
	r.Handle(
		"/api/collection/add/{id}", middleware.AuthMiddleware(addBourbonToCollectionHandler),
	).Methods("POST")
	// remove a bourbon by id from a collection and a usercollectionref
	r.Handle(
		"/api/collection/delete/{collectionId}/{bourbonId}", middleware.AuthMiddleware(deleteBourbonFromCollectionHandler),
	).Methods("DELETE")

	// wishlist routes
	// get wishlist by id
	r.Handle("/api/wishlist/{id}", middleware.AuthMiddleware(getWishlistById)).Methods("GET")
	// create new wishlist
	r.Handle("/api/wishlist", middleware.AuthMiddleware(createWishlistHandler)).Methods("POST")

	return r
}
