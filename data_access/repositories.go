package data_access

import (
	"context"
	"movie-vs-backend/models"

	"go.mongodb.org/mongo-driver/bson"
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

// SubmitBattle updates the winner of a battle
func (r *BattleRepository) SubmitBattle(ctx context.Context, battle *models.Battle) error {

	if battle.Winner.Title == battle.MovieA.Title {

	} else {

	}

	_, err := r.db.Collection("battles").UpdateOne(ctx, filter, update)
	return err
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
