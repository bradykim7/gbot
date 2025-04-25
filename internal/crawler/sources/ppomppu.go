package sources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bradykim7/gbot/internal/crawler"
	"github.com/bradykim7/gbot/internal/models"
	"go.uber.org/zap"
)

const (
	ppomppuBaseURL     = "https://www.ppomppu.co.kr/zboard/zboard.php?id=ppomppu"
	ppomppuItemURLBase = "https://www.ppomppu.co.kr/zboard/"
)

// PpomppuCrawler is a crawler for Ppomppu website
type PpomppuCrawler struct {
	*crawler.BaseCrawler
}

// NewPpomppuCrawler creates a new Ppomppu crawler
func NewPpomppuCrawler(log *zap.Logger) *PpomppuCrawler {
	return &PpomppuCrawler{
		BaseCrawler: crawler.NewBaseCrawler(log.Named("ppomppu-crawler")),
	}
}

// Name returns the name of the source
func (c *PpomppuCrawler) Name() string {
	return "Ppomppu"
}

// Crawl fetches and parses deals from Ppomppu
func (c *PpomppuCrawler) Crawl(ctx context.Context) ([]models.Product, error) {
	c.Logger.Info("Starting Ppomppu crawl")
	
	content, err := c.FetchURL(ctx, ppomppuBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Ppomppu: %w", err)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var products []models.Product

	// Extract deals from the page
	doc.Find("tr.list1, tr.list0").Each(func(i int, s *goquery.Selection) {
		// Skip ads and notices
		if s.Find("span.list_comment").Length() == 0 && !strings.Contains(s.Text(), "공지") {
			product, err := c.parseProduct(s)
			if err == nil && product != nil {
				products = append(products, *product)
			}
		}
	})

	c.Logger.Info("Ppomppu crawl completed", zap.Int("products_found", len(products)))
	return products, nil
}

// parseProduct extracts product information from HTML selection
func (c *PpomppuCrawler) parseProduct(s *goquery.Selection) (*models.Product, error) {
	// Extract title
	titleEl := s.Find("font.list_title")
	if titleEl.Length() == 0 {
		return nil, fmt.Errorf("title element not found")
	}
	
	title := strings.TrimSpace(titleEl.Text())
	if title == "" {
		return nil, fmt.Errorf("empty title")
	}

	// Extract URL
	urlPath, exists := s.Find("a").First().Attr("href")
	if !exists {
		return nil, fmt.Errorf("URL not found")
	}
	
	// Correct relative URL
	url := urlPath
	if !strings.HasPrefix(urlPath, "http") {
		url = ppomppuItemURLBase + strings.TrimPrefix(urlPath, "./")
	}

	// Extract price with regex
	priceRegex := regexp.MustCompile(`(\d{1,3}(,\d{3})*원|\d+원)`)
	priceMatches := priceRegex.FindStringSubmatch(title)
	var price int
	var priceStr string
	
	if len(priceMatches) > 0 {
		priceStr = priceMatches[0]
		// Clean up price string and convert to integer
		priceStr = strings.ReplaceAll(priceStr, ",", "")
		priceStr = strings.ReplaceAll(priceStr, "원", "")
		price, _ = strconv.Atoi(priceStr)
	}

	// Extract comments count
	commentsStr := strings.TrimSpace(s.Find("span.list_comment").Text())
	commentsStr = strings.Trim(commentsStr, "[]")
	comments, _ := strconv.Atoi(commentsStr)

	// Extract views count
	viewsStr := strings.TrimSpace(s.Find("td").Eq(5).Text())
	views, _ := strconv.Atoi(viewsStr)

	// Get date
	dateStr := strings.TrimSpace(s.Find("td").Eq(4).Text())
	
	return &models.Product{
		Title:        title,
		URL:          url,
		Price:        price,
		PriceString:  priceStr,
		Source:       "Ppomppu",
		Comments:     comments,
		Views:        views,
		DateStr:      dateStr,
		CrawledAt:    time.Now(),
	}, nil
}