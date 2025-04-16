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
	// Maximum number of retries (Some Movie Titles are not found in OMDB)
	const maxRetries = 3

	var MovieA, MovieB *models.Movie = nil, nil
	var MoviePickA, MoviePickB string = "", ""
	var err error

	// Get or create user battle state
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

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

	fmt.Println("Battle Count:", userState.BattleCount)

	// Define a struct to hold both movies
	type moviePair struct {
		MovieA *models.Movie
		MovieB *models.Movie
	}

	// Trigger different repository methods based on battle count
	// Will get random movie from CSV by default
	// Case 3 will get a random movie from top ten matches for MovieA
	// Case 5 will get a random movie from top ten wins for MovieB
	// Case 10 will get a random movie from top twenty for both MovieA and MovieB
	switch userState.BattleCount {
	case 3:

		// Get top ten matches with timeout
		movieChan := make(chan *moviePair)
		go func() {
			fmt.Println("Getting top ten matches...")
			bgCtx := context.Background()
			rand.New(rand.NewSource(time.Now().UnixNano()))
			index := rand.Intn(10)
			topTenMatches, err := s.battleRepo.GetTopTenByMatches(bgCtx, userID)
			if err != nil {
				fmt.Printf("Error getting top ten matches: %v\n", err)
				movieChan <- nil
				return
			}
			if len(topTenMatches) > 0 {
				movieA, err := s.FetchMovieFromOMDB(ctx, topTenMatches[index].MovieTitle)
				if err != nil {
					fmt.Printf("Error getting movie A in case 3: %v\n", err)
					movieChan <- nil
					return
				}

				movieB, err := s.getRandomMovieWithRetries(maxRetries)
				if err != nil {
					fmt.Printf("Error getting movie B in case 3: %v\n", err)
					movieChan <- nil
					return
				}

				movieChan <- &moviePair{MovieA: movieA, MovieB: movieB}
			} else {
				movieChan <- nil
			}
		}()

		// Wait for movies with timeout
		select {
		case pair := <-movieChan:
			if pair != nil {
				MovieA = pair.MovieA
				MovieB = pair.MovieB
			}
		case <-time.After(5 * time.Second):
			fmt.Println("Timeout waiting for top ten matches")
			return nil, fmt.Errorf("timeout waiting for top ten matches")
		}

	case 5:
		// Get top ten wins with timeout
		movieChan := make(chan *moviePair)
		go func() {
			fmt.Println("Getting top ten wins...")
			bgCtx := context.Background()
			rand.New(rand.NewSource(time.Now().UnixNano()))
			index := rand.Intn(10)
			topTenWins, err := s.battleRepo.GetTopTenByWins(bgCtx, userID)
			if err != nil {
				fmt.Printf("Error getting top ten wins: %v\n", err)
				movieChan <- nil
				return
			}
			if len(topTenWins) > 0 {
				movieA, err := s.getRandomMovieWithRetries(maxRetries)
				if err != nil {
					fmt.Printf("Error getting movie A in case 5: %v\r\n", err)
					movieChan <- nil
					return
				}

				movieB, err := s.FetchMovieFromOMDB(ctx, topTenWins[index].MovieTitle)
				if err != nil {
					fmt.Printf("Error getting movie B in case 5: %v\n", err)
					movieChan <- nil
					return
				}
				movieChan <- &moviePair{MovieA: movieA, MovieB: movieB}

			} else {
				movieChan <- nil
			}
		}()

		// Wait for movie with timeout
		select {
		case pair := <-movieChan:
			if pair != nil {
				MovieA = pair.MovieA
				MovieB = pair.MovieB
			}
		case <-time.After(5 * time.Second):
			fmt.Println("Timeout waiting for top ten wins")
		}
	case 10:
		// Reset counter and get all stats asynchronously
		movieChan := make(chan *moviePair)
		go func() {
			fmt.Println("Getting top twenty...")
			bgCtx := context.Background()
			rand.New(rand.NewSource(time.Now().UnixNano()))
			randomMovieIndex1 := rand.Intn(20)
			randomMovieIndex2 := rand.Intn(20)
			topTwenty, err := s.battleRepo.GetTopTwenty(bgCtx, userID)
			if err != nil {
				fmt.Printf("Error getting top twenty: %v\n", err)
			} else if len(topTwenty) > 0 {
				// Use the first movie from top twenty as movieA
				MoviePickA = topTwenty[randomMovieIndex1].MovieTitle
				MovieA, err = s.FetchMovieFromOMDB(ctx, MoviePickA)
				if err != nil {
					fmt.Printf("Error getting movie A in case 10: %v\n", err)
				}
				// Use the second movie from top twenty as movieB
				MoviePickB = topTwenty[randomMovieIndex2].MovieTitle
				MovieB, err = s.FetchMovieFromOMDB(ctx, MoviePickB)
				if err != nil {
					fmt.Printf("Error getting movie B in case 10: %v\n", err)
				}
				fmt.Println("Movie A in case 10:", MovieA)
				fmt.Println("Movie B in case 10:", MovieB)
				movieChan <- &moviePair{MovieA: MovieA, MovieB: MovieB}
			}
		}()
		// Wait for movie with timeout
		select {
		case pair := <-movieChan:
			if pair != nil {
				MovieA = pair.MovieA
				MovieB = pair.MovieB
			}
		case <-time.After(5 * time.Second):
			fmt.Println("Timeout waiting for top twenty")
		}
	default:
		fmt.Println("Do You have a movie Pick??", MoviePickA, MoviePickB)

		// Get first movie with retries
		MovieA, err = s.getRandomMovieWithRetries(maxRetries)
		if err != nil {
			return nil, err
		}

		// Get second movie with retries
		MovieB, err = s.getRandomMovieWithRetries(maxRetries)
		if err != nil {
			return nil, err
		}
		// reset random movie pick
		MoviePickA = ""
		MoviePickB = ""

	}

	userState.LastUpdated = time.Now()

	if s.AreMoviesIdentical(MovieA, MovieB) {
		MovieA, err = s.getRandomMovieFromCSV()
		if err != nil {
			return nil, fmt.Errorf("error getting second movie: %v", err)
		}
	}

	// Fetch movie details from OMDB API with retries
	var movieDetailsA, movieDetailsB *models.Movie
	for {
		var err error
		fmt.Println("MOVIEAAAAAAAAAAAAAAAAAAA:", MovieA)
		fmt.Println("MOVIEBBBBBBBBBBBBBBBBBBB:", MovieB)

		fmt.Println("MOVIE A TITLE ln 301", MovieA.Title)
		movieDetailsA, err = s.FetchMovieFromOMDB(ctx, MovieA.Title)
		if err != nil {
			fmt.Println("ERROR IN MovieA", err)
			MovieA, err = s.getRandomMovieWithRetries(maxRetries)
			if err != nil {
				return nil, fmt.Errorf("error getting new random movie A: %v", err)
			}
			continue // Try again with the new movie
		}
		break // Successfully got movie A details
	}

	for {
		var err error
		fmt.Println("MOVIE B TITLE ln 250", MovieB.Title)
		// Fetch movie details from OMDB API
		movieDetailsB, err = s.FetchMovieFromOMDB(ctx, MovieB.Title)
		if err != nil {
			fmt.Println("ERROR IN MovieB:: ", err)
			// Get new random movie with retries if OMDB API fails
			MovieB, err = s.getRandomMovieWithRetries(maxRetries)
			if err != nil {
				return nil, fmt.Errorf("error getting new random movie B: %v", err)
			}
			continue // Try again with the new movie
		}
		break // Successfully got movie B details
	}

	fmt.Println("Do You have a movieA", movieDetailsA.Title)
	fmt.Println("Do You have a movieB", movieDetailsB.Title)

	movieAFromMongo, err := s.movieRepo.FindMovieByTitle(ctx, movieDetailsA.Title)
	if err != nil {
		return nil, fmt.Errorf("error getting MovieA from MongoDB: %v", err)
	}
	if movieAFromMongo == nil {
		fmt.Printf("Movie A not found in MongoDB: %s, restarting flow...\n", movieDetailsA.Title)
		// Reset battle count before releasing lock
		userState.BattleCount = 0
		// Release the mutex lock before restarting
		s.stateMutex.Unlock()
		// Restart the Get Battle Pair flow
		return s.GetBattlePair(ctx, userID)
	}

	movieBFromMongo, err := s.movieRepo.FindMovieByTitle(ctx, movieDetailsB.Title)
	if err != nil {
		return nil, fmt.Errorf("error getting MovieB from MongoDB: %v", err)
	}
	if movieBFromMongo == nil {
		fmt.Printf("Movie B not found in MongoDB: %s, restarting flow...\n", movieDetailsB.Title)
		// Reset battle count before releasing lock
		userState.BattleCount = 0
		// Release the mutex lock before restarting
		s.stateMutex.Unlock()
		// Restart the flow
		return s.GetBattlePair(ctx, userID)
	}

	// Both movies exist in MongoDB, safe to proceed
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

	loserRanking, err := s.battleRepo.GetMovieRanking(ctx, userID, loser.ID)
	if err != nil {
		return fmt.Errorf("error getting loser ranking: %v", err)
	}

	// Elo math
	// ra
	currentWinnerRanking := float64(winnerRanking.ELORating)
	// rb
	currentLoserRanking := float64(loserRanking.ELORating)

	ea := 1.0 / (1.0 + math.Pow(10, (currentLoserRanking-currentWinnerRanking)/400))
	eb := 1.0 / (1.0 + math.Pow(10, (currentWinnerRanking-currentLoserRanking)/400))

	newWinnerRanking := currentWinnerRanking + K*(1-ea)
	newLoserRanking := currentLoserRanking + K*(0-eb)

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
func (s *GameService) GetTopTwenty(ctx context.Context, userID primitive.ObjectID) ([]models.MovieRanking, error) {
	return s.battleRepo.GetTopTwenty(ctx, userID)
}
