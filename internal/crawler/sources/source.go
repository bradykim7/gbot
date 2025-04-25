package sources

import (
	"context"

	"github.com/bradykim7/gbot/internal/models"
)

// Source defines the interface for all crawler sources
type Source interface {
	// Crawl fetches and parses products from the source
	Crawl(ctx context.Context) ([]models.Product, error)
	
	// Name returns the name of the source
	Name() string
}