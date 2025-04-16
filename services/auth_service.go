package services

import (
	"context"
	"errors"
	"fmt"
	"movie-vs-backend/data_access"
	"movie-vs-backend/helper"
	"movie-vs-backend/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  *data_access.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo *data_access.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (string, error) {
	existingUser, _ := s.userRepo.FindByEmail(ctx, req.Email)
	if existingUser != nil {
		return "", errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	movieRankings, err := helper.InitializeMovieRankings()
	if err != nil {
		return "", err
	}

	user := &models.User{
		Email:         req.Email,
		Password:      string(hashedPassword),
		CreatedAt:     time.Now(),
		MovieRankings: movieRankings,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	fmt.Println("tokenstring???", tokenString)

	return tokenString, nil
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (string, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return "", errors.New("invalid credentials - email not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", errors.New("invalid credentials - password problem")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	fmt.Println("tokenstring???", tokenString)

	return tokenString, nil
}
