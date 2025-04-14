package controllers

import (
	"net/http"
	"movie-vs-backend/models"
	"movie-vs-backend/services"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GameController struct {
	gameService *services.GameService
}

func NewGameController(gameService *services.GameService) *GameController {
	return &GameController{
		gameService: gameService,
	}
}

func (c *GameController) GetMovieBattlePair(ctx *gin.Context) {
	response, err := c.gameService.GetBattlePair(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *GameController) GetTopTwentyList(ctx *gin.Context) {
	users, err := c.gameService.GetTopTwenty(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top users"})
		return
	}

	ctx.JSON(http.StatusOK, users)
}

func (c *GameController) SubmitBattleWinner(ctx *gin.Context) {
	fmt.Println("Submit Battle Winer ENDPOINT", ctx)

	var req models.SubmitBattleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := ctx.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := c.gameService.SubmitBattle(ctx.Request.Context(), objID, &req); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit battle"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Battle result recorded successfully"})
}
