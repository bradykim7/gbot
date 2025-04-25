package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bradykim7/gbot/internal/bot"
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
	
	log := logger.Named("hybabot")
	
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
	
	// Initialize and run the bot
	discordBot, err := bot.New(cfg, log)
	if err != nil {
		log.Fatal("Failed to initialize bot", zap.Error(err))
	}
	
	if err := discordBot.Start(ctx); err != nil {
		log.Fatal("Bot error", zap.Error(err))
	}
	
	log.Info("Discord bot shut down successfully")
}