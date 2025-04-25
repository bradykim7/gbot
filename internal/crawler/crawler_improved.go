package crawler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bradykim7/gbot/internal/crawler/sources"
	"github.com/bradykim7/gbot/internal/models"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bradykim7/gbot/pkg/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// ImprovedCrawler is the main crawler for the application
type ImprovedCrawler struct {
	config       *config.Config
	log          *zap.Logger
	db           *storage.MongoDB
	notifier     *NotificationService
	sources      []sources.Source
	healthStatus map[string]bool
	lastRun      time.Time
	stats        CrawlerStats
	statsMutex   sync.RWMutex
}

// CrawlerStats tracks statistics about crawler operation
type CrawlerStats struct {
	TotalProducts      int       `json:"total_products"`
	NewProducts        int       `json:"new_products"`
	NotifiedProducts   int       `json:"notified_products"`
	LastRun            time.Time `json:"last_run"`
	RunCount           int       `json:"run_count"`
	LastError          string    `json:"last_error,omitempty"`
	SourceStats        map[string]SourceStats `json:"source_stats"`
}

// SourceStats tracks statistics for individual sources
type SourceStats struct {
	ProductsFound   int       `json:"products_found"`
	LastRun         time.Time `json:"last_run"`
	LastRunDuration string    `json:"last_run_duration"`
	LastError       string    `json:"last_error,omitempty"`
	SuccessRate     float64   `json:"success_rate"` // 0-1
}

// NewImprovedCrawler creates a new crawler instance
func NewImprovedCrawler(cfg *config.Config, log *zap.Logger) (*ImprovedCrawler, error) {
	// Connect to MongoDB
	db, err := storage.NewMongoDB(cfg)
	if err != nil {
		return nil, err
	}
	
	// Create notification service
	notifier, err := NewNotificationService(cfg, db, log)
	if err != nil {
		return nil, err
	}
	
	// Initialize sources
	ppomppu := sources.NewPpomppuCrawler(log)
	
	// TODO: Add other sources
	// quasarzone := sources.NewQuasarzoneCrawler(log)
	
	// Create crawler
	crawler := &ImprovedCrawler{
		config:   cfg,
		log:      log.Named("improved-crawler"),
		db:       db,
		notifier: notifier,
		sources: []sources.Source{
			ppomppu,
			// quasarzone,
		},
		healthStatus: make(map[string]bool),
		stats: CrawlerStats{
			SourceStats: make(map[string]SourceStats),
		},
	}
	
	// Initialize database indices
	if err := crawler.setupDatabaseIndices(context.Background()); err != nil {
		log.Warn("Failed to set up database indices", zap.Error(err))
	}
	
	return crawler, nil
}

// setupDatabaseIndices ensures necessary database indices exist for performance
func (c *ImprovedCrawler) setupDatabaseIndices(ctx context.Context) error {
	// Products collection indices
	productsCollection := c.db.Collection("products")
	
	// URL index (must be unique)
	_, err := productsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"url", 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		c.log.Warn("Failed to create URL index on products collection", zap.Error(err))
	}
	
	// Title text index for searching
	_, err = productsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"title", "text"}, {"product", "text"}},
	})
	if err != nil {
		c.log.Warn("Failed to create text index on products collection", zap.Error(err))
	}
	
	// Keyword alerts collection indices
	alertsCollection := c.db.Collection("keyword_alerts")
	
	// User ID + Keyword compound index (must be unique per user)
	_, err = alertsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"user_id", 1}, {"keyword", 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		c.log.Warn("Failed to create compound index on keyword_alerts collection", zap.Error(err))
	}
	
	// Notified products collection indices
	notifiedCollection := c.db.Collection("notified_products")
	
	// URL index (must be unique)
	_, err = notifiedCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"url", 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		c.log.Warn("Failed to create URL index on notified_products collection", zap.Error(err))
	}
	
	return nil
}

// Run executes a single crawl of all sources
func (c *ImprovedCrawler) Run(ctx context.Context) error {
	c.log.Info("Starting crawler run")
	
	// Update stats
	c.statsMutex.Lock()
	c.stats.LastRun = time.Now()
	c.stats.RunCount++
	c.stats.NewProducts = 0 // Reset for this run
	c.stats.NotifiedProducts = 0 // Reset for this run
	c.statsMutex.Unlock()
	
	startTime := time.Now()
	
	// Create channels for product collection and processing
	productChan := make(chan models.Product, 1000)
	errorChan := make(chan error, len(c.sources))
	
	// Create WaitGroup for source crawlers
	var wg sync.WaitGroup
	wg.Add(len(c.sources))
	
	// Crawl all sources in parallel
	for _, src := range c.sources {
		go func(source sources.Source) {
			defer wg.Done()
			
			sourceName := source.Name()
			sourceStartTime := time.Now()
			
			c.log.Info("Crawling source", zap.String("source", sourceName))
			
			products, err := source.Crawl(ctx)
			if err != nil {
				c.log.Error("Failed to crawl source", 
					zap.String("source", sourceName), 
					zap.Error(err))
				
				// Update source stats with error
				c.statsMutex.Lock()
				sourceStats := c.stats.SourceStats[sourceName]
				sourceStats.LastError = err.Error()
				sourceStats.LastRun = time.Now()
				sourceStats.LastRunDuration = time.Since(sourceStartTime).String()
				
				// Calculate success rate
				if sourceStats.SuccessRate == 0 {
					sourceStats.SuccessRate = 0 // First run failed
				} else {
					// Weight previous success rate at 90%, new result at 10%
					sourceStats.SuccessRate = sourceStats.SuccessRate*0.9 + 0*0.1
				}
				
				c.stats.SourceStats[sourceName] = sourceStats
				c.statsMutex.Unlock()
				
				errorChan <- fmt.Errorf("failed to crawl source %s: %w", sourceName, err)
				return
			}
			
			c.log.Info("Crawled source successfully", 
				zap.String("source", sourceName), 
				zap.Int("products_found", len(products)))
			
			// Update source stats with success
			c.statsMutex.Lock()
			sourceStats := c.stats.SourceStats[sourceName]
			sourceStats.ProductsFound = len(products)
			sourceStats.LastRun = time.Now()
			sourceStats.LastRunDuration = time.Since(sourceStartTime).String()
			sourceStats.LastError = "" // Clear any previous error
			
			// Calculate success rate
			if sourceStats.SuccessRate == 0 {
				sourceStats.SuccessRate = 1 // First run succeeded
			} else {
				// Weight previous success rate at 90%, new result at 10%
				sourceStats.SuccessRate = sourceStats.SuccessRate*0.9 + 1*0.1
			}
			
			c.stats.SourceStats[sourceName] = sourceStats
			c.stats.TotalProducts += len(products) // Update total found
			c.statsMutex.Unlock()
			
			// Update health status for this source
			c.healthStatus[sourceName] = true
			
			// Send products to channel
			for _, product := range products {
				// Set source and crawled time if not already set
				if product.Source == "" {
					product.Source = sourceName
				}
				if product.CrawledAt.IsZero() {
					product.CrawledAt = time.Now()
				}
				
				select {
				case productChan <- product:
					// Successfully sent product to channel
				case <-ctx.Done():
					// Context cancelled, stop sending
					errorChan <- ctx.Err()
					return
				}
			}
		}(src)
	}
	
	// Close channels when all sources are done
	go func() {
		wg.Wait()
		close(productChan)
		close(errorChan)
	}()
	
	// Process products
	var newProducts []models.Product
	
	// Get MongoDB collection
	collection := c.db.Collection("products")
	
	// Process each product
	for product := range productChan {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Check if product already exists
			var count int64
			filter := bson.M{
				"url": product.URL,
			}
			
			count, err := collection.CountDocuments(ctx, filter)
			if err != nil {
				c.log.Error("Failed to check product existence", 
					zap.Error(err), 
					zap.String("url", product.URL))
				continue
			}
			
			if count > 0 {
				c.log.Debug("Product already exists", zap.String("url", product.URL))
				continue
			}
			
			// Generate ID if not set
			if product.ID == "" {
				product.ID = primitive.NewObjectID().Hex()
			}
			
			// Insert new product
			_, err = collection.InsertOne(ctx, product)
			if err != nil {
				c.log.Error("Failed to insert product", 
					zap.Error(err), 
					zap.String("title", product.Title))
				continue
			}
			
			c.log.Info("New product found", 
				zap.String("title", product.Title), 
				zap.String("source", product.Source))
			
			newProducts = append(newProducts, product)
			
			// Update stats
			c.statsMutex.Lock()
			c.stats.NewProducts++
			c.statsMutex.Unlock()
		}
	}
	
	// Check for errors from goroutines
	var crawlErrors []error
	for err := range errorChan {
		crawlErrors = append(crawlErrors, err)
	}
	
	// Send notifications for new products
	if len(newProducts) > 0 {
		c.log.Info("Sending notifications for new products", zap.Int("count", len(newProducts)))
		
		if err := c.notifier.NotifyNewProducts(ctx, newProducts); err != nil {
			c.log.Error("Failed to send some notifications", zap.Error(err))
			
			// Update stats with error
			c.statsMutex.Lock()
			c.stats.LastError = err.Error()
			c.statsMutex.Unlock()
			
			crawlErrors = append(crawlErrors, err)
		} else {
			// Update stats with notification count
			c.statsMutex.Lock()
			c.stats.NotifiedProducts = len(newProducts)
			c.statsMutex.Unlock()
		}
	}
	
	// Update last run time
	c.lastRun = time.Now()
	
	// Log duration
	duration := time.Since(startTime)
	c.log.Info("Crawler run completed", 
		zap.Int("new_products", len(newProducts)),
		zap.Int("total_products", len(productChan)),
		zap.Duration("duration", duration))
	
	// Return any errors
	if len(crawlErrors) > 0 {
		// Format as a single error with all the error messages
		errorMsg := fmt.Sprintf("%d error(s) during crawl: ", len(crawlErrors))
		for i, err := range crawlErrors {
			if i > 0 {
				errorMsg += "; "
			}
			errorMsg += err.Error()
		}
		return fmt.Errorf("%s", errorMsg)
	}
	
	return nil
}

// StartScheduledRuns starts periodic crawler runs
func (c *ImprovedCrawler) StartScheduledRuns(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	c.log.Info("Starting scheduled crawler runs", zap.Duration("interval", interval))
	
	// Run immediately on startup
	if err := c.Run(ctx); err != nil {
		c.log.Error("Initial crawler run failed", zap.Error(err))
		
		// Store error in stats
		c.statsMutex.Lock()
		c.stats.LastError = err.Error()
		c.statsMutex.Unlock()
	}
	
	// Then run on schedule
	for {
		select {
		case <-ticker.C:
			if err := c.Run(ctx); err != nil {
				c.log.Error("Scheduled crawler run failed", zap.Error(err))
				
				// Store error in stats
				c.statsMutex.Lock()
				c.stats.LastError = err.Error()
				c.statsMutex.Unlock()
			}
		case <-ctx.Done():
			c.log.Info("Stopping scheduled crawler runs")
			return
		}
	}
}

// GetStats returns current crawler statistics
func (c *ImprovedCrawler) GetStats() CrawlerStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	
	// Return a copy of the stats
	statsCopy := c.stats
	return statsCopy
}

// Health returns the health status of the crawler
func (c *ImprovedCrawler) Health() map[string]bool {
	return c.healthStatus
}

// Close cleans up resources
func (c *ImprovedCrawler) Close() error {
	// Close the notifier
	c.notifier.Close()
	
	// Disconnect from MongoDB
	if err := c.db.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}
	
	return nil
}