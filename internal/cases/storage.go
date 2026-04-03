package cases

import (
	"context"

	"crypto_service/internal/entities"
)

//go:generate mockgen -source=storage.go -destination=mocks/storage_mock.go -package=mocks
type Storage interface {
	Store(ctx context.Context, coins []*entities.Coin) error
	GetAllTitles(ctx context.Context) ([]string, error)
	GetLastCoins(ctx context.Context, titles []string) ([]*entities.Coin, error)
	GetAggregatedCoins(ctx context.Context, titles []string, aggregationType string) ([]*entities.Coin, error)
}
