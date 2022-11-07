package main

import (
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/handlers"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/middleware"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		},
	)
}

func main() {
	//Connection mongoDB
	db.ConnectDB()

	// Optional Initial Seed of Db
	//data.SeedDBRecords()

	//Init Router
	r := mux.NewRouter()
	// basic middleware
	r.Use(commonMiddleware)
	// arrange our routes
	// bourbon routes
	r.HandleFunc("/api/bourbons", handlers.GetBourbons).Methods("GET")
	r.HandleFunc(
		"/api/bourbons/random", handlers.GetRandomBourbon,
	).Methods("GET")
	r.HandleFunc("/api/bourbons/{id}", handlers.GetBourbonById).Methods("GET")
	// user routes
	//r.HandleFunc("/api/user", handlers.CreateUser).Methods("GET")
	// create and login a new user
	newUserHandler := http.HandlerFunc(handlers.CreateUser)
	logoutUserHandler := http.HandlerFunc(handlers.LogoutUser)
	r.Handle("/api/user", middleware.NewUserMiddleware(newUserHandler))
	// login an existing user
	r.HandleFunc("/api/user/login", handlers.LoginUser).Methods("GET")
	// logout a user
	r.Handle("/api/user/logout", middleware.AuthMiddleware(logoutUserHandler))
	// collection routes - all authenticated routes?
	createCollectionHandler := http.HandlerFunc(handlers.CreateCollection)
	deleteCollectionHandler := http.HandlerFunc(handlers.DeleteCollection)
	addBourbonToCollectionHandler := http.HandlerFunc(handlers.AddBourbonToCollection)
	// create a new collection
	r.Handle(
		"/api/collection", middleware.AuthMiddleware(createCollectionHandler),
	)
	// delete an existing collection
	r.Handle(
		"/api/collection/{id}", middleware.AuthMiddleware(deleteCollectionHandler),
	)
	// add a bourbon by id into a collection and a usercollectionref
	r.Handle(
		"/api/collection/add/{id}", middleware.AuthMiddleware(addBourbonToCollectionHandler),
	)
	// set our port address
	fmt.Println("Server is up at port 5000")
	log.Fatal(http.ListenAndServe(":5000", r))

}
