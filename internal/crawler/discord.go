package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bradykim7/gbot/internal/models"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bradykim7/gbot/pkg/config"
	"github.com/bradykim7/gbot/pkg/logger"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// DiscordClient is a simple client for sending messages to Discord
type DiscordClient struct {
	token     string
	channelID string
	log       *logger.Logger
	client    *http.Client
}

// NewDiscordClient creates a new Discord client
func NewDiscordClient(token, channelID string) (*DiscordClient, error) {
	return &DiscordClient{
		token:     token,
		channelID: channelID,
		log:       logger.New("discord_client"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// SendMessage sends a message to Discord
func (c *DiscordClient) SendMessage(content string) error {
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", c.channelID)
	
	// Create payload
	payload := map[string]string{
		"content": content,
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	
	// Set headers
	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}
	
	c.log.Info("Message sent successfully to Discord")
	return nil
}

// DiscordNotifier handles sending notifications to Discord channels using the discordgo library
type DiscordNotifier struct {
	session     *discordgo.Session
	config      *config.Config
	db          *storage.MongoDB
	logger      *zap.Logger
	rateLimiter *time.Ticker
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(cfg *config.Config, db *storage.MongoDB, log *zap.Logger) (*DiscordNotifier, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Set up rate limiter to avoid Discord API limits (1 message per 2 seconds)
	rateLimiter := time.NewTicker(2 * time.Second)

	return &DiscordNotifier{
		session:     session,
		config:      cfg,
		db:          db,
		logger:      log.Named("discord-notifier"),
		rateLimiter: rateLimiter,
	}, nil
}

// SendProductNotifications sends product notifications to users who have matching alerts
func (n *DiscordNotifier) SendProductNotifications(ctx context.Context, products []models.Product) error {
	if len(products) == 0 {
		n.logger.Info("No products to notify about")
		return nil
	}

	n.logger.Info("Processing products for notifications", zap.Int("count", len(products)))

	// Get all active alerts
	collection := n.db.Collection("keyword_alerts")
	cursor, err := collection.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return fmt.Errorf("failed to retrieve active alerts: %w", err)
	}
	defer cursor.Close(ctx)

	var alerts []models.KeywordAlert
	if err := cursor.All(ctx, &alerts); err != nil {
		return fmt.Errorf("failed to decode alerts: %w", err)
	}

	n.logger.Info("Found active alerts", zap.Int("count", len(alerts)))

	// For each product, find matching alerts and send notifications
	for _, product := range products {
		// Check if we've already notified about this product
		if n.isProductNotified(ctx, product.URL) {
			n.logger.Debug("Product already notified", zap.String("url", product.URL))
			continue
		}

		// Find matching alerts
		matchingAlerts := n.findMatchingAlerts(product, alerts)
		
		if len(matchingAlerts) > 0 {
			n.logger.Info("Found matching alerts for product", 
				zap.String("title", product.Title), 
				zap.Int("matches", len(matchingAlerts)))

			// Create notification embed
			embed := n.createProductEmbed(product, matchingAlerts)

			// Send notification to each unique channel
			sentChannels := make(map[string]bool)
			
			for _, alert := range matchingAlerts {
				if !sentChannels[alert.ChannelID] {
					// Wait for rate limiter to avoid rate limits
					<-n.rateLimiter.C
					
					_, err := n.session.ChannelMessageSendEmbed(alert.ChannelID, embed)
					if err != nil {
						n.logger.Error("Failed to send Discord message", 
							zap.Error(err), 
							zap.String("channel_id", alert.ChannelID))
						continue
					}
					
					n.logger.Info("Sent notification", 
						zap.String("channel_id", alert.ChannelID),
						zap.String("product", product.Title))
					
					sentChannels[alert.ChannelID] = true
				}
			}

			// Mark product as notified
			n.markProductNotified(ctx, product)
		}
	}

	return nil
}

// findMatchingAlerts returns alerts that match the given product
func (n *DiscordNotifier) findMatchingAlerts(product models.Product, alerts []models.KeywordAlert) []models.KeywordAlert {
	var matches []models.KeywordAlert
	
	for _, alert := range alerts {
		// Check if keyword is in title (case insensitive)
		if strings.Contains(strings.ToLower(product.Title), strings.ToLower(alert.Keyword)) {
			matches = append(matches, alert)
		}
	}
	
	return matches
}

// createProductEmbed creates a rich embed for product notification
func (n *DiscordNotifier) createProductEmbed(product models.Product, alerts []models.KeywordAlert) *discordgo.MessageEmbed {
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
		if !mentionedUsers[alert.Username] {
			usernames = append(usernames, "@"+alert.Username)
			mentionedUsers[alert.Username] = true
		}
	}

	// Create fields
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Source",
			Value:  product.Source,
			Inline: true,
		},
	}

	// Add price field if available
	if product.PriceString != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Price",
			Value:  fmt.Sprintf("%s원", product.PriceString),
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
	description := fmt.Sprintf("Found a deal matching your alert! %s", strings.Join(usernames, " "))

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       product.Title,
		URL:         product.URL,
		Description: description,
		Color:       0x00ff00, // Green color
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Crawled at %s", product.CrawledAt.Format("2006-01-02 15:04:05")),
		},
	}

	return embed
}

// isProductNotified checks if a product has already been notified
func (n *DiscordNotifier) isProductNotified(ctx context.Context, url string) bool {
	collection := n.db.Collection("notified_products")
	
	count, err := collection.CountDocuments(ctx, bson.M{"url": url})
	if err != nil {
		n.logger.Error("Failed to check if product was notified", zap.Error(err), zap.String("url", url))
		return false
	}
	
	return count > 0
}

// markProductNotified marks a product as notified in the database
func (n *DiscordNotifier) markProductNotified(ctx context.Context, product models.Product) {
	collection := n.db.Collection("notified_products")
	
	notifiedProduct := struct {
		URL       string    `bson:"url"`
		Title     string    `bson:"title"`
		NotifiedAt time.Time `bson:"notified_at"`
	}{
		URL:       product.URL,
		Title:     product.Title,
		NotifiedAt: time.Now(),
	}
	
	_, err := collection.InsertOne(ctx, notifiedProduct)
	if err != nil {
		n.logger.Error("Failed to mark product as notified", zap.Error(err), zap.String("url", product.URL))
	}
}

// Close cleans up resources
func (n *DiscordNotifier) Close() {
	n.rateLimiter.Stop()
	if n.session != nil {
		n.session.Close()
	}
}