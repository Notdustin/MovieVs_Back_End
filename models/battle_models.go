package models

type BattleResponse struct {
	MovieA Movie `json:"movie_a"`
	MovieB Movie `json:"movie_b"`
}

type SubmitBattleRequest struct {
	Winner Movie `json:"winner" binding:"required"`
	MovieA Movie `json:"movie_a" binding:"required"`
	MovieB Movie `json:"movie_b" binding:"required"`
}
