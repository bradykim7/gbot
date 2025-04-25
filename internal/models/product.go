package models

import (
	"fmt"
	"strconv"
)

// Product represents a product from a deal website
type Product struct {
	ID         string `bson:"_id,omitempty"`
	Title      string `bson:"title"`
	Website    string `bson:"website"`
	Product    string `bson:"product"`
	Category   string `bson:"category"`
	URL        string `bson:"url"`
	KOPrice    int    `bson:"ko_price,omitempty"`
	USPrice    float64 `bson:"us_price,omitempty"`
	UploadDate int64  `bson:"upload_date,omitempty"`
	UploadSite string `bson:"upload_site,omitempty"`
}

// GetPriceString returns a formatted price string
func (p *Product) GetPriceString() string {
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
		out = append(out, c)
	}
	
	return string(out)
}