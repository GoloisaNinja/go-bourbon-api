package main

import (
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/gorilla/handlers"
	"log"
	"net/http"
)

const PORT = ":5000"

func main() {
	// Connection mongoDB
	db.ConnectDB()

	// Optional Initial Seed of Db
	//data.SeedDBRecords()
	// cors
	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With", "Authorization", "Bearer", "Accept", "Accept-Language", "Origin"})
	originOk := handlers.AllowedOrigins([]string{"http://localhost:3000"})
	methodsOk := handlers.AllowedMethods([]string{"PUT", "POST", "GET", "DELETE", "OPTIONS"})
	// bring in the routes to serve
	srv := &http.Server{
		Addr:    PORT,
		Handler: handlers.CORS(originOk, headersOk, methodsOk)(routes()),
	}

	fmt.Printf("Server is up on port %s", PORT)
	err := srv.ListenAndServe()
	log.Fatal(err)

}
