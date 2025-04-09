package controllers

import (
	"movie-vs-backend/models"
	"movie-vs-backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var req models.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var message string
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, e := range ve {
				switch e.Field() {
				case "Email":
					message = "Please provide a valid email address"
				case "Password":
					if e.Tag() == "min" {
						message = "Password must be at least 6 characters long"
					} else {
						message = "Password is required"
					}
				default:
					message = "Invalid input data"
				}
				break // Only show first error
			}
		} else {
			message = "Invalid request format"
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	user, err := c.authService.Register(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req models.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var message string
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, e := range ve {
				switch e.Field() {
				case "Email":
					message = "Please provide a valid email address"
				case "Password":
					if e.Tag() == "min" {
						message = "Password must be at least 6 characters long"
					} else {
						message = "Password is required"
					}
				default:
					message = "Invalid input data"
				}
				break // Only show first error
			}
		} else {
			message = "Invalid request format"
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	token, err := c.authService.Login(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": token})
}

func (c *AuthController) Logout(ctx *gin.Context) {
	// In a stateless JWT setup, client-side logout is sufficient
	ctx.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
