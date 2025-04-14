package data_access

import (
	"context"
	"fmt"
	"movie-vs-backend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	db         *MongoDB
	collection *mongo.Collection
}

type MovieRepository struct {
	db *MongoDB
}

type BattleRepository struct {
	db *MongoDB
}

func NewUserRepository(db *MongoDB) *UserRepository {
	return &UserRepository{
		db:         db,
		collection: db.Collection("users"),
	}
}

func NewMovieRepository(db *MongoDB) *MovieRepository {
	return &MovieRepository{db: db}
}

// FindMovieByTitle searches for a movie in the users collection by its title and returns the movie if found
func (r *MovieRepository) FindMovieByTitle(ctx context.Context, title string) (*models.Movie, error) {
	fmt.Printf("Searching for movie with title: %s\n", title)

	// Find movie in users' movie_rankings array by title
	var result struct {
		MovieRankings []models.MovieRanking `bson:"movie_rankings"`
	}

	err := r.db.Collection("users").FindOne(ctx,
		bson.D{{
			Key:   "movie_rankings.movie_title",
			Value: title,
		}},
	).Decode(&result)

	if err == mongo.ErrNoDocuments {
		fmt.Printf("No movie found with title: %s\n", title)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error finding movie: %v", err)
	}

	// Find the matching movie ranking
	for _, ranking := range result.MovieRankings {
		if ranking.MovieTitle == title {
			return &models.Movie{
				ID:    ranking.MovieID,
				Title: ranking.MovieTitle,
			}, nil
		}
	}

	return nil, nil
}

func NewBattleRepository(db *MongoDB) *BattleRepository {
	return &BattleRepository{db: db}
}

// UserRepository methods
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &user, err
}

// SaveMovieRanking saves or updates a movie ranking for a user
func (r *BattleRepository) SaveMovieRanking(ctx context.Context, userID primitive.ObjectID, ranking *models.MovieRanking) error {
	// First try to find and update an existing ranking
	result, err := r.db.Collection("users").UpdateOne(
		ctx,
		bson.M{
			"_id":                     userID,
			"movie_rankings.movie_id": ranking.MovieID,
		},
		bson.M{
			"$set": bson.M{
				"movie_rankings.$.movie_title":  ranking.MovieTitle,
				"movie_rankings.$.elo_rating":   ranking.ELORating,
				"movie_rankings.$.match_count":  ranking.MatchCount,
				"movie_rankings.$.win_count":    ranking.WinCount,
				"movie_rankings.$.loss_count":   ranking.LossCount,
				"movie_rankings.$.last_updated": ranking.LastUpdated,
			},
		},
	)

	if err != nil {
		return err
	}

	// If no existing ranking was found, add it as a new one
	if result.MatchedCount == 0 {
		_, err = r.db.Collection("users").UpdateOne(
			ctx,
			bson.M{"_id": userID},
			bson.M{"$push": bson.M{"movie_rankings": ranking}},
		)
	}

	return err
}

// GetMovieRanking returns the ranking for a specific movie for a user
func (r *BattleRepository) GetMovieRanking(ctx context.Context, userID primitive.ObjectID, movieID primitive.ObjectID) (*models.MovieRanking, error) {
	// Find the user's document and extract the specific movie ranking
	var result struct {
		MovieRankings []models.MovieRanking `bson:"movie_rankings"`
	}

	err := r.db.Collection("users").FindOne(
		ctx,
		bson.M{"_id": userID},
	).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found: %v", err)
		}
		return nil, err
	}

	// Look for the specific movie ranking in the array
	for _, ranking := range result.MovieRankings {
		if ranking.MovieID == movieID {
			return &ranking, nil
		}
	}

	// If no ranking exists, return a new ranking with default values
	return &models.MovieRanking{
		MovieID:     movieID,
		ELORating:   1200, // Default ELO rating
		MatchCount:  0,
		WinCount:    0,
		LossCount:   0,
		LastUpdated: time.Now(),
	}, nil
}

// GetTopTwenty returns the top twenty movies based on battle wins
func (r *BattleRepository) GetTopTwenty(ctx context.Context) ([]models.Movie, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"winner": bson.M{"$exists": true}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$winner",
			"wins":  bson.M{"$sum": 1},
			"movie": bson.M{"$first": "$movie_a"},
		}}},
		{{Key: "$sort", Value: bson.M{"wins": -1}}},
		{{Key: "$limit", Value: 20}},
		{{Key: "$project", Value: bson.M{
			"_id":    "$movie._id",
			"title":  "$movie.title",
			"year":   "$movie.year",
			"poster": "$movie.poster",
			"wins":   1,
		}}},
	}

	cursor, err := r.db.Collection("battles").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var movies []models.Movie
	if err = cursor.All(ctx, &movies); err != nil {
		return nil, err
	}
	return movies, nil
}
