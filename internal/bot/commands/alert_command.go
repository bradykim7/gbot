package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bradykim7/gbot/internal/models"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bradykim7/gbot/pkg/logger"
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

// NewAlertCommand는 새로운 알림 명령어 핸들러를 생성합니다
func NewAlertCommand(log *zap.Logger, db *storage.MongoDB, prefix string) *AlertCommand {
	return &AlertCommand{
		log:    log.Named("alert-command"),
		db:     db,
		prefix: prefix,
	}
}

// Register는 명령어 핸들러를 등록합니다
func (c *AlertCommand) Register(session *discordgo.Session) {
	session.AddHandler(c.handleAddAlert)
	session.AddHandler(c.handleRemoveAlert)
	session.AddHandler(c.handleListAlerts)
}

// handleAddAlert는 알림 추가 명령어를 처리합니다
func (c *AlertCommand) handleAddAlert(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 봇 자신의 메시지는 무시
	if m.Author.ID == s.State.User.ID {
		return
	}

	// 명령어가 alert add인지 확인
	prefix := c.prefix + "alert add "
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	// 키워드 추출
	keyword := strings.TrimPrefix(m.Content, prefix)
	keyword = strings.TrimSpace(keyword)

	if keyword == "" {
		s.ChannelMessageSend(m.ChannelID, "추가할 키워드를 입력해주세요.")
		return
	}

	// 데이터베이스에 알림 생성
	alert := models.KeywordAlert{
		Keyword:   keyword,
		UserID:    m.Author.ID,
		Username:  m.Author.Username,
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
		CreatedAt: time.Now(),
		IsActive:  true,
	}

	// 알림이 이미 존재하는지 확인
	exists, err := c.checkAlertExists(m.Author.ID, keyword)
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
	_, err = collection.InsertOne(context.Background(), alert)
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

// handleRemoveAlert는 알림 삭제 명령어를 처리합니다
func (c *AlertCommand) handleRemoveAlert(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 봇 자신의 메시지는 무시
	if m.Author.ID == s.State.User.ID {
		return
	}

	// 명령어가 alert remove인지 확인
	prefix := c.prefix + "alert remove "
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	// 키워드 추출
	keyword := strings.TrimPrefix(m.Content, prefix)
	keyword = strings.TrimSpace(keyword)

	if keyword == "" {
		s.ChannelMessageSend(m.ChannelID, "삭제할 키워드를 입력해주세요.")
		return
	}

	// 데이터베이스에서 알림 삭제
	collection := c.db.Collection("keyword_alerts")
	filter := bson.M{
		"user_id": m.Author.ID,
		"keyword": keyword,
	}

	result, err := collection.DeleteOne(context.Background(), filter)
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

// handleListAlerts는 알림 목록 명령어를 처리합니다
func (c *AlertCommand) handleListAlerts(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 봇 자신의 메시지는 무시
	if m.Author.ID == s.State.User.ID {
		return
	}

	// 명령어가 alert list인지 확인
	prefix := c.prefix + "alert list"
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	// 데이터베이스에서 알림 목록 가져오기
	collection := c.db.Collection("keyword_alerts")
	filter := bson.M{
		"user_id":  m.Author.ID,
		"is_active": true,
	}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		c.log.Error("알림 목록 조회 실패", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "알림 목록을 조회하는 중 오류가 발생했습니다.")
		return
	}
	defer cursor.Close(context.Background())

	var alerts []models.KeywordAlert
	if err := cursor.All(context.Background(), &alerts); err != nil {
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
func (c *AlertCommand) checkAlertExists(userID, keyword string) (bool, error) {
	collection := c.db.Collection("keyword_alerts")
	filter := bson.M{
		"user_id": userID,
		"keyword": keyword,
		"is_active": true,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}