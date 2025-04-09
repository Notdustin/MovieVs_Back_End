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

// MovieRepository methods
func (r *MovieRepository) GetRandomPair(ctx context.Context) ([]models.Movie, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: 2}}}},
	}

	cursor, err := r.db.Collection("movies").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var movies []models.Movie
	if err = cursor.All(ctx, &movies); err != nil {
		return nil, err
	}
	return movies, nil
}

// SaveMovieRanking saves or updates a movie ranking for a user
func (r *BattleRepository) SaveMovieRanking(ctx context.Context, userID primitive.ObjectID, ranking *models.MovieRanking) error {
	// Update the specific movie ranking in the user's movie_rankings array
	_, err := r.db.Collection("users").UpdateOne(
		ctx,
		bson.M{
			"_id": userID,
			"movie_rankings.movie_id": ranking.MovieID,
		},
		bson.M{
			"$set": bson.M{"movie_rankings.$": ranking},
		},
	)

	if err != nil {
		// If the movie ranking doesn't exist in the array, push it
		if err == mongo.ErrNoDocuments {
			_, err = r.db.Collection("users").UpdateOne(
				ctx,
				bson.M{"_id": userID},
				bson.M{"$push": bson.M{"movie_rankings": ranking}},
			)
		}
	}

	return err
}

// SaveBattle saves a battle result to the database
func (r *BattleRepository) SaveBattle(ctx context.Context, battle *models.Battle) error {
	_, err := r.db.Collection("battles").InsertOne(ctx, battle)
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
