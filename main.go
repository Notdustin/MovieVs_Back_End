package main

import (
	"context"
	"fmt"
	"log"
	"movie-vs-backend/config"
	"movie-vs-backend/controllers"
	"movie-vs-backend/data_access"
	"movie-vs-backend/middleware"
	"movie-vs-backend/services"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// No constants needed - moved to environment configuration

func setupCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	fmt.Println("Configuration loaded for environment:", cfg.Env)

	// Initialize MongoDB connection
	mongodb, err := data_access.NewMongoDB(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongodb.Close(context.Background())

	// Initialize repositories
	userRepo := data_access.NewUserRepository(mongodb)
	movieRepo := data_access.NewMovieRepository(mongodb)
	battleRepo := data_access.NewBattleRepository(mongodb)

	// Set JWT secret for middleware
	middleware.SetJWTSecret(cfg.JWTSecret)

	// Initialize services
	authService := services.NewAuthService(userRepo, cfg.JWTSecret)
	gameService := services.NewGameService(cfg.MovieAPIKey, cfg.MovieAPIBaseURL, movieRepo, battleRepo, userRepo)

	// Initialize controllers
	authController := controllers.NewAuthController(authService)
	gameController := controllers.NewGameController(gameService)

	// Setup Gin router
	r := gin.Default()
	r.Use(setupCORS())

	// Health check endpoint
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Public routes
	api := r.Group("/api")
	{
		api.POST("/register", authController.Register)
		api.POST("/login", authController.Login)
		api.POST("/logout", authController.Logout)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/battle", gameController.GetMovieBattlePair)
			protected.GET("/leaderboard", gameController.GetTopTwentyList)
			protected.POST("/battle", gameController.SubmitBattleWinner)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
