package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bradykim7/gbot/pkg/logger"
	"go.uber.org/zap"
)

const (
	maxRetries        = 3
	defaultTimeout    = 30 * time.Second
	retryWaitDuration = 2 * time.Second
)

// BaseCrawler provides common functionality for all crawlers
type BaseCrawler struct {
	Client  *http.Client
	Logger  *zap.Logger
	Headers map[string]string
}

// NewBaseCrawler creates a new base crawler with default settings
func NewBaseCrawler(log *zap.Logger) *BaseCrawler {
	return &BaseCrawler{
		Client: &http.Client{
			Timeout: defaultTimeout,
		},
		Logger:  log.Named("base-crawler"),
		Headers: getDefaultHeaders(),
	}
}

// FetchURL retrieves the content of a URL with retry logic
func (c *BaseCrawler) FetchURL(ctx context.Context, url string) ([]byte, error) {
	var (
		resp    *http.Response
		err     error
		content []byte
		retries = 0
	)

	for retries < maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			c.Logger.Error("Failed to create request", zap.Error(err), zap.String("url", url))
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		for key, value := range c.Headers {
			req.Header.Set(key, value)
		}

		// Execute request
		resp, err = c.Client.Do(req)
		if err != nil {
			c.Logger.Warn("HTTP request failed", 
				zap.Error(err), 
				zap.String("url", url), 
				zap.Int("attempt", retries+1))
			
			retries++
			if retries < maxRetries {
				time.Sleep(retryWaitDuration)
				continue
			}
			return nil, fmt.Errorf("failed to fetch URL after %d attempts: %w", maxRetries, err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			c.Logger.Warn("Non-OK HTTP status", 
				zap.Int("status", resp.StatusCode), 
				zap.String("url", url), 
				zap.Int("attempt", retries+1))
			
			retries++
			if retries < maxRetries {
				time.Sleep(retryWaitDuration)
				continue
			}
			return nil, fmt.Errorf("failed to fetch URL after %d attempts: status code %d", maxRetries, resp.StatusCode)
		}

		// Read response body
		content, err = io.ReadAll(resp.Body)
		if err != nil {
			c.Logger.Warn("Failed to read response body", 
				zap.Error(err), 
				zap.String("url", url), 
				zap.Int("attempt", retries+1))
			
			retries++
			if retries < maxRetries {
				time.Sleep(retryWaitDuration)
				continue
			}
			return nil, fmt.Errorf("failed to read response body after %d attempts: %w", maxRetries, err)
		}

		c.Logger.Debug("Successfully fetched URL", 
			zap.String("url", url), 
			zap.Int("content_length", len(content)))
		
		return content, nil
	}

	return nil, fmt.Errorf("failed to fetch URL after %d attempts", maxRetries)
}

// getDefaultHeaders returns common headers for HTTP requests
func getDefaultHeaders() map[string]string {
	return map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
		"Cache-Control":   "no-cache",
		"Pragma":          "no-cache",
	}
}