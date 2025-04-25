package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FoodType은 음식 유형을 나타냅니다
type FoodType string

const (
	// FoodTypeLunch는 점심 음식을 나타냅니다
	FoodTypeLunch FoodType = "lunch"

	// FoodTypeDinner는 저녁 음식을 나타냅니다
	FoodTypeDinner FoodType = "dinner"
)

// Food는 음식 추천을 나타냅니다
type Food struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	FoodType  FoodType           `bson:"food_type" json:"food_type"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	CreatedBy string             `bson:"created_by" json:"created_by"`
	IsActive  bool               `bson:"is_active" json:"is_active"`
}

// NewFood는 새로운 음식을 생성합니다
func NewFood(name string, foodType FoodType, createdBy string) *Food {
	return &Food{
		Name:      name,
		FoodType:  foodType,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
		IsActive:  true,
	}
}