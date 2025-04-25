package commands

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PingCommand는 "pong"으로 응답하는 간단한 명령어입니다
type PingCommand struct{
	prefix string
}

// NewPingCommand는 새로운 ping 명령어를 생성합니다
func NewPingCommand() *PingCommand {
	return &PingCommand{
		prefix: "!",  // 기본 접두사
	}
}

// Register는 명령어 핸들러를 등록합니다
func (c *PingCommand) Register(session *discordgo.Session) {
	session.AddHandler(c.handlePing)
}

// handlePing은 ping 명령어를 처리합니다
func (c *PingCommand) handlePing(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 봇 자신의 메시지는 무시
	if m.Author.ID == s.State.User.ID {
		return
	}

	// 메시지가 "!ping"인지 확인
	if !strings.EqualFold(m.Content, c.prefix+"ping") {
		return
	}

	// 응답 시간 계산
	start := time.Now()
	msg, err := s.ChannelMessageSend(m.ChannelID, "Pinging...")
	if err != nil {
		return
	}

	elapsed := time.Since(start)
	
	// 지연 시간 정보로 메시지 수정
	_, err = s.ChannelMessageEdit(m.ChannelID, msg.ID, 
		"Pong! Latency: " + elapsed.Round(time.Millisecond).String())
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, 
			"Pong! Latency: " + elapsed.Round(time.Millisecond).String())
	}
}