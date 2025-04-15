package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"movie-vs-backend/data_access"
	"movie-vs-backend/models"
)

type GameService struct {
	omdbClient *data_access.OMDBClient
	movieRepo  *data_access.MovieRepository
	battleRepo *data_access.BattleRepository
	userRepo   *data_access.UserRepository
	userStates map[primitive.ObjectID]*models.UserBattleState
	stateMutex sync.RWMutex
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

func (s *GameService) GetBattlePair(ctx context.Context, userID primitive.ObjectID) (*models.BattleResponse, error) {

	// Get or create user battle state
	s.stateMutex.Lock()
	if s.userStates == nil {
		s.userStates = make(map[primitive.ObjectID]*models.UserBattleState)
	}
	userState, exists := s.userStates[userID]
	if !exists {
		userState = &models.UserBattleState{
			UserID:      userID,
			BattleCount: 0,
			LastUpdated: time.Now(),
		}
		s.userStates[userID] = userState
	}
	
	// Increment battle count and handle special cases
	userState.BattleCount++
	if userState.BattleCount > 10 {
		userState.BattleCount = 1
	}
	
	// Trigger different repository methods based on battle count
	switch userState.BattleCount {
	case 3:
		// Get top twenty movies asynchronously
		go func() {
			_, _ = s.GetTopTwenty(ctx, userID)
		}()
	case 5:
		// Get top ten wins asynchronously
		go func() {
			_, _ = s.battleRepo.GetTopTwenty(ctx, userID) // Using GetTopTwenty as a substitute since GetTopTenWins doesn't exist
		}()
	case 10:
		// Reset counter and get all stats asynchronously
		go func() {
			_, _ = s.GetTopTwenty(ctx, userID) // Using GetTopTwenty as a substitute since GetAllStats doesn't exist
		}()
	}
	
	userState.LastUpdated = time.Now()
	s.stateMutex.Unlock()

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

	if s.AreMoviesIdentical(movieA, movieB) {
		movieA, err = s.getRandomMovieFromCSV()
		if err != nil {
			return nil, fmt.Errorf("error getting second movie: %v", err)
		}
	}

	// Fetch movie details from OMDB API with retries
	var movieDetailsA, movieDetailsB *models.Movie
	for {
		var err error

		movieDetailsA, err = s.FetchMovieFromOMDB(ctx, movieA.Title)
		if err != nil {
			fmt.Println("ERROR IN MovieA", err)
			movieA, err = s.getRandomMovieWithRetries(maxRetries)
			if err != nil {
				return nil, fmt.Errorf("error getting new random movie A: %v", err)
			}
			continue // Try again with the new movie
		}
		break // Successfully got movie A details
	}

	for {
		var err error
		// Fetch movie details from OMDB API
		movieDetailsB, err = s.FetchMovieFromOMDB(ctx, movieB.Title)
		if err != nil {
			fmt.Println("ERROR IN MovieB:: ", err)
			// Get new random movie with retries if OMDB API fails
			movieB, err = s.getRandomMovieWithRetries(maxRetries)
			if err != nil {
				return nil, fmt.Errorf("error getting new random movie B: %v", err)
			}
			continue // Try again with the new movie
		}
		break // Successfully got movie B details
	}

	fmt.Println("Do You have a movieA", movieDetailsA.Title)
	fmt.Println("Do You have a movieB", movieDetailsB.Title)

	fmt.Printf("Searching for Movie A - Title: %s, Year: %s, IMDB ID: %s\n", movieDetailsA.Title, movieDetailsA.Year, movieDetailsA.IMDBID)
	movieAFromMongo, err := s.movieRepo.FindMovieByTitle(ctx, movieDetailsA.Title)
	if err != nil {
		return nil, fmt.Errorf("error getting MovieA from MongoDB: %v", err)
	}
	if movieAFromMongo == nil {
		fmt.Printf("Movie not found in MongoDB: %s\n", movieDetailsA.Title)
	}

	movieBFromMongo, err := s.movieRepo.FindMovieByTitle(ctx, movieDetailsB.Title)
	if err != nil {
		return nil, fmt.Errorf("error getting MovieB from MongoDB: %v", err)
	}
	if movieBFromMongo == nil {
		fmt.Printf("Movie not found in MongoDB: %s\n", movieDetailsB.Title)
	}

	fmt.Println("MONGO MOVIE A TITLE", movieAFromMongo.Title)
	fmt.Println("MONGO MOVIE B TITLE", movieBFromMongo.Title)

	movieDetailsA.ID = movieAFromMongo.ID
	movieDetailsB.ID = movieBFromMongo.ID

	fmt.Println("Do You have a movie A ID AFTER", movieDetailsA.ID)
	fmt.Println("Do You have a movie B ID AFTER", movieDetailsB.ID)

	return &models.BattleResponse{
		MovieA: *movieDetailsA,
		MovieB: *movieDetailsB,
	}, nil

}

// SubmitBattle handles the submission of a battle result
func (s *GameService) SubmitBattle(ctx context.Context, userID primitive.ObjectID, req *models.SubmitBattleRequest) error {
	// Create a new battle record
	battle := &models.Battle{
		MovieA: req.MovieA,
		MovieB: req.MovieB,
		Winner: req.Winner,
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
	winnerRanking.MovieTitle = winner.Title
	winnerRanking.MatchCount++
	winnerRanking.WinCount++
	winnerRanking.LastUpdated = time.Now()

	// Update loser ranking
	loserRanking.ELORating = int(newLoserRanking)
	loserRanking.MovieTitle = loser.Title
	loserRanking.MatchCount++
	loserRanking.LossCount++
	loserRanking.LastUpdated = time.Now()

	winnerJSON, _ := json.MarshalIndent(winnerRanking, "", "  ")
	loserJSON, _ := json.MarshalIndent(loserRanking, "", "  ")
	fmt.Printf("THIS IS THE WINNER:\n%s\n", string(winnerJSON))
	fmt.Printf("THIS IS THE LOSER:\n%s\n", string(loserJSON))

	// Save updated rankings
	if err := s.battleRepo.SaveMovieRanking(ctx, userID, winnerRanking); err != nil {
		fmt.Println("ERROR In SaveMovieRanking Winner")
		return fmt.Errorf("error saving winner ranking: %v", err)
	}
	if err := s.battleRepo.SaveMovieRanking(ctx, userID, loserRanking); err != nil {
		fmt.Println("ERROR In SaveMovieRanking Loser")
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
func (s *GameService) GetTopTwenty(ctx context.Context, userID primitive.ObjectID) ([]models.MovieRanking, error) {
	return s.battleRepo.GetTopTwenty(ctx, userID)
}
