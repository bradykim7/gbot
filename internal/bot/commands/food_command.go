package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bradykim7/gbot/internal/models"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// FoodCommand는 음식 추천 관련 명령어를 처리합니다
type FoodCommand struct {
	log      *zap.Logger
	db       *storage.MongoDB
	prefix   string
	repo     *storage.FoodRepository
}

// NewFoodCommand는 새로운 음식 명령어 핸들러를 생성합니다
func NewFoodCommand(log *zap.Logger, db *storage.MongoDB, prefix string) *FoodCommand {
	return &FoodCommand{
		log:      log.Named("food-command"),
		db:       db,
		prefix:   prefix,
		repo:     storage.NewFoodRepository(db, log),
	}
}

// Register는 음식 명령어 핸들러를 등록합니다
func (c *FoodCommand) Register(session *discordgo.Session) {
	session.AddHandler(c.handleLunchRecommend)
	session.AddHandler(c.handleDinnerRecommend)
	session.AddHandler(c.handleListFood)
	session.AddHandler(c.handleRegisterFood)
	session.AddHandler(c.handleDeleteFood)
}

// handleLunchRecommend는 점심 추천 명령어(점메추)를 처리합니다
func (c *FoodCommand) handleLunchRecommend(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 봇 자신의 메시지는 무시
	if m.Author.ID == s.State.User.ID {
		return
	}

	// 명령어가 점메추인지 확인
	if m.Content != c.prefix+"점메추" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 랜덤 점심 음식 가져오기
	food, err := c.repo.GetRandomFood(ctx, models.FoodTypeLunch)
	if err != nil {
		c.log.Error("랜덤 점심 음식 가져오기 실패", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "점심 추천을 가져오는 중 오류가 발생했습니다.")
		return
	}

	// 임베드 생성
	embed := &discordgo.MessageEmbed{
		Title:       "오늘의 점심 메뉴 추천",
		Description: fmt.Sprintf("오늘 점심은 **%s** 어떠세요?", food.Name),
		Color:       0xFF9900, // 주황색
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("요청자: %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleDinnerRecommend handles the dinner recommendation command (저메추)
func (c *FoodCommand) handleDinnerRecommend(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if command is 저메추
	if m.Content != c.prefix+"저메추" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get random dinner food
	food, err := c.repo.GetRandomFood(ctx, models.FoodTypeDinner)
	if err != nil {
		c.log.Error("Failed to get random dinner food", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "저녁 추천을 가져오는 중 오류가 발생했습니다.")
		return
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       "오늘의 저녁 메뉴 추천",
		Description: fmt.Sprintf("오늘 저녁은 **%s** 어떠세요?", food.Name),
		Color:       0x3366FF, // Blue
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Requested by %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleListFood handles the food list command (메뉴)
func (c *FoodCommand) handleListFood(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if command starts with 메뉴
	if !strings.HasPrefix(m.Content, c.prefix+"메뉴") {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	parts := strings.Fields(m.Content)
	var foodType models.FoodType
	var title string

	// Determine food type from command args
	if len(parts) > 1 && parts[1] == "점심" {
		foodType = models.FoodTypeLunch
		title = "점심 메뉴 목록"
	} else if len(parts) > 1 && parts[1] == "저녁" {
		foodType = models.FoodTypeDinner
		title = "저녁 메뉴 목록"
	} else {
		// Default to listing all food types
		var lunchMsg, dinnerMsg string

		// Get lunch foods
		lunchFoods, err := c.repo.GetAllFoods(ctx, models.FoodTypeLunch)
		if err != nil {
			c.log.Error("Failed to get all lunch foods", zap.Error(err))
		} else {
			var lunchNames []string
			for _, food := range lunchFoods {
				lunchNames = append(lunchNames, food.Name)
			}
			lunchMsg = fmt.Sprintf("**점심 메뉴(%d)**: %s", len(lunchFoods), strings.Join(lunchNames, ", "))
		}

		// Get dinner foods
		dinnerFoods, err := c.repo.GetAllFoods(ctx, models.FoodTypeDinner)
		if err != nil {
			c.log.Error("Failed to get all dinner foods", zap.Error(err))
		} else {
			var dinnerNames []string
			for _, food := range dinnerFoods {
				dinnerNames = append(dinnerNames, food.Name)
			}
			dinnerMsg = fmt.Sprintf("**저녁 메뉴(%d)**: %s", len(dinnerFoods), strings.Join(dinnerNames, ", "))
		}

		// Create embed with both types
		embed := &discordgo.MessageEmbed{
			Title:       "메뉴 목록",
			Description: lunchMsg + "\n\n" + dinnerMsg,
			Color:       0x00FF00, // Green
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Requested by %s", m.Author.Username),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Get foods of specific type
	foods, err := c.repo.GetAllFoods(ctx, foodType)
	if err != nil {
		c.log.Error("Failed to get all foods", zap.Error(err), zap.String("type", string(foodType)))
		s.ChannelMessageSend(m.ChannelID, "메뉴 목록을 가져오는 중 오류가 발생했습니다.")
		return
	}

	// Format food names
	var foodNames []string
	for _, food := range foods {
		foodNames = append(foodNames, food.Name)
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("**총 %d개의 메뉴**: %s", len(foods), strings.Join(foodNames, ", ")),
		Color:       0x00FF00, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Requested by %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleRegisterFood handles the food registration command (점메추등록/저메추등록)
func (c *FoodCommand) handleRegisterFood(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check command type
	var foodType models.FoodType
	var prefix string

	if strings.HasPrefix(m.Content, c.prefix+"점메추등록") {
		foodType = models.FoodTypeLunch
		prefix = c.prefix + "점메추등록"
	} else if strings.HasPrefix(m.Content, c.prefix+"저메추등록") {
		foodType = models.FoodTypeDinner
		prefix = c.prefix + "저메추등록"
	} else {
		return
	}

	// Extract food name
	foodName := strings.TrimSpace(strings.TrimPrefix(m.Content, prefix))
	if foodName == "" {
		s.ChannelMessageSend(m.ChannelID, "등록할 메뉴 이름을 입력해주세요.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create food
	food := models.NewFood(foodName, foodType, m.Author.Username)

	// Save to database
	err := c.repo.SaveFood(ctx, food)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("'%s' 메뉴는 이미 등록되어 있습니다.", foodName))
		} else {
			c.log.Error("Failed to save food", zap.Error(err), zap.String("name", foodName))
			s.ChannelMessageSend(m.ChannelID, "메뉴를 등록하는 중 오류가 발생했습니다.")
		}
		return
	}

	// Create success embed
	typeStr := "점심"
	if foodType == models.FoodTypeDinner {
		typeStr = "저녁"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "메뉴 등록 완료",
		Description: fmt.Sprintf("'%s' 메뉴가 %s 목록에 등록되었습니다.", foodName, typeStr),
		Color:       0x00FF00, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Added by %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleDeleteFood handles the food deletion command (점메추삭제/저메추삭제)
func (c *FoodCommand) handleDeleteFood(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check command type
	var foodType models.FoodType
	var prefix string

	if strings.HasPrefix(m.Content, c.prefix+"점메추삭제") {
		foodType = models.FoodTypeLunch
		prefix = c.prefix + "점메추삭제"
	} else if strings.HasPrefix(m.Content, c.prefix+"저메추삭제") {
		foodType = models.FoodTypeDinner
		prefix = c.prefix + "저메추삭제"
	} else {
		return
	}

	// Extract food name
	foodName := strings.TrimSpace(strings.TrimPrefix(m.Content, prefix))
	if foodName == "" {
		s.ChannelMessageSend(m.ChannelID, "삭제할 메뉴 이름을 입력해주세요.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Delete from database
	err := c.repo.DeleteFood(ctx, foodName, foodType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("'%s' 메뉴를 찾을 수 없습니다.", foodName))
		} else {
			c.log.Error("Failed to delete food", zap.Error(err), zap.String("name", foodName))
			s.ChannelMessageSend(m.ChannelID, "메뉴를 삭제하는 중 오류가 발생했습니다.")
		}
		return
	}

	// Create success embed
	typeStr := "점심"
	if foodType == models.FoodTypeDinner {
		typeStr = "저녁"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "메뉴 삭제 완료",
		Description: fmt.Sprintf("'%s' 메뉴가 %s 목록에서 삭제되었습니다.", foodName, typeStr),
		Color:       0xFF0000, // Red
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Removed by %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}