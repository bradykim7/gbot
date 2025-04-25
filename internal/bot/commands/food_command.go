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

// Execute implements the Command interface
func (c *FoodCommand) Execute(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		c.sendHelpMessage(s, m.ChannelID)
		return
	}

	subCommand := args[0]
	args = args[1:]

	switch subCommand {
	case "lunch", "점심":
		c.handleLunchRecommendArgs(s, m, args)
	case "dinner", "저녁":
		c.handleDinnerRecommendArgs(s, m, args)
	case "list", "목록":
		c.handleListFoodArgs(s, m, args)
	case "add", "추가":
		c.handleRegisterFoodArgs(s, m, args)
	case "remove", "삭제":
		c.handleDeleteFoodArgs(s, m, args)
	default:
		c.sendHelpMessage(s, m.ChannelID)
	}
}

// Help implements the Command interface
func (c *FoodCommand) Help() string {
	return fmt.Sprintf("**Food Command Usage**\n"+
		"%s food lunch/점심 - Get lunch recommendation\n"+
		"%s food dinner/저녁 - Get dinner recommendation\n"+
		"%s food list/목록 [lunch/dinner] - List all registered food\n"+
		"%s food add/추가 [lunch/dinner] [name] - Add new food\n"+
		"%s food remove/삭제 [lunch/dinner] [name] - Remove food",
		c.prefix, c.prefix, c.prefix, c.prefix, c.prefix)
}

// sendHelpMessage sends the help message to the specified channel
func (c *FoodCommand) sendHelpMessage(s *discordgo.Session, channelID string) {
	s.ChannelMessageSend(channelID, c.Help())
}

// handleLunchRecommendArgs handles the lunch recommendation with arguments
func (c *FoodCommand) handleLunchRecommendArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get random lunch food
	food, err := c.repo.GetRandomFood(ctx, models.FoodTypeLunch)
	if err != nil {
		c.log.Error("Failed to get random lunch food", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "점심 추천을 가져오는 중 오류가 발생했습니다.")
		return
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       "오늘의 점심 메뉴 추천",
		Description: fmt.Sprintf("오늘 점심은 **%s** 어떠세요?", food.Name),
		Color:       0xFF9900, // Orange
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Requested by %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleDinnerRecommendArgs handles the dinner recommendation with arguments
func (c *FoodCommand) handleDinnerRecommendArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
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

// handleListFoodArgs handles listing food with arguments
func (c *FoodCommand) handleListFoodArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var foodType models.FoodType
	var title string

	// Determine food type from arguments
	if len(args) > 0 {
		switch args[0] {
		case "lunch", "점심":
			foodType = models.FoodTypeLunch
			title = "점심 메뉴 목록"
		case "dinner", "저녁":
			foodType = models.FoodTypeDinner
			title = "저녁 메뉴 목록"
		default:
			// Default to showing both lists
			foodType = ""
		}
	}

	// If no food type specified, show both lists
	if foodType == "" {
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

// handleRegisterFoodArgs handles food registration with arguments
func (c *FoodCommand) handleRegisterFoodArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "사용법: "+c.prefix+"food add [lunch/dinner] [food name]")
		return
	}

	var foodType models.FoodType
	typeArg := args[0]
	foodName := strings.Join(args[1:], " ")

	// Determine food type
	switch typeArg {
	case "lunch", "점심":
		foodType = models.FoodTypeLunch
	case "dinner", "저녁":
		foodType = models.FoodTypeDinner
	default:
		s.ChannelMessageSend(m.ChannelID, "유효한 메뉴 유형(lunch/점심 또는 dinner/저녁)을 입력해주세요.")
		return
	}

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

// handleDeleteFoodArgs handles food deletion with arguments
func (c *FoodCommand) handleDeleteFoodArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "사용법: "+c.prefix+"food remove [lunch/dinner] [food name]")
		return
	}

	var foodType models.FoodType
	typeArg := args[0]
	foodName := strings.Join(args[1:], " ")

	// Determine food type
	switch typeArg {
	case "lunch", "점심":
		foodType = models.FoodTypeLunch
	case "dinner", "저녁":
		foodType = models.FoodTypeDinner
	default:
		s.ChannelMessageSend(m.ChannelID, "유효한 메뉴 유형(lunch/점심 또는 dinner/저녁)을 입력해주세요.")
		return
	}

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

// NewFoodCommand는 새로운 음식 명령어 핸들러를 생성합니다
func NewFoodCommand(log *zap.Logger, db *storage.MongoDB, prefix string) *FoodCommand {
	return &FoodCommand{
		log:      log.Named("food-command"),
		db:       db,
		prefix:   prefix,
		repo:     storage.NewFoodRepository(db, log),
	}
}

// Register는 더 이상 사용되지 않습니다. 대신 Command 인터페이스를 통해 명령어가 처리됩니다.
func (c *FoodCommand) Register(session *discordgo.Session) {
	// 빈 구현 - 하위 호환성 유지용
}