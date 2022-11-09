package handlers

import (
	"context"
	"errors"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/db"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/models"
	"github.com/GoloisaNinja/go-bourbon-api/pkg/responses"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"regexp"
	"strconv"
)

type BourbonsResponse struct {
	Bourbons     []models.Bourbon `json:"bourbons"`
	TotalRecords int              `json:"total_records"`
}

// declare and set collections to collection vars
var bourbonsCollection = db.GetCollection(
	db.DB,
	"bourbons",
)

// GetBourbons gets paginated bourbons
func GetBourbons(w http.ResponseWriter, r *http.Request) {
	sortQuery := "title"
	searchQuery := " "
	sortDirection := 1
	limit := 20
	page := 1
	q := r.URL.Query()
	if q.Get("page") != "" && q.Get("page") != "1" {
		p, err := strconv.Atoi(q.Get("page"))
		if err != nil {
			var er responses.ErrorResponse
			er.Respond(w, 500, "error", err.Error())
			return
		}
		page = p
	}
	skip := (page - 1) * limit
	if q.Get("sort") != "" && q.Get("sort") != "title_asc" {
		//r := regexp.MustCompile(`^(?P<S>\w+)_(?P<D>\w+)$`)
		r, err := regexp.Compile(`^(?P<S>\w+)_(?P<D>\w+)$`)
		if err != nil {
			var er responses.ErrorResponse
			er.Respond(w, 400, "error", err.Error())
			return
		}
		res := r.FindStringSubmatch(q.Get("sort"))
		if len(res) == 0 {
			resLenErr := errors.New("sort params in request were bad")
			var er responses.ErrorResponse
			er.Respond(w, 400, "error", resLenErr.Error())
			return
		}
		sortIndex := r.SubexpIndex("S")
		dirIndex := r.SubexpIndex("D")
		switch res[sortIndex] {
		case "abv":
			sortQuery = "abv_value"
		case "age":
			sortQuery = "age_value"
		case "price":
			sortQuery = "price_value"
		case "score":
			sortQuery = "review.score"
		default:
			sortQuery = res[sortIndex]
		}
		if res[dirIndex] == "desc" {
			sortDirection = -1
		}
	}

	if q.Get("search") != " " {
		searchQuery = q.Get("search")
	}
	//opts := options.Find().SetSort(bson.D{{sortQuery, sortDirection}}).SetSkip(int64(skip)).SetLimit(int64(limit))
	sr := primitive.Regex{searchQuery, "i"}
	// working filter lol
	//filter := bson.M{"title": sr}
	// boss level filter that incorporates title, bottler, distiller
	filter := bson.M{
		"$or": []bson.M{
			bson.M{"title": sr},
			bson.M{"bottler": sr}, bson.M{"distiller": sr},
		},
	}
	// working matchStage lol
	//matchStage := bson.D{{"$match", bson.D{{"title", sr}}}}
	// boss level matchStage that incorporates title, bottler, distiller
	orStage := []bson.D{
		bson.D{{"title", sr}}, bson.D{{"bottler", sr}},
		bson.D{{"distiller", sr}},
	}
	matchStage := bson.D{{"$match", bson.D{{"$or", orStage}}}}
	sortStage := bson.D{{"$sort", bson.D{{sortQuery, sortDirection}}}}
	skipStage := bson.D{{"$skip", skip}}
	limitStage := bson.D{{"$limit", limit}}
	count, ctErr := bourbonsCollection.CountDocuments(
		context.TODO(),
		filter,
	)
	if ctErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", ctErr.Error())
		return
	}

	var bourbons []models.Bourbon
	cursor, fetchErr := bourbonsCollection.Aggregate(
		context.TODO(),
		mongo.Pipeline{matchStage, sortStage, skipStage, limitStage},
	)
	if fetchErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", fetchErr.Error())
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		var bourbon models.Bourbon
		err := cursor.Decode(&bourbon)
		if err != nil {
			var er responses.ErrorResponse
			er.Respond(w, 500, "error", err.Error())
			return
		}
		bourbons = append(
			bourbons,
			bourbon,
		)
	}

	if cursErr := cursor.Err(); cursErr != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", cursErr.Error())
		return
	}
	if len(bourbons) > 0 {
		br := BourbonsResponse{
			Bourbons:     bourbons,
			TotalRecords: int(count),
		}
		var successResponse responses.StandardResponse
		successResponse.Respond(w, 200, "success", br)
	} else {
		var errResponse responses.ErrorResponse
		nfError := errors.New("not found")
		errResponse.Respond(w, 404, "error", nfError.Error())
	}

}

// GetRandomBourbon gets a random bourbon from the db using a aggregation pipe $sample
func GetRandomBourbon(w http.ResponseWriter, r *http.Request) {
	pipeline := []bson.M{bson.M{"$sample": bson.M{"size": 1}}}
	cursor, err := bourbonsCollection.Aggregate(
		context.TODO(),
		pipeline,
	)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	defer cursor.Close(context.TODO())
	var bourbon models.Bourbon
	for cursor.Next(context.TODO()) {
		err := cursor.Decode(&bourbon)
		if err != nil {
			var er responses.ErrorResponse
			er.Respond(w, 500, "error", err.Error())
			return
		}
	}
	if err := cursor.Err(); err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	if bourbon.Title != "" {
		var br responses.StandardResponse
		br.Respond(w, 200, "success", bourbon)
	} else {
		err = errors.New("not found")
		var er responses.ErrorResponse
		er.Respond(w, 404, "error", err.Error())
	}

}

// GetBourbonById gets a bourbon from the db using the ID passed in url params
func GetBourbonById(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	// convert id string to ObjectId
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 500, "error", err.Error())
		return
	}
	filter := bson.M{"_id": objectId}
	var bourbon models.Bourbon
	err = bourbonsCollection.FindOne(
		context.TODO(),
		filter,
	).Decode(&bourbon)
	if err != nil {
		var er responses.ErrorResponse
		er.Respond(w, 400, "error", err.Error())
		return
	}
	if bourbon.Title != "" {
		var br responses.StandardResponse
		br.Respond(w, 200, "success", bourbon)
	} else {
		err = errors.New("not found")
		var er responses.ErrorResponse
		er.Respond(w, 404, "error", err.Error())
	}

}
