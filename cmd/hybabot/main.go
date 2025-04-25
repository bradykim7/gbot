package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/hybabot/internal/bot"
	"github.com/yourusername/hybabot/pkg/config"
	"github.com/yourusername/hybabot/pkg/logger"
)

func main() {
	// Initialize logger
	log := logger.New("bot")
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Create context that will be canceled on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle graceful shutdown
	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
		<-sc
		log.Info("Received shutdown signal, exiting...")
		cancel()
	}()
	
	// Initialize and run the bot
	discordBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}
	
	if err := discordBot.Start(ctx); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}