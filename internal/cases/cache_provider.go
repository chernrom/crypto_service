package cases

import (
	"context"
	"crypto_service/internal/entities"
)

type CacheProvider interface {
	Store(ctx context.Context, operationType string, key string, value any) error
	GetCoins(ctx context.Context, operationType string, titles []string) ([]*entities.Coin, error)
	Flush(ctx context.Context) error
}
