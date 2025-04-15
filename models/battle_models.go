package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type BattleResponse struct {
	MovieA Movie `json:"movie_a"`
	MovieB Movie `json:"movie_b"`
}

type SubmitBattleRequest struct {
	Winner Movie `json:"winner" binding:"required"`
	MovieA Movie `json:"movie_a" binding:"required"`
	MovieB Movie `json:"movie_b" binding:"required"`
}

type TopTwentyResponse struct {
	Movies []Movie `json:"movies"`
}

type UserBattleState struct {
	UserID       primitive.ObjectID
	BattleCount  int
	LastUpdated  time.Time
}
