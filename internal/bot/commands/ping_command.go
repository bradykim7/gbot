package commands

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// PingCommand는 "pong"으로 응답하는 간단한 명령어입니다
type PingCommand struct{
	prefix string
}

// Execute implements the Command interface
func (c *PingCommand) Execute(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
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

// Help implements the Command interface
func (c *PingCommand) Help() string {
	return "Responds with pong to verify the bot is running"
}

// NewPingCommand는 새로운 ping 명령어를 생성합니다
func NewPingCommand(prefix string) *PingCommand {
	return &PingCommand{
		prefix: prefix,
	}
}

// Register는 명령어 핸들러를 등록합니다
// Keeping this for backward compatibility but it's not used anymore
func (c *PingCommand) Register(session *discordgo.Session) {
	// Implementation removed as it's now done through the Command interface
}