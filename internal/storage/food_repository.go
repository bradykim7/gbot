package storage

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/bradykim7/gbot/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// FoodRepository handles persistence for food recommendations
type FoodRepository struct {
	db     *MongoDB
	log    *zap.Logger
	random *rand.Rand
}

// NewFoodRepository creates a new food repository
func NewFoodRepository(db *MongoDB, log *zap.Logger) *FoodRepository {
	// Initialize random source with time-based seed
	source := rand.NewSource(time.Now().UnixNano())
	
	return &FoodRepository{
		db:     db,
		log:    log.Named("food-repository"),
		random: rand.New(source),
	}
}

// GetRandomFood returns a random food of the given type
func (r *FoodRepository) GetRandomFood(ctx context.Context, foodType models.FoodType) (*models.Food, error) {
	// Get collection
	collection := r.db.Collection("foods")
	
	// Create filter for active foods of given type
	filter := bson.M{
		"food_type": foodType,
		"is_active": true,
	}
	
	// Count documents matching filter
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count foods: %w", err)
	}
	
	if count == 0 {
		return nil, fmt.Errorf("no foods found for type %s", foodType)
	}
	
	// Get a random index
	randomIndex := r.random.Int63n(count)
	
	// Find the document at that index
	opts := options.Find().SetSkip(randomIndex).SetLimit(1)
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find food: %w", err)
	}
	defer cursor.Close(ctx)
	
	// Get the food
	var food models.Food
	if cursor.Next(ctx) {
		if err := cursor.Decode(&food); err != nil {
			return nil, fmt.Errorf("failed to decode food: %w", err)
		}
		return &food, nil
	}
	
	return nil, fmt.Errorf("no food found at random index")
}

// GetAllFoods returns all foods of the given type
func (r *FoodRepository) GetAllFoods(ctx context.Context, foodType models.FoodType) ([]models.Food, error) {
	// Get collection
	collection := r.db.Collection("foods")
	
	// Create filter for active foods of given type
	filter := bson.M{
		"food_type": foodType,
		"is_active": true,
	}
	
	// Find all matching foods
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find foods: %w", err)
	}
	defer cursor.Close(ctx)
	
	// Decode foods
	var foods []models.Food
	if err := cursor.All(ctx, &foods); err != nil {
		return nil, fmt.Errorf("failed to decode foods: %w", err)
	}
	
	return foods, nil
}

// SaveFood saves a food to the database
func (r *FoodRepository) SaveFood(ctx context.Context, food *models.Food) error {
	// Get collection
	collection := r.db.Collection("foods")
	
	// Check if food already exists
	filter := bson.M{
		"name":      food.Name,
		"food_type": food.FoodType,
	}
	
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check food existence: %w", err)
	}
	
	if count > 0 {
		return fmt.Errorf("food already exists")
	}
	
	// Insert the food
	_, err = collection.InsertOne(ctx, food)
	if err != nil {
		return fmt.Errorf("failed to insert food: %w", err)
	}
	
	r.log.Info("Food saved",
		zap.String("name", food.Name),
		zap.String("type", string(food.FoodType)),
		zap.String("created_by", food.CreatedBy))
	
	return nil
}

// DeleteFood marks a food as inactive
func (r *FoodRepository) DeleteFood(ctx context.Context, name string, foodType models.FoodType) error {
	// Get collection
	collection := r.db.Collection("foods")
	
	// Create filter
	filter := bson.M{
		"name":      name,
		"food_type": foodType,
	}
	
	// Update the document to set is_active to false
	update := bson.M{
		"$set": bson.M{
			"is_active": false,
		},
	}
	
	// Update the document
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update food: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("food not found")
	}
	
	r.log.Info("Food deleted",
		zap.String("name", name),
		zap.String("type", string(foodType)))
	
	return nil
}

// CountFoods counts the number of foods of the given type
func (r *FoodRepository) CountFoods(ctx context.Context, foodType models.FoodType) (int64, error) {
	// Get collection
	collection := r.db.Collection("foods")
	
	// Create filter
	filter := bson.M{
		"food_type": foodType,
		"is_active": true,
	}
	
	// Count documents
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count foods: %w", err)
	}
	
	return count, nil
}

// SearchFoods searches for foods by name
func (r *FoodRepository) SearchFoods(ctx context.Context, query string, foodType models.FoodType) ([]models.Food, error) {
	// Get collection
	collection := r.db.Collection("foods")
	
	// Create filter for active foods with name containing query
	filter := bson.M{
		"name": bson.M{
			"$regex":   query,
			"$options": "i", // case-insensitive
		},
		"food_type": foodType,
		"is_active": true,
	}
	
	// Find all matching foods
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find foods: %w", err)
	}
	defer cursor.Close(ctx)
	
	// Decode foods
	var foods []models.Food
	if err := cursor.All(ctx, &foods); err != nil {
		return nil, fmt.Errorf("failed to decode foods: %w", err)
	}
	
	return foods, nil
}