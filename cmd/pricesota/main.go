package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bradykim7/gbot/internal/crawler"
	"github.com/bradykim7/gbot/pkg/config"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()
	
	log := logger.Named("pricesota")
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}
	
	// Create context that will be canceled on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle graceful shutdown
	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
		<-sc
		log.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()
	
	// Initialize crawler
	webCrawler, err := crawler.New(cfg, log)
	if err != nil {
		log.Fatal("Failed to initialize crawler", zap.Error(err))
	}
	defer func() {
		if err := webCrawler.Close(); err != nil {
			log.Error("Error closing crawler", zap.Error(err))
		}
	}()
	
	// Start crawler with scheduled runs
	log.Info("Starting web crawler service")
	
	// Configure interval
	interval := time.Duration(cfg.CrawlIntervalMinutes) * time.Minute
	log.Info("Crawler configured", zap.Duration("interval", interval))
	
	// Start scheduled runs (this blocks until context is canceled)
	webCrawler.StartScheduledRuns(ctx, interval)
	
	log.Info("Web crawler service shut down successfully")
}