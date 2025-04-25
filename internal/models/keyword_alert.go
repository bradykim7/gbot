package models

import (
	"strings"
)

// KeywordAlert는 키워드 기반 상품 알림을 나타냅니다
type KeywordAlert struct {
	ID        string `bson:"_id,omitempty"`
	Keyword   string `bson:"keyword"`
	UserID    string `bson:"user_id"`
	ChannelID string `bson:"channel_id"`
	GuildID   string `bson:"guild_id"`
	CreatedAt int64  `bson:"created_at"`
	IsActive  bool   `bson:"is_active"`
}

// String은 알림의 문자열 표현을 반환합니다
func (k *KeywordAlert) String() string {
	return "Alert for '" + k.Keyword + "' by <@" + k.UserID + ">"
}

// KeywordExists는 사용자의 키워드 알림이 존재하는지 확인합니다
func KeywordExists(alerts []*KeywordAlert, keyword, userID string) bool {
	normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
	
	for _, alert := range alerts {
		if strings.ToLower(alert.Keyword) == normalizedKeyword && alert.UserID == userID {
			return true
		}
	}
	
	return false
}

// GetMatchingAlerts는 상품 제목과 일치하는.ㄴ 알림을 찾습니다
func GetMatchingAlerts(alerts []*KeywordAlert, title string) []*KeywordAlert {
	normalizedTitle := strings.ToLower(title)
	var matching []*KeywordAlert
	
	for _, alert := range alerts {
		if alert.IsActive && strings.Contains(normalizedTitle, strings.ToLower(alert.Keyword)) {
			matching = append(matching, alert)
		}
	}
	
	return matching
}
