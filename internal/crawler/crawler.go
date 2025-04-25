package crawler

import (
	"context"
	"sync"
	"time"

	"github.com/bradykim7/gbot/internal/crawler/sources"
	"github.com/bradykim7/gbot/internal/models"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bradykim7/gbot/pkg/config"
	"go.uber.org/zap"
)

// SourceInterface represents a source that can be crawled
type SourceInterface interface {
	Crawl(ctx context.Context) ([]models.Product, error)
	Name() string
}

// Crawler represents the web crawler
type Crawler struct {
	config    *config.Config
	log       *zap.Logger
	db        *storage.MongoDB
	sources   []SourceInterface
	notifier  *DiscordNotifier
	client    *DiscordClient  // Legacy client for backward compatibility
}

// New creates a new crawler instance
func New(cfg *config.Config, log *zap.Logger) (*Crawler, error) {
	// Connect to MongoDB
	db, err := storage.NewMongoDB(cfg)
	if err != nil {
		return nil, err
	}
	
	// Create Discord notifier
	notifier, err := NewDiscordNotifier(cfg, db, log)
	if err != nil {
		return nil, err
	}
	
	// Create legacy Discord client
	client, err := NewDiscordClient(cfg.DiscordToken, cfg.ProductChannelID)
	if err != nil {
		return nil, err
	}
	
	// Create sources
	ppomppu := sources.NewPpomppuCrawler(log)
	
	// TODO: Implement other sources
	// quasarzone := sources.NewQuasarzoneCrawler(log)
	
	// Create crawler
	crawler := &Crawler{
		config:    cfg,
		log:       log.Named("crawler"),
		db:        db,
		notifier:  notifier,
		client:    client,
		sources: []SourceInterface{
			ppomppu,
			// quasarzone,
		},
	}
	
	return crawler, nil
}

// Run runs the crawler once
func (c *Crawler) Run(ctx context.Context) error {
	c.log.Info("Starting crawler run")
	
	// Create WaitGroup for parallelization
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	// Channel for products
	productChan := make(chan models.Product, 100)
	
	// Crawl all sources in parallel
	for _, src := range c.sources {
		wg.Add(1)
		go func(source SourceInterface) {
			defer wg.Done()
			
			c.log.Info("Crawling source", zap.String("source", source.Name()))
			products, err := source.Crawl(ctx)
			if err != nil {
				c.log.Error("Failed to crawl source", 
					zap.String("source", source.Name()), 
					zap.Error(err))
				return
			}
			
			c.log.Info("Crawled source successfully", 
				zap.String("source", source.Name()), 
				zap.Int("products", len(products)))
			
			// Send products to channel
			mu.Lock()
			for _, product := range products {
				productChan <- product
			}
			mu.Unlock()
			
		}(src)
	}
	
	// Close channel when all sources are done
	go func() {
		wg.Wait()
		close(productChan)
	}()
	
	// Process products
	var newProducts []models.Product
	
	// Create product collection
	collection := c.db.Collection("products")
	
	// Process each product
	for product := range productChan {
		// Check if product already exists
		var count int64
		filter := map[string]interface{}{
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
	}
	
	// Send notifications for new products
	if len(newProducts) > 0 {
		c.log.Info("Sending notifications for new products", zap.Int("count", len(newProducts)))
		
		if err := c.notifier.SendProductNotifications(ctx, newProducts); err != nil {
			c.log.Error("Failed to send notifications", zap.Error(err))
		}
	}
	
	c.log.Info("Crawler run completed", zap.Int("new_products", len(newProducts)))
	return nil
}

// StartScheduledRuns starts periodic crawler runs
func (c *Crawler) StartScheduledRuns(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	c.log.Info("Starting scheduled crawler runs", zap.Duration("interval", interval))
	
	// Run immediately
	if err := c.Run(ctx); err != nil {
		c.log.Error("Initial crawler run failed", zap.Error(err))
	}
	
	// Then run on schedule
	for {
		select {
		case <-ticker.C:
			if err := c.Run(ctx); err != nil {
				c.log.Error("Scheduled crawler run failed", zap.Error(err))
			}
		case <-ctx.Done():
			c.log.Info("Stopping scheduled crawler runs")
			return
		}
	}
}

// Close cleans up resources
func (c *Crawler) Close() error {
	c.notifier.Close()
	
	if err := c.db.Disconnect(); err != nil {
		return err
	}
	
	return nil
}