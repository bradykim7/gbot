#!/bin/bash

# Create main directories
mkdir -p cmd/hybabot
mkdir -p cmd/pricesota
mkdir -p internal/bot/commands
mkdir -p internal/bot/handlers
mkdir -p internal/bot/services
mkdir -p internal/models
mkdir -p internal/storage
mkdir -p internal/crawler/sources
mkdir -p internal/crawler/parser
mkdir -p internal/common
mkdir -p pkg/discord
mkdir -p pkg/config
mkdir -p pkg/logger
mkdir -p configs

# Create main files
touch cmd/hybabot/main.go
touch cmd/pricesota/main.go
touch docker-compose.yml
touch Dockerfile.bot
touch Dockerfile.crawler
touch .env

echo "Directory structure created successfully!"
