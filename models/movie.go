package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Movie struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title      string             `bson:"title" json:"title"`
	Year       int                `bson:"year" json:"year"`
	PosterURL  string             `bson:"poster_url" json:"poster_url"`
	Plot       string             `bson:"plot" json:"plot"`
	Director   string             `bson:"director" json:"director"`
	Genre      string             `bson:"genre" json:"genre"`
	Actors     string             `bson:"actors" json:"actors"`
	IMDBRating string             `bson:"imdb_rating" json:"imdb_rating"`
	IMDBID     string             `bson:"imdb_id" json:"imdb_id"`
}
