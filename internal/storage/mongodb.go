package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	"github.com/bradykim7/gbot/pkg/config"
)

// MongoDB represents a MongoDB connection
type MongoDB struct {
	client *mongo.Client
	db     *mongo.Database
	log    *zap.Logger
	cfg    *config.Config
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(cfg *config.Config) (*MongoDB, error) {
	// Create logger
	log, _ := zap.NewProduction()
	logger := log.Named("mongodb")
	
	// Determine default URI
	uri := cfg.MongoDBURI
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	
	// Ping the database
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	
	logger.Info("Connected to MongoDB", zap.String("uri", uri))
	
	// Use default database
	dbName := "discord_bot"
	if cfg.IsDevelopment {
		dbName = "discord_bot_dev"
	}
	
	return &MongoDB{
		client: client,
		db:     client.Database(dbName),
		log:    logger,
		cfg:    cfg,
	}, nil
}

// Disconnect closes the MongoDB connection
func (m *MongoDB) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	m.log.Info("Closing MongoDB connection")
	return m.client.Disconnect(ctx)
}

// Collection returns a MongoDB collection
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.db.Collection(name)
}

// Client returns the MongoDB client
func (m *MongoDB) Client() *mongo.Client {
	return m.client
}

// Database returns the current database
func (m *MongoDB) Database() *mongo.Database {
	return m.db
}

// SetDatabase changes the current database
func (m *MongoDB) SetDatabase(name string) {
	m.log.Info("Switching database", zap.String("database", name))
	m.db = m.client.Database(name)
}

// UseWebcrawlerDatabase switches to the webcrawler database
func (m *MongoDB) UseWebcrawlerDatabase() {
	// Use webcrawler-specific URI if available
	if m.cfg.MongoDBURIWebcrawler != "" {
		// Create a new connection to the webcrawler database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// Connect to MongoDB
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(m.cfg.MongoDBURIWebcrawler))
		if err != nil {
			m.log.Error("Failed to connect to webcrawler MongoDB, using default instead", 
				zap.Error(err))
		} else {
			// Successful connection, update client and database
			m.client = client
			m.db = client.Database("webcrawler")
			m.log.Info("Connected to webcrawler MongoDB", 
				zap.String("uri", m.cfg.MongoDBURIWebcrawler))
			return
		}
	}
	
	// Fallback to using the default client with a different database
	m.log.Info("Using webcrawler database with default connection")
	m.db = m.client.Database("webcrawler")
}