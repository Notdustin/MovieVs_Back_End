package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BattleResponse struct {
	MovieA Movie `json:"movie_a"`
	MovieB Movie `json:"movie_b"`
}

type SubmitBattleRequest struct {
	WinnerID primitive.ObjectID `json:"winner_id" binding:"required"`
	MovieAID primitive.ObjectID `json:"movie_a_id" binding:"required"`
	MovieBID primitive.ObjectID `json:"movie_b_id" binding:"required"`
}
