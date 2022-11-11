package models

type Wishlist struct {
	User     *UserRef   `bson:"user" json:"user"`
	Name     string     `bson:"name" json:"name"`
	Private  bool       `bson:"private" json:"private"`
	Bourbons []*Bourbon `bson:"bourbons" json:"bourbons"`
}
