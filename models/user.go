package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	// User information
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"-"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	LastLogin time.Time          `bson:"last_login" json:"last_login"`

	// Movie rankings - each user's personal movie ratings and stats
	MovieRankings []MovieRanking `bson:"movie_rankings" json:"movie_rankings"`
}
