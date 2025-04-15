package models

// OmdbResponse represents the response from the OMDB API
type OmdbResponse struct {
	Response   string `json:"Response"`
	Error      string `json:"Error"`
	Title      string `json:"Title" bson:"title"`
	Year       string `json:"Year" bson:"year"`
	Plot       string `json:"Plot" bson:"plot"`
	Director   string `json:"Director" bson:"director"`
	Poster     string `json:"Poster" bson:"poster_url"`
	Genre      string `json:"Genre" bson:"genre"`
	Actors     string `json:"Actors" bson:"actors"`
	ImdbRating string `json:"imdbRating" bson:"imdb_rating"`
	ImdbID     string `json:"imdbID" bson:"imdb_id"`
}
