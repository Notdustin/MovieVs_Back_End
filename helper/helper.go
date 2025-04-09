package helper

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"movie-vs-backend/models"
)

// InitializeMovieRankings reads the IMDB-Movie-Data.csv file and creates MovieRanking objects
// with initial ELO ratings of 1200 and zero counts
func InitializeMovieRankings() ([]models.MovieRanking, error) {
	// Open the CSV file
	file, err := os.Open("IMDB-Movie-Data.csv")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	// Find the index of the title column
	titleIndex := -1
	for i, column := range header {
		if column == "Title" {
			titleIndex = i
			break
		}
	}
	if titleIndex == -1 {
		return nil, errors.New("title column not found in CSV")
	}

	var rankings []models.MovieRanking
	now := time.Now()

	// Read each row and create a MovieRanking object
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Create a new MovieRanking for each movie
		ranking := models.MovieRanking{
			MovieID:     primitive.NewObjectID(),
			MovieTitle:  row[titleIndex],
			ELORating:   1200, // Initial ELO rating
			MatchCount:  0,    // Initial match count
			WinCount:    0,    // Initial win count
			LossCount:   0,    // Initial loss count
			LastUpdated: now,  // Current time
		}

		rankings = append(rankings, ranking)
	}

	return rankings, nil
}
