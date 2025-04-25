# Discord Bot - Go Implementation

## Overview
This repository contains a Discord bot written in Go. It has two main components:
1. **HybaBot** - A Discord bot that handles user commands
2. **PriceSota** - A web crawler that scans deal websites and sends notifications

## Bug Fixes
The following bugs were fixed in the latest update:

1. **Command Registry**:
   - Fixed inconsistency between direct handler registration and command registry pattern
   - Implemented the Command interface properly for all commands
   - Removed redundant command handlers

2. **Logger Consistency**:
   - Fixed inconsistent logger usage between zap.Logger and custom logger
   - Improved error handling in logger initialization
   - Added better context to log messages

3. **Model Structure**:
   - Added missing Username field to KeywordAlert model
   - Fixed inconsistencies in data structures
   - Enhanced Product model with additional fields needed for notifications

4. **Command Processing**:
   - Improved command argument parsing
   - Added better help messages and error responses
   - Fixed prefix handling in ping command
   
5. **Crawler Improvements**:
   - Fixed product data collection in PpomppuCrawler
   - Added proper price string handling
   - Improved user notification to handle missing usernames
   - Added fallback to user ID mentions when usernames are not available
   
6. **Discord Bot Notifications**:
   - Fixed price display in product notifications
   - Improved error handling in notification sending
   - Added proper formatting of timestamps

## Features
- **Commands**:
  - `!ping` - Check bot latency
  - `!alert add [keyword]` - Add a keyword alert
  - `!alert remove [keyword]` - Remove a keyword alert
  - `!alert list` - List all your keyword alerts
  - `!food lunch` - Get a random lunch recommendation
  - `!food dinner` - Get a random dinner recommendation
  - `!food list [lunch/dinner]` - List all foods
  - `!food add [lunch/dinner] [name]` - Add a food
  - `!food remove [lunch/dinner] [name]` - Remove a food

## Setup
1. Clone the repository
2. Create a .env file with your Discord token and MongoDB URI
3. Run `go build ./cmd/hybabot` to build the bot
4. Run `go build ./cmd/pricesota` to build the crawler

## Configuration
The following environment variables are used:
- `DISCORD_TOKEN` - Your Discord bot token
- `DISCORD_GUILD` - Your Discord guild ID (optional)
- `COMMAND_PREFIX` - Command prefix (default: !)
- `MONGODB_URI` - MongoDB connection URI
- `PRODUCT_CHANNEL_ID` - Channel ID for product notifications

## Korean Commands
모든 명령어는 한국어로도 사용할 수 있습니다:
- `!알림 추가 [키워드]` - 키워드 알림 추가
- `!알림 삭제 [키워드]` - 키워드 알림 삭제
- `!알림 목록` - 모든 키워드 알림 보기
- `!메뉴 점심` - 점심 추천 받기
- `!메뉴 저녁` - 저녁 추천 받기
- `!메뉴 목록 [점심/저녁]` - 모든 메뉴 보기
- `!메뉴 추가 [점심/저녁] [이름]` - 메뉴 추가
- `!메뉴 삭제 [점심/저녁] [이름]` - 메뉴 삭제