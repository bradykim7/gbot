package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bradykim7/gbot/internal/models"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// AlertCommand는 키워드 알림 관련 명령어를 처리합니다
type AlertCommand struct {
	log    *zap.Logger
	db     *storage.MongoDB
	prefix string
}

// Execute implements the Command interface
func (c *AlertCommand) Execute(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		c.sendHelpMessage(s, m.ChannelID)
		return
	}

	subCommand := args[0]
	args = args[1:]

	switch subCommand {
	case "add", "추가":
		c.handleAddAlertFromArgs(s, m, args)
	case "remove", "삭제":
		c.handleRemoveAlertFromArgs(s, m, args)
	case "list", "목록":
		c.handleListAlertsFromArgs(s, m, args)
	default:
		c.sendHelpMessage(s, m.ChannelID)
	}
}

// Help implements the Command interface
func (c *AlertCommand) Help() string {
	return fmt.Sprintf("**Alert Command Usage**\n"+
		"%s alert add [keyword] - Add a keyword alert\n"+
		"%s alert remove [keyword] - Remove a keyword alert\n"+
		"%s alert list - List all your keyword alerts", 
		c.prefix, c.prefix, c.prefix)
}

// sendHelpMessage sends the help message to the specified channel
func (c *AlertCommand) sendHelpMessage(s *discordgo.Session, channelID string) {
	s.ChannelMessageSend(channelID, c.Help())
}

// handleAddAlertFromArgs processes alert add command from parsed arguments
func (c *AlertCommand) handleAddAlertFromArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "추가할 키워드를 입력해주세요.")
		return
	}
	
	keyword := strings.Join(args, " ")
	
	// Create timeout context for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 데이터베이스에 알림 생성
	alert := models.KeywordAlert{
		Keyword:   keyword,
		UserID:    m.Author.ID,
		Username:  m.Author.Username,
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
		CreatedAt: time.Now().Unix(),
		IsActive:  true,
	}

	// 알림이 이미 존재하는지 확인
	exists, err := c.checkAlertExists(ctx, m.Author.ID, keyword)
	if err != nil {
		c.log.Error("알림 존재 여부 확인 실패", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "알림 존재 여부를 확인하는 중 오류가 발생했습니다.")
		return
	}

	if exists {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("'%s' 키워드에 대한 알림이 이미 존재합니다.", keyword))
		return
	}

	// 알림 삽입
	collection := c.db.Collection("keyword_alerts")
	_, err = collection.InsertOne(ctx, alert)
	if err != nil {
		c.log.Error("알림 삽입 실패", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "알림을 추가하는 중 오류가 발생했습니다.")
		return
	}

	// 응답 임베드 생성
	embed := &discordgo.MessageEmbed{
		Title:       "키워드 알림 추가됨",
		Description: fmt.Sprintf("키워드: **%s**에 대한 알림이 성공적으로 추가되었습니다.", keyword),
		Color:       0x00ff00, // 녹색
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("요청자: %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	c.log.Info("알림 추가됨", 
		zap.String("keyword", keyword), 
		zap.String("user_id", m.Author.ID),
		zap.String("author", m.Author.Username))
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleRemoveAlertFromArgs processes alert remove command from parsed arguments
func (c *AlertCommand) handleRemoveAlertFromArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "삭제할 키워드를 입력해주세요.")
		return
	}
	
	keyword := strings.Join(args, " ")

	// Create timeout context for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 데이터베이스에서 알림 삭제
	collection := c.db.Collection("keyword_alerts")
	filter := bson.M{
		"user_id": m.Author.ID,
		"keyword": keyword,
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		c.log.Error("알림 삭제 실패", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "알림을 삭제하는 중 오류가 발생했습니다.")
		return
	}

	if result.DeletedCount == 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("'%s' 키워드에 대한 알림을 찾을 수 없습니다.", keyword))
		return
	}

	// 응답 임베드 생성
	embed := &discordgo.MessageEmbed{
		Title:       "키워드 알림 삭제됨",
		Description: fmt.Sprintf("키워드: **%s**에 대한 알림이 성공적으로 삭제되었습니다.", keyword),
		Color:       0xff0000, // 빨간색
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("요청자: %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	c.log.Info("알림 삭제됨", 
		zap.String("keyword", keyword), 
		zap.String("user_id", m.Author.ID))
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleListAlertsFromArgs processes alert list command from parsed arguments
func (c *AlertCommand) handleListAlertsFromArgs(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Create timeout context for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 데이터베이스에서 알림 목록 가져오기
	collection := c.db.Collection("keyword_alerts")
	filter := bson.M{
		"user_id":  m.Author.ID,
		"is_active": true,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		c.log.Error("알림 목록 조회 실패", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "알림 목록을 조회하는 중 오류가 발생했습니다.")
		return
	}
	defer cursor.Close(ctx)

	var alerts []models.KeywordAlert
	if err := cursor.All(ctx, &alerts); err != nil {
		c.log.Error("알림 디코딩 실패", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "알림 정보를 디코딩하는 중 오류가 발생했습니다.")
		return
	}

	if len(alerts) == 0 {
		s.ChannelMessageSend(m.ChannelID, "활성화된 알림이 없습니다.")
		return
	}

	// 응답 임베드 생성
	embed := &discordgo.MessageEmbed{
		Title:       "키워드 알림 목록",
		Description: fmt.Sprintf("%d개의 활성화된 알림이 있습니다:", len(alerts)),
		Color:       0x0000ff, // 파란색
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("요청자: %s", m.Author.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 각 알림에 대한 필드 추가
	for i, alert := range alerts {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("알림 #%d", i+1),
			Value: alert.Keyword,
		})
	}

	c.log.Info("알림 목록 조회됨", 
		zap.String("user_id", m.Author.ID), 
		zap.Int("count", len(alerts)))
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// checkAlertExists는 사용자 ID와 키워드로 알림이 존재하는지 확인합니다
func (c *AlertCommand) checkAlertExists(ctx context.Context, userID, keyword string) (bool, error) {
	collection := c.db.Collection("keyword_alerts")
	filter := bson.M{
		"user_id": userID,
		"keyword": keyword,
		"is_active": true,
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}