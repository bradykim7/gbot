package crawler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bradykim7/gbot/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// AlertMatcher handles matching products with user alerts
type AlertMatcher struct {
	logger *zap.Logger
	db     *MongoDB
}

// NewAlertMatcher creates a new AlertMatcher
func NewAlertMatcher(db *MongoDB, logger *zap.Logger) *AlertMatcher {
	return &AlertMatcher{
		logger: logger.Named("alert-matcher"),
		db:     db,
	}
}

// FindMatchingAlerts finds all alerts matching the given product
func (m *AlertMatcher) FindMatchingAlerts(ctx context.Context, product models.Product) ([]models.KeywordAlert, error) {
	// Get all active alerts
	collection := m.db.Collection("keyword_alerts")
	filter := bson.M{"is_active": true}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active alerts: %w", err)
	}
	defer cursor.Close(ctx)

	var alerts []models.KeywordAlert
	if err := cursor.All(ctx, &alerts); err != nil {
		return nil, fmt.Errorf("failed to decode alerts: %w", err)
	}

	m.logger.Debug("Retrieved active alerts", zap.Int("count", len(alerts)))

	// Find matching alerts
	var matches []models.KeywordAlert
	var matchedKeywords []string

	// Create a normalized product title for case-insensitive search
	normalizedTitle := strings.ToLower(product.Title)
	
	// Include other searchable fields
	searchText := normalizedTitle
	if product.Product != "" && product.Product != product.Title {
		searchText += " " + strings.ToLower(product.Product)
	}
	if product.Category != "" {
		searchText += " " + strings.ToLower(product.Category)
	}

	// Check each alert against the search text
	for _, alert := range alerts {
		// Normalize keyword for case-insensitive comparison
		keyword := strings.ToLower(alert.Keyword)
		
		// Check if keyword is in search text
		if strings.Contains(searchText, keyword) {
			matches = append(matches, alert)
			matchedKeywords = append(matchedKeywords, alert.Keyword)
			
			// Update the alert's last notification time
			if err := m.updateAlertNotification(ctx, alert.ID); err != nil {
				m.logger.Warn("Failed to update alert notification metadata",
					zap.Error(err),
					zap.String("alert_id", alert.ID))
			}
		}
	}
	
	// Update product with matched keywords
	if len(matchedKeywords) > 0 {
		productCollection := m.db.Collection("products")
		if product.ID != "" {
			_, err = productCollection.UpdateByID(ctx, 
				product.ID, 
				bson.M{"$set": bson.M{"keywords": matchedKeywords}})
			
			if err != nil {
				m.logger.Warn("Failed to update product keywords",
					zap.Error(err),
					zap.String("product_id", product.ID))
			}
		}
	}
	
	m.logger.Info("Found matching alerts",
		zap.Int("matches", len(matches)),
		zap.Strings("keywords", matchedKeywords),
		zap.String("product_title", product.Title))
		
	return matches, nil
}

// UpdateAlertNotification updates the alert's notification metadata
func (m *AlertMatcher) updateAlertNotification(ctx context.Context, alertID string) error {
	// Skip if ID is empty
	if alertID == "" {
		return nil
	}
	
	collection := m.db.Collection("keyword_alerts")
	
	// Convert string ID to ObjectID if needed
	var objectID interface{}
	if primitive.IsValidObjectID(alertID) {
		objID, _ := primitive.ObjectIDFromHex(alertID)
		objectID = objID
	} else {
		objectID = alertID
	}
	
	// Update LastNotified and increment NotifyCount
	update := bson.M{
		"$set": bson.M{
			"last_notified": time.Now().Unix(),
		},
		"$inc": bson.M{
			"notify_count": 1,
		},
	}
	
	_, err := collection.UpdateByID(ctx, objectID, update)
	if err != nil {
		return fmt.Errorf("failed to update alert notification: %w", err)
	}
	
	return nil
}

// GetAlertsByUser retrieves all active alerts for the specified user
func (m *AlertMatcher) GetAlertsByUser(ctx context.Context, userID string) ([]models.KeywordAlert, error) {
	collection := m.db.Collection("keyword_alerts")
	filter := bson.M{
		"user_id": userID,
		"is_active": true,
	}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts for user: %w", err)
	}
	defer cursor.Close(ctx)
	
	var alerts []models.KeywordAlert
	if err := cursor.All(ctx, &alerts); err != nil {
		return nil, fmt.Errorf("failed to decode user alerts: %w", err)
	}
	
	return alerts, nil
}

// GetPopularAlerts returns the most commonly triggered alerts
func (m *AlertMatcher) GetPopularAlerts(ctx context.Context, limit int) ([]models.KeywordAlert, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	
	collection := m.db.Collection("keyword_alerts")
	
	// Find alerts with the highest notify_count
	opts := options.Find().
		SetSort(bson.D{{"notify_count", -1}}).
		SetLimit(int64(limit))
	
	cursor, err := collection.Find(ctx, 
		bson.M{"is_active": true, "notify_count": bson.M{"$gt": 0}},
		opts)
	
	if err != nil {
		return nil, fmt.Errorf("failed to find popular alerts: %w", err)
	}
	defer cursor.Close(ctx)
	
	var alerts []models.KeywordAlert
	if err := cursor.All(ctx, &alerts); err != nil {
		return nil, fmt.Errorf("failed to decode popular alerts: %w", err)
	}
	
	return alerts, nil
}