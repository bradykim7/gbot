package bot

import (
	"context"
	"fmt"

	"github.com/bradykim7/gbot/internal/bot/commands"
	"github.com/bradykim7/gbot/internal/storage"
	"github.com/bradykim7/gbot/pkg/config"
	"go.uber.org/zap"
	"github.com/bwmarrin/discordgo"
)

// Bot은 Discord 봇을 나타냅니다
type Bot struct {
	session  *discordgo.Session
	config   *config.Config
	log      *zap.Logger
	commands *commands.Registry
	db       *storage.MongoDB
}

// New는 새로운 Bot 인스턴스를 생성합니다
func New(cfg *config.Config, log *zap.Logger) (*Bot, error) {
	// Discord 세션 생성
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("Discord 세션 생성 오류: %w", err)
	}

	// MongoDB 연결
	db, err := storage.NewMongoDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("MongoDB 연결 오류: %w", err)
	}
	
	// 봇 인스턴스 생성
	bot := &Bot{
		session:  session,
		config:   cfg,
		log:      log.Named("bot"),
		commands: commands.NewRegistry(cfg.CommandPrefix),
		db:       db,
	}
	
	// 이벤트 핸들러 설정
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onMessageCreate)
	
	// Intents 설정
	session.Identify.Intents = discordgo.IntentsGuildMessages | 
		discordgo.IntentsGuildVoiceStates | 
		discordgo.IntentsDirectMessages | 
		discordgo.IntentsMessageContent
	
	// 명령어 등록
	bot.registerCommands()
	
	return bot, nil
}

// Start는 봇을 시작합니다
func (b *Bot) Start(ctx context.Context) error {
	// Discord에 연결
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("Discord 세션 열기 오류: %w", err)
	}
	
	b.log.Info("봇이 실행 중입니다. 종료하려면 CTRL-C를 누르세요.")
	
	// 컨텍스트가 취소될 때까지 대기
	<-ctx.Done()
	
	// 리소스 정리
	return b.Close()
}

// Close는 리소스를 정리합니다
func (b *Bot) Close() error {
	if err := b.session.Close(); err != nil {
		return fmt.Errorf("Discord 세션 닫기 오류: %w", err)
	}
	
	if err := b.db.Disconnect(); err != nil {
		return fmt.Errorf("MongoDB 연결 해제 오류: %w", err)
	}
	
	return nil
}

// onReady는 봇이 준비되었을 때의 이벤트 핸들러입니다
func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	b.log.Info("봇 로그인 완료", 
		zap.String("username", r.User.Username), 
		zap.String("discriminator", r.User.Discriminator))
	
	// 상태 설정
	err := s.UpdateGameStatus(0, "with golang")
	if err != nil {
		b.log.Error("상태 설정 오류", zap.Error(err))
	}
}

// onMessageCreate는 메시지가 생성되었을 때의 이벤트 핸들러입니다
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 봇 자신의 메시지는 무시
	if m.Author.ID == s.State.User.ID {
		return
	}
	
	// 메시지 로깅
	b.log.Debug("메시지 수신됨", 
		zap.String("guild_id", m.GuildID), 
		zap.String("channel_id", m.ChannelID), 
		zap.String("user_id", m.Author.ID), 
		zap.String("username", m.Author.Username), 
		zap.String("content", m.Content))
	
	// 명령어 처리
	b.commands.Handle(s, m)
}

// registerCommands는 모든 명령어를 등록합니다
func (b *Bot) registerCommands() {
	// Ping 명령어 등록
	pingCmd := commands.NewPingCommand()
	pingCmd.Register(b.session)
	
	// 알림 명령어 등록
	alertCmd := commands.NewAlertCommand(b.log, b.db, b.config.CommandPrefix)
	alertCmd.Register(b.session)
	
	// 음식 명령어 등록
	foodCmd := commands.NewFoodCommand(b.log, b.db, b.config.CommandPrefix)
	foodCmd.Register(b.session)
	
	// TODO: 다른 명령어들도 구현되는 대로 등록
	// 참고: Registry 패턴 대신 직접 핸들러 등록 방식 사용
}