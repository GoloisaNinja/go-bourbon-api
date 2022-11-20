package main

import (
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/gorilla/handlers"
	"log"
	"net/http"
	"os"
)

func main() {
	// Connection mongoDB
	db.ConnectDB()

	// Optional Initial Seed of Db
	//data.SeedDBRecords()
	// cors
	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With", "Authorization", "Bearer", "Accept", "Accept-Language", "Origin"})
	originOk := handlers.AllowedOrigins([]string{"https://hellogobourbon.netlify.app", "http://localhost:3000"})
	methodsOk := handlers.AllowedMethods([]string{"PUT", "POST", "GET", "DELETE", "OPTIONS"})
	// set port
	port := ":" + os.Getenv("PORT")
	if port == "" {
		port = ":5000"
	}
	// bring in the routes to serve
	srv := &http.Server{
		Addr:    port,
		Handler: handlers.CORS(originOk, headersOk, methodsOk)(routes()),
	}

	fmt.Printf("Server is up on port %s", port)
	err := srv.ListenAndServe()
	log.Fatal(err)

}
