package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MovieRanking represents a user's personal ranking and stats for a specific movie
type MovieRanking struct {
	MovieID     primitive.ObjectID `bson:"movie_id" json:"movie_id"`
	MovieTitle  string            `bson:"movie_title" json:"movie_title"`         // Denormalized for quick access
	ELORating   int               `bson:"elo_rating" json:"elo_rating"`          // User's personal ELO rating for this movie
	MatchCount  int               `bson:"match_count" json:"match_count"`        // Number of times user has rated this movie
	WinCount    int               `bson:"win_count" json:"win_count"`            // Number of times user chose this movie
	LossCount   int               `bson:"loss_count" json:"loss_count"`          // Number of times user didn't choose this movie
	LastUpdated time.Time         `bson:"last_updated" json:"last_updated"`      // Last time user rated this movie
}
