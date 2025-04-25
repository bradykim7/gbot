package models

import (
	"fmt"
	"strconv"
	"time"
)

// Product represents a product from a deal website
type Product struct {
	ID            string    `bson:"_id,omitempty"`
	Title         string    `bson:"title"`
	Website       string    `bson:"website"`
	Product       string    `bson:"product"`
	Category      string    `bson:"category"`
	URL           string    `bson:"url"`
	KOPrice       int       `bson:"ko_price,omitempty"`
	USPrice       float64   `bson:"us_price,omitempty"`
	PriceString   string    `bson:"price_string,omitempty"`
	UploadDate    int64     `bson:"upload_date,omitempty"`
	UploadSite    string    `bson:"upload_site,omitempty"`
	Comments      int       `bson:"comments,omitempty"`
	Views         int       `bson:"views,omitempty"`
	CrawledAt     time.Time `bson:"crawled_at,omitempty"`
	Source        string    `bson:"source,omitempty"`
	ImageURL      string    `bson:"image_url,omitempty"`     // 상품 이미지 URL
	IsHot         bool      `bson:"is_hot,omitempty"`        // 인기 상품 여부
	Rating        float64   `bson:"rating,omitempty"`        // 평점 (있는 경우)
	DiscountRate  int       `bson:"discount_rate,omitempty"` // 할인율 (%)
	OriginalPrice int       `bson:"original_price,omitempty"`// 원래 가격
	Notified      bool      `bson:"notified"`                // 알림 발송 여부
	Keywords      []string  `bson:"keywords,omitempty"`      // 매칭된 키워드 목록
}

// GetPriceString returns a formatted price string
func (p *Product) GetPriceString() string {
	// If we already have a formatted price string, use it
	if p.PriceString != "" {
		return p.PriceString
	}
	
	// Otherwise, format based on the price values
	if p.KOPrice > 0 {
		return fmt.Sprintf("%s KRW", formatNumber(p.KOPrice))
	} else if p.USPrice > 0 {
		return fmt.Sprintf("$%.2f USD", p.USPrice)
	}
	return "Price unknown"
}

// String returns a string representation of the product
func (p *Product) String() string {
	return fmt.Sprintf("%s (%s) from %s", p.Product, p.GetPriceString(), p.Website)
}

// formatNumber formats a number with commas
func formatNumber(n int) string {
	in := strconv.FormatInt(int64(n), 10)
	out := make([]byte, 0, len(in)+(len(in)-1)/3)
	
	// Add commas
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	
	return string(out)
}