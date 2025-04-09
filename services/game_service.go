package services

import (
	"context"
	"encoding/csv"
	"fmt"
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

func (s *GameService) GetBattlePair(ctx context.Context) (*models.BattleResponse, error) {
	// Read the CSV file where movie titles are stored
	file, err := os.Open("IMDB-Movie-Data.csv")
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %v", err)
	}
	fmt.Println("Do You have a file", file)
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

	// Get two random index of movie titles
	if len(records) < 2 {
		return nil, fmt.Errorf("not enough movies in the CSV file")
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	index1 := rand.Intn(len(records))
	index2 := index1
	// redo if same index ()
	for index2 == index1 {
		index2 = rand.Intn(len(records))
	}

	movieA := models.Movie{
		Title: records[index1][1], // Title is in the second column
	}

	movieB := models.Movie{
		Title: records[index2][1],
	}

	fmt.Println("Do You have a movieA", movieA)
	fmt.Println("Do You have a movieB", movieB)

	// Fetch movie details from OMDB API
	movieADetailsA, err := s.FetchMovieFromOMDB(ctx, movieA.Title)
	if err != nil {
		return nil, fmt.Errorf("error fetching MovieA details from OMDB API: %v", err)
	}

	movieBDetailsB, err := s.FetchMovieFromOMDB(ctx, movieB.Title)
	if err != nil {
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
		MovieA:    models.Movie{ID: req.MovieAID},
		MovieB:    models.Movie{ID: req.MovieBID},
		Winner:    models.Movie,
		CreatedAt: time.Now(),
	}

	// Submit the battle result
	return s.battleRepo.SubmitBattle(ctx, battle)
}

// GetTopTwenty returns the top twenty movies based on battle wins
func (s *GameService) GetTopTwenty(ctx context.Context) ([]models.Movie, error) {
	return s.battleRepo.GetTopTwenty(ctx)
}
