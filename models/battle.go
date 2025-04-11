package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Battle struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	MovieA    Movie              `bson:"movie_a" json:"movie_a"`
	MovieB    Movie              `bson:"movie_b" json:"movie_b"`
	Winner    Movie              `bson:"winner" json:"winner"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
