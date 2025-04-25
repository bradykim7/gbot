package storage

import (
	"context"
	"encoding/binary"
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
	
	// Get a random index using a cryptographically secure random number
	// to avoid potential bias in the selection
	randomIndexInt64, err := secureRandomInt64(count)
	if err != nil {
		// Fall back to the standard random if crypto random fails
		randomIndexInt64 = r.random.Int63n(count)
	}
	
	// Find the document at that index
	opts := options.Find().SetSkip(randomIndexInt64).SetLimit(1)
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

// secureRandomInt64 generates a cryptographically secure random number in range [0, max)
func secureRandomInt64(max int64) (int64, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	// Calculate how many bits we need
	bits := max
	bits--
	bitLength := 0
	for bits > 0 {
		bits >>= 1
		bitLength++
	}

	// Calculate the mask
	mask := int64(1)<<uint(bitLength) - 1

	// Generate random numbers until we get one in range
	var randomInt64 int64
	var buf [8]byte

	for {
		_, err := rand.Read(buf[:])
		if err != nil {
			return 0, err
		}

		// Convert bytes to int64 and apply mask
		randomInt64 = int64(binary.BigEndian.Uint64(buf[:])) & mask

		// Check if the number is in range
		if randomInt64 < max {
			return randomInt64, nil
		}
	}
}