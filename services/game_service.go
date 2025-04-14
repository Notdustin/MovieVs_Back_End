package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"movie-vs-backend/data_access"
	"movie-vs-backend/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GameService struct {
	omdbClient *data_access.OMDBClient
	movieRepo  *data_access.MovieRepository
	battleRepo *data_access.BattleRepository
	userRepo   *data_access.UserRepository
}

func NewGameService(
	omdbAPIKey string,
	omdbBaseURL string,
	movieRepo *data_access.MovieRepository,
	battleRepo *data_access.BattleRepository,
	userRepo *data_access.UserRepository,
) *GameService {
	return &GameService{
		omdbClient: data_access.NewOMDBClient(omdbAPIKey, omdbBaseURL),
		movieRepo:  movieRepo,
		battleRepo: battleRepo,
		userRepo:   userRepo,
	}
}

func (s *GameService) FetchMovieFromOMDB(ctx context.Context, title string) (*models.Movie, error) {
	return s.omdbClient.FetchMovie(ctx, title)
}

func (s *GameService) getRandomMovieFromCSV() (*models.Movie, error) {
	// Read the CSV file where movie titles are stored
	file, err := os.Open("IMDB-Movie-Data.csv")
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %v", err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Skip the header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV header: %v", err)
	}

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV records: %v", err)
	}

	// Check if there are any movies
	if len(records) < 1 {
		return nil, fmt.Errorf("no movies in the CSV file")
	}

	// Get random movie
	rand.New(rand.NewSource(time.Now().UnixNano()))
	index := rand.Intn(len(records))

	return &models.Movie{
		Title: records[index][1], // Title is in the second column
	}, nil
}

// getRandomMovieWithRetries attempts to get a random movie with a specified number of retries
func (s *GameService) getRandomMovieWithRetries(maxRetries int) (*models.Movie, error) {
	var movie *models.Movie
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		movie, err = s.getRandomMovieFromCSV()
		if err == nil {
			break
		}
		if attempt < maxRetries {
			continue
		}
		return nil, fmt.Errorf("failed to get movie after %d attempts: %v", maxRetries, err)
	}

	return movie, nil
}

func (s *GameService) GetBattlePair(ctx context.Context) (*models.BattleResponse, error) {
	// Maximum number of retries (Some Movie Titles are not found in OMDB)
	const maxRetries = 3

	// Get first movie with retries
	movieA, err := s.getRandomMovieWithRetries(maxRetries)
	if err != nil {
		return nil, err
	}

	// Get second movie with retries
	movieB, err := s.getRandomMovieWithRetries(maxRetries)
	if err != nil {
		return nil, err
	}

	fmt.Println("Do You have a movieA", movieA)
	fmt.Println("Do You have a movieB", movieB)

	if s.AreMoviesIdentical(movieA, movieB) {
		movieA, err = s.getRandomMovieFromCSV()
		if err != nil {
			return nil, fmt.Errorf("error getting second movie: %v", err)
		}
	}

	// Fetch movie details from OMDB API
	movieADetailsA, err := s.FetchMovieFromOMDB(ctx, movieA.Title)
	if err != nil {
		fmt.Println("ERROR IN MovieA", err)
		return nil, fmt.Errorf("error fetching MovieA details from OMDB API: %v", err)
	}

	movieBDetailsB, err := s.FetchMovieFromOMDB(ctx, movieB.Title)
	if err != nil {
		fmt.Println("ERROR IN MovieB", err)
		return nil, fmt.Errorf("error fetching MovieB details from OMDB API: %v", err)
	}

	return &models.BattleResponse{
		MovieA: *movieADetailsA,
		MovieB: *movieBDetailsB,
	}, nil

}

// SubmitBattle handles the submission of a battle result
func (s *GameService) SubmitBattle(ctx context.Context, userID primitive.ObjectID, req *models.SubmitBattleRequest) error {
	// Create a new battle record
	battle := &models.Battle{
		MovieA:    req.MovieA,
		MovieB:    req.MovieB,
		Winner:    req.Winner,
		CreatedAt: time.Now(),
	}

	// Elo constants (determines how much ratings can change after a single match/battle.)
	const K = 32.0

	var winner, loser *models.Movie
	if battle.Winner.Title == battle.MovieA.Title {
		winner = &battle.MovieA
		loser = &battle.MovieB
	} else {
		winner = &battle.MovieB
		loser = &battle.MovieA
	}

	// Load current Elo ratings
	winnerRanking, err := s.battleRepo.GetMovieRanking(ctx, userID, winner.ID)
	if err != nil {
		return fmt.Errorf("error getting winner ranking: %v", err)
	}

	fmt.Println("Do You have a winnerRanking??", winnerRanking)

	loserRanking, err := s.battleRepo.GetMovieRanking(ctx, userID, loser.ID)
	if err != nil {
		return fmt.Errorf("error getting loser ranking: %v", err)
	}

	fmt.Println("Do You have a loserRanking??", loserRanking)

	// Elo math
	// ra
	currentWinnerRanking := float64(winnerRanking.ELORating)
	// rb
	currentLoserRanking := float64(loserRanking.ELORating)

	ea := 1.0 / (1.0 + math.Pow(10, (currentLoserRanking-currentWinnerRanking)/400))
	eb := 1.0 / (1.0 + math.Pow(10, (currentWinnerRanking-currentLoserRanking)/400))

	newWinnerRanking := currentWinnerRanking + K*(1-ea)
	newLoserRanking := currentLoserRanking + K*(0-eb)

	fmt.Println("Do You have a newWinnerRanking??", newWinnerRanking)
	fmt.Println("Do You have a newLoserRanking??", newLoserRanking)

	// Update winner ranking
	winnerRanking.ELORating = int(newWinnerRanking)
	winnerRanking.MatchCount++
	winnerRanking.WinCount++
	winnerRanking.LastUpdated = time.Now()

	// Update loser ranking
	loserRanking.ELORating = int(newLoserRanking)
	loserRanking.MatchCount++
	loserRanking.LossCount++
	loserRanking.LastUpdated = time.Now()

	// Save updated rankings
	if err := s.battleRepo.SaveMovieRanking(ctx, userID, winnerRanking); err != nil {
		return fmt.Errorf("error saving winner ranking: %v", err)
	}
	if err := s.battleRepo.SaveMovieRanking(ctx, userID, loserRanking); err != nil {
		return fmt.Errorf("error saving loser ranking: %v", err)
	}

	return nil
}

// AreMoviesIdentical checks if two movies are identical by comparing all relevant fields
func (s *GameService) AreMoviesIdentical(movieA, movieB *models.Movie) bool {
	// If either movie is nil, they can't be identical
	if movieA == nil || movieB == nil {
		return false
	}

	// Compare all relevant fields
	return movieA.Title == movieB.Title
}

// GetTopTwenty returns the top twenty movies based on battle wins
func (s *GameService) GetTopTwenty(ctx context.Context) ([]models.Movie, error) {
	return s.battleRepo.GetTopTwenty(ctx)
}
