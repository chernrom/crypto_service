package cases

import (
	"context"
	"crypto_service/internal/entities"
)

type CacheProvider interface {
	GetCoins(ctx context.Context, key string) ([]*entities.Coin, error)
	SetCoins(ctx context.Context, key string, coins []*entities.Coin) error
	Invalidate(ctx context.Context) error
}
