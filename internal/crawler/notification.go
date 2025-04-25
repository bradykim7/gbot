package crawler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bradykim7/gbot/internal/models"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bradykim7/gbot/pkg/config"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// NotificationService handles sending notifications to Discord users
type NotificationService struct {
	session     *discordgo.Session
	config      *config.Config
	db          *storage.MongoDB
	logger      *zap.Logger
	rateLimiter *time.Ticker
	alertMatcher *AlertMatcher
}

// NewNotificationService creates a new notification service
func NewNotificationService(cfg *config.Config, db *storage.MongoDB, log *zap.Logger) (*NotificationService, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Set up rate limiter to avoid Discord API limits (1 message per 2 seconds)
	rateLimiter := time.NewTicker(2 * time.Second)
	
	alertMatcher := NewAlertMatcher(db, log)

	return &NotificationService{
		session:      session,
		config:       cfg,
		db:           db,
		logger:       log.Named("notification-service"),
		rateLimiter:  rateLimiter,
		alertMatcher: alertMatcher,
	}, nil
}

// NotifyNewProducts sends notifications for newly found products
func (n *NotificationService) NotifyNewProducts(ctx context.Context, products []models.Product) error {
	if len(products) == 0 {
		n.logger.Info("No products to notify about")
		return nil
	}

	n.logger.Info("Processing products for notifications", zap.Int("count", len(products)))

	// Process each product
	var wg sync.WaitGroup
	wg.Add(len(products))
	
	// Create semaphore to limit concurrency to 5 at a time
	sem := make(chan struct{}, 5)
	
	// Collect errors
	var notificationErrors []error
	var errorMutex sync.Mutex
	
	for _, product := range products {
		// Skip products that were already notified
		if n.isProductNotified(ctx, product.URL) {
			n.logger.Debug("Product already notified", zap.String("url", product.URL))
			wg.Done()
			continue
		}
		
		// Process each product concurrently but with controlled concurrency
		sem <- struct{}{} // Acquire semaphore
		
		go func(p models.Product) {
			defer func() {
				<-sem // Release semaphore
				wg.Done()
			}()
			
			// Find matching alerts
			matchingAlerts, err := n.alertMatcher.FindMatchingAlerts(ctx, p)
			if err != nil {
				errorMutex.Lock()
				notificationErrors = append(notificationErrors, fmt.Errorf("failed to find matching alerts: %w", err))
				errorMutex.Unlock()
				return
			}
			
			if len(matchingAlerts) == 0 {
				return // No matching alerts, nothing to notify
			}
			
			n.logger.Debug("Found matching alerts", 
				zap.String("product", p.Title), 
				zap.Int("matches", len(matchingAlerts)))
			
			// Send notifications
			err = n.sendProductNotifications(ctx, p, matchingAlerts)
			if err != nil {
				errorMutex.Lock()
				notificationErrors = append(notificationErrors, err)
				errorMutex.Unlock()
			}
		}(product)
	}

	// Wait for all notifications to finish
	wg.Wait()
	
	// If there were errors, log them and return a combined error
	if len(notificationErrors) > 0 {
		n.logger.Error("Some notifications failed", 
			zap.Int("failure_count", len(notificationErrors)), 
			zap.Int("total_products", len(products)))
		
		return fmt.Errorf("some notifications failed: %v", notificationErrors)
	}
	
	n.logger.Info("All notifications processed successfully", 
		zap.Int("product_count", len(products)))
	return nil
}

// sendProductNotifications sends notifications for a single product to all matching alert channels
func (n *NotificationService) sendProductNotifications(ctx context.Context, product models.Product, alerts []models.KeywordAlert) error {
	if len(alerts) == 0 {
		return nil
	}

	// Create notification embed
	embed := n.createProductEmbed(product, alerts)

	// Send notification to each unique channel
	sentChannels := make(map[string]bool)
	channelErrors := make(map[string]error)
	var notificationErrors []error
	
	for _, alert := range alerts {
		if !sentChannels[alert.ChannelID] {
			// Wait for rate limiter to avoid rate limits
			select {
			case <-n.rateLimiter.C:
				// Continue with sending
			case <-ctx.Done():
				// Context canceled, stop sending
				return ctx.Err()
			}
			
			_, err := n.session.ChannelMessageSendEmbed(alert.ChannelID, embed)
			if err != nil {
				n.logger.Error("Failed to send Discord message", 
					zap.Error(err), 
					zap.String("channel_id", alert.ChannelID))
				channelErrors[alert.ChannelID] = err
				notificationErrors = append(notificationErrors, fmt.Errorf("failed to send notification to channel %s: %w", alert.ChannelID, err))
				continue
			}
			
			n.logger.Info("Sent notification", 
				zap.String("channel_id", alert.ChannelID),
				zap.String("product", product.Title))
			
			sentChannels[alert.ChannelID] = true
		}
	}

	// Only mark product as notified if at least one notification was sent
	if len(sentChannels) > 0 {
		if err := n.markProductNotified(ctx, product); err != nil {
			n.logger.Error("Failed to mark product as notified", 
				zap.Error(err), 
				zap.String("product_url", product.URL))
			
			notificationErrors = append(notificationErrors, fmt.Errorf("failed to mark product as notified: %w", err))
		}
	}
	
	// If there were errors, log them and return a combined error
	if len(notificationErrors) > 0 {
		if len(notificationErrors) == len(alerts) {
			// All notifications failed
			return fmt.Errorf("all notifications failed: %v", notificationErrors)
		} else {
			// Some notifications succeeded, some failed
			n.logger.Warn("Some channel notifications failed but others succeeded",
				zap.Int("successful", len(sentChannels)),
				zap.Int("failed", len(channelErrors)))
			
			// Return a summary error but don't fail the whole process
			return fmt.Errorf("%d of %d notifications failed", len(channelErrors), len(alerts))
		}
	}
	
	return nil
}

// createProductEmbed creates a rich embed for product notification
func (n *NotificationService) createProductEmbed(product models.Product, alerts []models.KeywordAlert) *discordgo.MessageEmbed {
	// Collect unique keywords that matched
	keywords := make(map[string]bool)
	for _, alert := range alerts {
		keywords[alert.Keyword] = true
	}
	
	// Join keywords into comma-separated string
	var keywordList []string
	for k := range keywords {
		keywordList = append(keywordList, k)
	}
	
	// Collect unique usernames to mention
	var usernames []string
	mentionedUsers := make(map[string]bool)
	for _, alert := range alerts {
		// Check if Username is set
		if alert.Username != "" && !mentionedUsers[alert.Username] {
			usernames = append(usernames, "@"+alert.Username)
			mentionedUsers[alert.Username] = true
		} else if !mentionedUsers[alert.UserID] {
			// Fallback to user ID if username not available
			usernames = append(usernames, "<@"+alert.UserID+">")
			mentionedUsers[alert.UserID] = true
		}
	}

	// Create embed fields
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Source",
			Value:  product.Source,
			Inline: true,
		},
	}

	// Add price field if available
	priceStr := product.GetPriceString()
	if priceStr != "Price unknown" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Price",
			Value:  priceStr,
			Inline: true,
		})
	}
	
	// Add discount rate if available
	if product.DiscountRate > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Discount",
			Value:  fmt.Sprintf("%d%%", product.DiscountRate),
			Inline: true,
		})
	}

	// Add comments/views if available
	if product.Comments > 0 || product.Views > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Stats",
			Value:  fmt.Sprintf("Comments: %d | Views: %d", product.Comments, product.Views),
			Inline: true,
		})
	}

	// Add matched keywords field
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Matched Keywords",
		Value:  strings.Join(keywordList, ", "),
		Inline: false,
	})

	// Create description with mentions
	description := fmt.Sprintf("새로운 특가 상품을 발견했습니다! %s", strings.Join(usernames, " "))

	// Create embed color based on hotness or discount rate
	color := 0x00ff00 // Default green
	if product.IsHot {
		color = 0xff0000 // Hot item = red
	} else if product.DiscountRate >= 50 {
		color = 0xff6600 // Big discount = orange
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       product.Title,
		URL:         product.URL,
		Description: description,
		Color:       color,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Crawled at %s", product.CrawledAt.Format("2006-01-02 15:04:05")),
		},
	}
	
	// Add image if available
	if product.ImageURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: product.ImageURL,
		}
	}

	return embed
}

// isProductNotified checks if a product has already been notified
func (n *NotificationService) isProductNotified(ctx context.Context, url string) bool {
	collection := n.db.Collection("notified_products")
	
	count, err := collection.CountDocuments(ctx, bson.M{"url": url})
	if err != nil {
		n.logger.Error("Failed to check if product was notified", zap.Error(err), zap.String("url", url))
		return false
	}
	
	return count > 0
}

// markProductNotified marks a product as notified in the database
func (n *NotificationService) markProductNotified(ctx context.Context, product models.Product) error {
	collection := n.db.Collection("notified_products")
	
	notifiedProduct := struct {
		URL        string    `bson:"url"`
		Title      string    `bson:"title"`
		NotifiedAt time.Time `bson:"notified_at"`
		ProductID  string    `bson:"product_id,omitempty"`
	}{
		URL:        product.URL,
		Title:      product.Title,
		NotifiedAt: time.Now(),
		ProductID:  product.ID,
	}
	
	_, err := collection.InsertOne(ctx, notifiedProduct)
	if err != nil {
		return fmt.Errorf("failed to mark product as notified: %w", err)
	}
	
	// Also update the product's notified status if it has an ID
	if product.ID != "" {
		productCollection := n.db.Collection("products")
		_, err = productCollection.UpdateByID(ctx, 
			product.ID, 
			bson.M{"$set": bson.M{"notified": true}})
			
		if err != nil {
			n.logger.Warn("Failed to update product notification status",
				zap.Error(err),
				zap.String("product_id", product.ID))
			// Don't fail the whole process for this
		}
	}
	
	return nil
}

// Close cleans up resources
func (n *NotificationService) Close() {
	n.rateLimiter.Stop()
	if n.session != nil {
		n.session.Close()
	}
}