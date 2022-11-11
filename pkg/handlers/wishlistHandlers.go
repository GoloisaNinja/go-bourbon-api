package handlers

import "github.com/GoloisaNinja/go-bourbon-api/pkg/db"

var wishlistsCollection = db.GetCollection(db.DB, "wishlists")
