package main

import (
	"fmt"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"log"
	"net/http"
)

const PORT = ":5000"

func main() {
	// Connection mongoDB
	db.ConnectDB()

	// Optional Initial Seed of Db
	//data.SeedDBRecords()

	// bring in the routes to serve
	srv := &http.Server{
		Addr:    PORT,
		Handler: routes(),
	}

	fmt.Printf("Server is up on port %s", PORT)
	err := srv.ListenAndServe()
	log.Fatal(err)

}
