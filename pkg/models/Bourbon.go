package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Review struct {
	Intro   string `json:"intro"`
	Nose    string `json:"nose"`
	Taste   string `json:"taste"`
	Finish  string `json:"finish"`
	Overall string `json:"overall"`
	Score   string `json:"score"`
	Author  string `json:"author"`
}

type Bourbon struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id"`
	Title      string             `json:"title"`
	Image      string             `json:"image"`
	Distiller  string             `json:"distiller"`
	Bottler    string             `json:"bottler"`
	Abv        string             `json:"abv"`
	AbvValue   float64            `json:"abv_value" bson:"abv_value"`
	Age        string             `json:"age"`
	AgeValue   int                `json:"age_value" bson:"age_value"`
	PriceArray []string           `json:"price_array" bson:"price_array"`
	PriceValue int                `json:"price_value" bson:"price_value"`
	Review     *Review            `json:"review"`
}
