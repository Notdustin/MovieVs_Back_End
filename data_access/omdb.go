package data_access

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"movie-vs-backend/models"
)

type OMDBClient struct {
	apiKey  string
	baseURL string
}

func NewOMDBClient(apiKey, baseURL string) *OMDBClient {
	return &OMDBClient{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

func (c *OMDBClient) FetchMovie(ctx context.Context, title string) (*models.Movie, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OMDB API key not found")
	}

	fmt.Println(c.apiKey)

	fmt.Println("Do You have a title???", title)

	// Create the URL with the API key and title
	url := fmt.Sprintf("%s?apikey=%s&t=%s", c.baseURL, c.apiKey, title)

	fmt.Println("URL ================", url)

	// Make the HTTP request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("DIDN'T RCV A MOVIE FROM OMDB", err)
		return nil, fmt.Errorf("error making request to OMDB API: %v", err)
	}
	defer resp.Body.Close()

	// Decode the response
	var omdbResp models.OmdbResponse
	if err := json.NewDecoder(resp.Body).Decode(&omdbResp); err != nil {
		return nil, fmt.Errorf("error decoding OMDB response: %v", err)
	}

	// Create and return the movie model
	movie := &models.Movie{
		Title:      omdbResp.Title,
		Year:       omdbResp.Year,
		Plot:       omdbResp.Plot,
		Director:   omdbResp.Director,
		PosterURL:  omdbResp.Poster,
		Genre:      omdbResp.Genre,
		Actors:     omdbResp.Actors,
		IMDBRating: omdbResp.ImdbRating,
		IMDBID:     omdbResp.ImdbID,
	}

	return movie, nil
}
