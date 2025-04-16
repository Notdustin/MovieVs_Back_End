package controllers

import (
	"movie-vs-backend/models"
	"movie-vs-backend/services"
	"net/http"

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
	userID, exists := ctx.Get("user_id")

	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	response, err := c.gameService.GetBattlePair(ctx.Request.Context(), userObjectID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *GameController) GetTopTwentyList(ctx *gin.Context) {
	// Get user ID from context
	userID, _ := ctx.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	response, err := c.gameService.GetTopTwenty(ctx.Request.Context(), objID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top users"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *GameController) SubmitBattleWinner(ctx *gin.Context) {

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
